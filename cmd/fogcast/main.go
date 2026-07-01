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
	manager := session.NewManager()

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
