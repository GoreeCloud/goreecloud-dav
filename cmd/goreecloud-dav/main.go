package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GoreeCloud/goreecloud-dav/internal/auth"
	"github.com/GoreeCloud/goreecloud-dav/internal/config"
	"github.com/GoreeCloud/goreecloud-dav/internal/dav"
	"github.com/GoreeCloud/goreecloud-dav/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	store, err := storage.NewFS(cfg.DataDir)
	if err != nil {
		log.Fatalf("storage initialization error: %v", err)
	}

	provider := auth.DevelopmentProvider{
		Username: cfg.Username,
		Password: cfg.Password,
	}
	app := dav.New(store, provider, cfg.MaxBodyBytes)

	server := &http.Server{
		Addr:              cfg.Listen,
		Handler:           app.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("GoreeCloud DAV development service listening on %s", cfg.Listen)
		errCh <- server.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
		return
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}
}
