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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dereknola/FogCast/internal/config"
	"github.com/dereknola/FogCast/internal/session"
	"github.com/gorilla/websocket"
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

func TestMapUploadReplacesPreviousAsset(t *testing.T) {
	dataDir := t.TempDir()
	server := NewServer(config.Config{
		StaticDir:      "static",
		DataDir:        dataDir,
		MaxUploadBytes: 5 * 1024 * 1024,
	}, session.NewManager(nil))

	first := uploadMap(t, server)
	firstPath := filepath.Join(dataDir, "maps", first.ID+".webp")
	if _, err := os.Stat(firstPath); err != nil {
		t.Fatalf("expected first uploaded map file at %q: %v", firstPath, err)
	}

	second := uploadMap(t, server)
	if second.ID == first.ID {
		t.Fatalf("expected replacement map to use a new id")
	}

	if _, err := os.Stat(firstPath); !os.IsNotExist(err) {
		t.Fatalf("expected first map asset to be removed, got err=%v", err)
	}

	oldAssetReq := httptest.NewRequest(http.MethodGet, first.URL, nil)
	oldAssetRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(oldAssetRes, oldAssetReq)
	if oldAssetRes.Code != http.StatusNotFound {
		t.Fatalf("expected status %d for replaced asset, got %d", http.StatusNotFound, oldAssetRes.Code)
	}
}

func TestMapUploadPersistsActiveMapMetadata(t *testing.T) {
	dataDir := t.TempDir()
	server := NewServer(config.Config{
		StaticDir:      "static",
		DataDir:        dataDir,
		MaxUploadBytes: 5 * 1024 * 1024,
	}, session.NewManager(nil))

	uploaded := uploadMap(t, server)

	persisted, err := session.LoadActiveMap(dataDir)
	if err != nil {
		t.Fatalf("load persisted active map: %v", err)
	}
	if persisted == nil {
		t.Fatalf("expected persisted active map to be present")
	}

	if *persisted != uploaded {
		t.Fatalf("persisted map mismatch: expected %+v, got %+v", uploaded, *persisted)
	}
}

func TestMapUploadShroudsMaskByDefault(t *testing.T) {
	dataDir := t.TempDir()
	manager := session.NewManager(nil)
	if ok := manager.RevealAll(); len(ok) == 0 {
		t.Fatalf("expected reveal all to produce mask data")
	}

	server := NewServer(config.Config{
		StaticDir:      "static",
		DataDir:        dataDir,
		MaxUploadBytes: 5 * 1024 * 1024,
	}, manager)

	uploadMap(t, server)

	mask := manager.MaskCopy()
	for i, value := range mask {
		if value != 0 {
			t.Fatalf("expected mask to be fully shrouded after upload; index %d has %d", i, value)
		}
	}
}

func TestMapUploadCanDisableAutoShroudAll(t *testing.T) {
	dataDir := t.TempDir()
	manager := session.NewManager(nil)
	revealedMask := manager.RevealAll()

	server := NewServer(config.Config{
		StaticDir:      "static",
		DataDir:        dataDir,
		MaxUploadBytes: 5 * 1024 * 1024,
	}, manager)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	filePart, err := writer.CreateFormFile("map", "battlemap.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	if err := png.Encode(filePart, sampleImage()); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	if err := writer.WriteField("autoShroudAll", "false"); err != nil {
		t.Fatalf("write autoShroudAll field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/map", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res := httptest.NewRecorder()

	server.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, res.Code)
	}

	mask := manager.MaskCopy()
	if !bytes.Equal(mask, revealedMask) {
		t.Fatalf("expected mask to remain unchanged when autoShroudAll=false")
	}
}

func TestManualPushStagesMapUntilPush(t *testing.T) {
	dataDir := t.TempDir()
	server := NewServer(config.Config{
		StaticDir:      "static",
		DataDir:        dataDir,
		MaxUploadBytes: 5 * 1024 * 1024,
	}, session.NewManager(nil))

	first := uploadMap(t, server)
	second := uploadMapWithFields(t, server, map[string]string{"autoSync": "false"})

	if first.ID == second.ID {
		t.Fatalf("expected second upload to generate a new map id")
	}

	dmState := getState(t, server, "/api/state")
	if dmState.ActiveMap == nil || dmState.ActiveMap.ID != second.ID {
		t.Fatalf("expected DM state to show staged map %q, got %+v", second.ID, dmState.ActiveMap)
	}

	playerStateBeforePush := getState(t, server, "/api/player/state")
	if playerStateBeforePush.ActiveMap == nil || playerStateBeforePush.ActiveMap.ID != first.ID {
		t.Fatalf("expected player state to remain on published map %q before push, got %+v", first.ID, playerStateBeforePush.ActiveMap)
	}

	pushReq := httptest.NewRequest(http.MethodPost, "/api/push", nil)
	pushRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(pushRes, pushReq)
	if pushRes.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, pushRes.Code)
	}

	playerStateAfterPush := getState(t, server, "/api/player/state")
	if playerStateAfterPush.ActiveMap == nil || playerStateAfterPush.ActiveMap.ID != second.ID {
		t.Fatalf("expected player state to switch to map %q after push, got %+v", second.ID, playerStateAfterPush.ActiveMap)
	}
}

func TestPlayerWSReceivesInitialMask(t *testing.T) {
	server := NewServer(config.Config{StaticDir: "static"}, session.NewManager(nil))
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	playerConn, _, err := websocket.DefaultDialer.Dial(wsURL(httpServer.URL, "/ws/player"), nil)
	if err != nil {
		t.Fatalf("dial player ws: %v", err)
	}
	defer playerConn.Close()

	if err := playerConn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	messageType, payload, err := playerConn.ReadMessage()
	if err != nil {
		t.Fatalf("read initial player message: %v", err)
	}
	if messageType != websocket.BinaryMessage {
		t.Fatalf("expected binary message, got %d", messageType)
	}
	if len(payload) != 512*512 {
		t.Fatalf("expected mask payload length %d, got %d", 512*512, len(payload))
	}

	for i, value := range payload {
		if value != 0 {
			t.Fatalf("expected initial mask value 0 at index %d, got %d", i, value)
		}
	}
}

func TestDMWSBroadcastsMaskToPlayers(t *testing.T) {
	server := NewServer(config.Config{StaticDir: "static"}, session.NewManager(nil))
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	playerConn, _, err := websocket.DefaultDialer.Dial(wsURL(httpServer.URL, "/ws/player"), nil)
	if err != nil {
		t.Fatalf("dial player ws: %v", err)
	}
	defer playerConn.Close()

	if err := playerConn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set player read deadline: %v", err)
	}
	if _, _, err := playerConn.ReadMessage(); err != nil {
		t.Fatalf("read initial player message: %v", err)
	}

	dmConn, _, err := websocket.DefaultDialer.Dial(wsURL(httpServer.URL, "/ws/dm"), nil)
	if err != nil {
		t.Fatalf("dial dm ws: %v", err)
	}
	defer dmConn.Close()

	updatedMask := bytes.Repeat([]byte{255}, 512*512)
	if err := dmConn.WriteMessage(websocket.BinaryMessage, updatedMask); err != nil {
		t.Fatalf("send dm mask update: %v", err)
	}

	if err := playerConn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set player read deadline: %v", err)
	}

	messageType, payload, err := playerConn.ReadMessage()
	if err != nil {
		t.Fatalf("read player broadcast: %v", err)
	}
	if messageType != websocket.BinaryMessage {
		t.Fatalf("expected binary broadcast, got %d", messageType)
	}
	if !bytes.Equal(payload, updatedMask) {
		t.Fatalf("expected broadcast payload to match dm payload")
	}
}

func TestDMWSReceivesInitialMaskOnConnect(t *testing.T) {
	manager := session.NewManager(nil)
	expectedMask := bytes.Repeat([]byte{200}, 512*512)
	if ok := manager.SetMask(expectedMask); !ok {
		t.Fatalf("expected mask update to succeed")
	}

	server := NewServer(config.Config{StaticDir: "static"}, manager)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	dmConn, _, err := websocket.DefaultDialer.Dial(wsURL(httpServer.URL, "/ws/dm"), nil)
	if err != nil {
		t.Fatalf("dial dm ws: %v", err)
	}
	defer dmConn.Close()

	if err := dmConn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set dm read deadline: %v", err)
	}

	messageType, payload, err := dmConn.ReadMessage()
	if err != nil {
		t.Fatalf("read initial dm message: %v", err)
	}
	if messageType != websocket.BinaryMessage {
		t.Fatalf("expected binary message, got %d", messageType)
	}
	if !bytes.Equal(payload, expectedMask) {
		t.Fatalf("expected initial dm mask to match server state")
	}
}

func uploadMap(t *testing.T, server *Server) session.MapState {
	t.Helper()
	return uploadMapWithFields(t, server, nil)
}

func uploadMapWithFields(t *testing.T, server *Server, fields map[string]string) session.MapState {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	filePart, err := writer.CreateFormFile("map", "battlemap.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	if err := png.Encode(filePart, sampleImage()); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field %q: %v", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/map", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res := httptest.NewRecorder()

	server.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, res.Code)
	}

	var uploaded session.MapState
	if err := json.NewDecoder(res.Body).Decode(&uploaded); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}

	return uploaded
}

func getState(t *testing.T, server *Server, path string) session.State {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d for %s, got %d", http.StatusOK, path, res.Code)
	}

	var state session.State
	if err := json.NewDecoder(res.Body).Decode(&state); err != nil {
		t.Fatalf("decode state from %s: %v", path, err)
	}

	return state
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

func wsURL(httpURL, path string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http") + path
}
