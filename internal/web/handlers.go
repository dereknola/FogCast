package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dereknola/FogCast/internal/assets"
	"github.com/dereknola/FogCast/internal/config"
	"github.com/dereknola/FogCast/internal/session"
)

type Server struct {
	cfg     config.Config
	session *session.Manager
	mux     *http.ServeMux
	hub     *maskHub
}

func NewServer(cfg config.Config, manager *session.Manager) *Server {
	server := &Server{
		cfg:     cfg,
		session: manager,
		mux:     http.NewServeMux(),
		hub:     newMaskHub(),
	}
	server.routes()

	return server
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("/", s.handleRoot)
	s.mux.HandleFunc("/api/state", s.handleState)
	s.mux.HandleFunc("/api/map", s.handleMapUpload)
	s.mux.HandleFunc("/ws/dm", s.handleDMWS)
	s.mux.HandleFunc("/ws/player", s.handlePlayerWS)
	s.mux.HandleFunc("/assets/maps/", s.handleMapAsset)
	s.mux.Handle("/dm/", http.StripPrefix("/dm/", s.staticHandler("dm")))
	s.mux.HandleFunc("/dm", s.handleApp("dm"))
	s.mux.Handle("/player/", http.StripPrefix("/player/", s.staticHandler("player")))
	s.mux.HandleFunc("/player", s.handleApp("player"))
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := landingTemplate.Execute(w, nil); err != nil {
		http.Error(w, "render landing page", http.StatusInternalServerError)
	}
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.session.State()); err != nil {
		http.Error(w, "encode state", http.StatusInternalServerError)
	}
}

func (s *Server) handleMapUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadBytes+1024)
	if err := r.ParseMultipartForm(s.cfg.MaxUploadBytes + 1024); err != nil {
		http.Error(w, "invalid upload payload", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("map")
	if err != nil {
		http.Error(w, "missing map file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	result, err := assets.OptimizeUploadedMap(s.cfg.DataDir, header.Filename, file, s.cfg.MaxUploadBytes)
	if err != nil {
		switch {
		case errors.Is(err, assets.ErrFileTooLarge):
			http.Error(w, "map file too large", http.StatusRequestEntityTooLarge)
		case errors.Is(err, assets.ErrUnsupportedType), errors.Is(err, assets.ErrInvalidDimensions):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "failed to optimize map", http.StatusInternalServerError)
		}
		return
	}

	previous := s.session.State().ActiveMap

	activeMap := &session.MapState{
		ID:     result.ID,
		Name:   result.Name,
		Width:  result.Width,
		Height: result.Height,
		URL:    fmt.Sprintf("/assets/maps/%s.webp", result.ID),
	}

	s.session.SetActiveMap(activeMap)
	if err := session.SaveActiveMap(s.cfg.DataDir, activeMap); err != nil {
		http.Error(w, "failed to persist active map", http.StatusInternalServerError)
		return
	}

	if previous != nil && previous.ID != activeMap.ID {
		_ = os.Remove(filepath.Join(s.cfg.DataDir, "maps", previous.ID+".webp"))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(activeMap); err != nil {
		http.Error(w, "encode map response", http.StatusInternalServerError)
	}
}

func (s *Server) handleMapAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	fileName := strings.TrimPrefix(r.URL.Path, "/assets/maps/")
	if fileName == "" || strings.Contains(fileName, "/") || !strings.HasSuffix(fileName, ".webp") {
		http.NotFound(w, r)
		return
	}

	activeMap := s.session.State().ActiveMap
	if activeMap == nil || fileName != activeMap.ID+".webp" {
		http.NotFound(w, r)
		return
	}

	assetPath := filepath.Join(s.cfg.DataDir, "maps", fileName)
	if _, err := os.Stat(assetPath); err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "image/webp")
	http.ServeFile(w, r, assetPath)
}

func (s *Server) handleApp(app string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}

		indexPath := filepath.Join(s.cfg.StaticDir, app, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			http.ServeFile(w, r, indexPath)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := placeholderTemplate.Execute(w, app); err != nil {
			http.Error(w, fmt.Sprintf("render %s placeholder", app), http.StatusInternalServerError)
		}
	}
}

func (s *Server) staticHandler(app string) http.Handler {
	return http.FileServer(http.Dir(filepath.Join(s.cfg.StaticDir, app)))
}

func methodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Allow", allowed)
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

var landingTemplate = template.Must(template.New("landing").Parse(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>FogCast</title>
    <style>
      body { margin: 0; min-height: 100vh; display: grid; place-items: center; font-family: system-ui, sans-serif; background: #10141f; color: #f5f7fb; }
			main { width: min(36rem, calc(100vw - 2rem)); text-align: center; }
			.brand { width: min(14rem, 42vw); height: auto; display: block; margin: 0 auto 1rem; }
      a { color: #8cc8ff; }
			.links { display: flex; gap: 1rem; flex-wrap: wrap; justify-content: center; }
    </style>
  </head>
  <body>
    <main>
			<img class="brand" src="/dm/fog_cast.webp" alt="FogCast logo">
      <p>Local-network battlemap fog of war.</p>
      <p class="links"><a href="/dm">Open DM controls</a><a href="/player">Open player display</a></p>
    </main>
  </body>
</html>`))

var placeholderTemplate = template.Must(template.New("placeholder").Parse(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>FogCast {{ . }}</title>
    <style>
      body { margin: 0; min-height: 100vh; display: grid; place-items: center; font-family: system-ui, sans-serif; background: #10141f; color: #f5f7fb; }
      main { width: min(36rem, calc(100vw - 2rem)); }
      code { background: #20283a; padding: 0.15rem 0.35rem; border-radius: 0.25rem; }
      a { color: #8cc8ff; }
    </style>
  </head>
  <body>
    <main>
      <h1>FogCast {{ . }}</h1>
      <p>The {{ . }} frontend has not been built yet.</p>
      <p>Run <code>scripts/build.sh</code>, then reload this page.</p>
      <p><a href="/api/state">View server state</a></p>
    </main>
  </body>
</html>`))
