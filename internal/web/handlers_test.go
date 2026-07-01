package web

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dereknola/FogCast/internal/config"
	"github.com/dereknola/FogCast/internal/session"
)

func TestStateEndpoint(t *testing.T) {
	server := NewServer(config.Config{StaticDir: "static"}, session.NewManager(nil))

	request := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var state session.State
	if err := json.NewDecoder(response.Body).Decode(&state); err != nil {
		t.Fatalf("decode state: %v", err)
	}

	if state.Mask.Width != 512 || state.Mask.Height != 512 {
		t.Fatalf("expected default 512x512 mask, got %dx%d", state.Mask.Width, state.Mask.Height)
	}
}

func TestRootEndpoint(t *testing.T) {
	server := NewServer(config.Config{StaticDir: "static"}, session.NewManager(nil))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}

func TestMapUploadAndAssetServing(t *testing.T) {
	dataDir := t.TempDir()
	server := NewServer(config.Config{
		StaticDir:      "static",
		DataDir:        dataDir,
		MaxUploadBytes: 5 * 1024 * 1024,
	}, session.NewManager(nil))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	filePart, err := writer.CreateFormFile("map", "battlemap.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	if err := png.Encode(filePart, sampleImage()); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	uploadReq := httptest.NewRequest(http.MethodPost, "/api/map", body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadRes := httptest.NewRecorder()

	server.Handler().ServeHTTP(uploadRes, uploadReq)

	if uploadRes.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, uploadRes.Code)
	}

	var uploaded session.MapState
	if err := json.NewDecoder(uploadRes.Body).Decode(&uploaded); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}

	if uploaded.ID == "" || uploaded.URL == "" {
		t.Fatalf("expected populated map id and url, got %+v", uploaded)
	}

	stateReq := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	stateRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(stateRes, stateReq)

	if stateRes.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, stateRes.Code)
	}

	var state session.State
	if err := json.NewDecoder(stateRes.Body).Decode(&state); err != nil {
		t.Fatalf("decode state response: %v", err)
	}

	if state.ActiveMap == nil {
		t.Fatalf("expected active map to be set")
	}

	assetReq := httptest.NewRequest(http.MethodGet, state.ActiveMap.URL, nil)
	assetRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(assetRes, assetReq)

	if assetRes.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, assetRes.Code)
	}
	if ct := assetRes.Header().Get("Content-Type"); ct != "image/webp" {
		t.Fatalf("expected content type image/webp, got %q", ct)
	}
}

func TestMapUploadRejectsInvalidType(t *testing.T) {
	server := NewServer(config.Config{
		StaticDir:      "static",
		DataDir:        t.TempDir(),
		MaxUploadBytes: 1024 * 1024,
	}, session.NewManager(nil))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	filePart, err := writer.CreateFormFile("map", "notes.txt")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	if _, err := filePart.Write([]byte("this is not an image")); err != nil {
		t.Fatalf("write file part: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/map", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res := httptest.NewRecorder()

	server.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}
}

func sampleImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 8), G: uint8(y * 8), B: 120, A: 255})
		}
	}

	return img
}
