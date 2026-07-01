package main

import (
	"log"
	"net/http"
	"time"

	"github.com/dereknola/FogCast/internal/config"
	"github.com/dereknola/FogCast/internal/session"
	"github.com/dereknola/FogCast/internal/web"
)

func main() {
	cfg := config.Load()

	activeMap, err := session.LoadActiveMap(cfg.DataDir)
	if err != nil {
		log.Printf("warning: could not load persisted state: %v", err)
	}

	manager := session.NewManager(activeMap)

	persistedMask, err := session.LoadMask(cfg.DataDir, manager.MaskLength())
	if err != nil {
		log.Printf("warning: could not load persisted mask snapshot: %v", err)
	} else if len(persistedMask) > 0 {
		_ = manager.SetMask(persistedMask)
	}

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           web.NewServer(cfg, manager).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("FogCast listening on %s", cfg.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
