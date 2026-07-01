package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dereknola/FogCast/internal/config"
	"github.com/dereknola/FogCast/internal/session"
)

type Server struct {
	cfg     config.Config
	session *session.Manager
	mux     *http.ServeMux
}

func NewServer(cfg config.Config, manager *session.Manager) *Server {
	server := &Server{
		cfg:     cfg,
		session: manager,
		mux:     http.NewServeMux(),
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
      main { width: min(36rem, calc(100vw - 2rem)); }
      a { color: #8cc8ff; }
      .links { display: flex; gap: 1rem; flex-wrap: wrap; }
    </style>
  </head>
  <body>
    <main>
      <h1>FogCast</h1>
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
