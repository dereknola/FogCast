package web

import (
	"encoding/binary"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/dereknola/FogCast/internal/session"
	"github.com/gorilla/websocket"
)

const (
	wsWriteTimeout            = 5 * time.Second
	maskPatchMessageType byte = 1
)

type maskHub struct {
	mu      sync.Mutex
	players map[*websocket.Conn]struct{}
}

func newMaskHub() *maskHub {
	return &maskHub{players: make(map[*websocket.Conn]struct{})}
}

func (h *maskHub) addPlayer(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.players[conn] = struct{}{}
}

func (h *maskHub) removePlayer(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.players, conn)
}

func (h *maskHub) broadcast(mask []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.players {
		if err := conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout)); err != nil {
			conn.Close()
			delete(h.players, conn)
			continue
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, mask); err != nil {
			conn.Close()
			delete(h.players, conn)
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type dmControlMessage struct {
	Type string `json:"type"`
}

func (s *Server) handlePlayerWS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.hub.addPlayer(conn)

	initialMask := s.session.MaskCopy()
	if err := conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout)); err == nil {
		if err := conn.WriteMessage(websocket.BinaryMessage, initialMask); err != nil {
			s.hub.removePlayer(conn)
			conn.Close()
			return
		}
	}

	go func() {
		defer func() {
			s.hub.removePlayer(conn)
			conn.Close()
		}()

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

func (s *Server) handleDMWS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	initialMask := s.session.MaskCopy()
	if err := conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout)); err == nil {
		if err := conn.WriteMessage(websocket.BinaryMessage, initialMask); err != nil {
			return
		}
	}

	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}

		switch messageType {
		case websocket.BinaryMessage:
			if ok := s.session.SetMask(payload); ok {
				_ = session.SaveMask(s.cfg.DataDir, payload)
				s.hub.broadcast(payload)
				continue
			}

			patch, ok := decodeMaskPatchMessage(payload)
			if !ok {
				continue
			}
			if ok := s.session.ApplyMaskPatch(patch); !ok {
				continue
			}
			_ = session.SaveMask(s.cfg.DataDir, s.session.MaskCopy())
			s.hub.broadcast(payload)
		case websocket.TextMessage:
			var msg dmControlMessage
			if err := json.Unmarshal(payload, &msg); err != nil {
				continue
			}

			switch msg.Type {
			case "reveal_all":
				mask := s.session.RevealAll()
				_ = session.SaveMask(s.cfg.DataDir, mask)
				s.hub.broadcast(mask)
			case "shroud_all":
				mask := s.session.ShroudAll()
				_ = session.SaveMask(s.cfg.DataDir, mask)
				s.hub.broadcast(mask)
			}
		}
	}
}

func decodeMaskPatchMessage(payload []byte) (session.MaskPatch, bool) {
	const headerSize = 1 + 4 + 4 + 4 + 4
	if len(payload) < headerSize {
		return session.MaskPatch{}, false
	}

	if payload[0] != maskPatchMessageType {
		return session.MaskPatch{}, false
	}

	x := int(binary.LittleEndian.Uint32(payload[1:5]))
	y := int(binary.LittleEndian.Uint32(payload[5:9]))
	width := int(binary.LittleEndian.Uint32(payload[9:13]))
	height := int(binary.LittleEndian.Uint32(payload[13:17]))
	if width <= 0 || height <= 0 {
		return session.MaskPatch{}, false
	}

	expectedLength := headerSize + (width * height)
	if len(payload) != expectedLength {
		return session.MaskPatch{}, false
	}

	return session.MaskPatch{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
		Data:   payload[headerSize:],
	}, true
}
