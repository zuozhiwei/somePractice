package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"skilltree-go/internal/api"
	"skilltree-go/internal/frontend"
	"skilltree-go/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	key, err := base64.StdEncoding.DecodeString(os.Getenv("SKILLTREE_JWT_SECRET"))
	if err != nil || len(key) < 32 {
		return fmt.Errorf("SKILLTREE_JWT_SECRET must be a base64-encoded key of at least 32 bytes")
	}
	if zone := os.Getenv("SKILLTREE_TIMEZONE"); zone != "" {
		if _, err := time.LoadLocation(zone); err != nil {
			return fmt.Errorf("invalid SKILLTREE_TIMEZONE: %w", err)
		}
	}
	path := env("SKILLTREE_DB", "data/skilltree.db")
	db, err := openDB(path)
	if err != nil {
		return err
	}
	defer db.Close()
	server := &http.Server{Addr: env("SKILLTREE_LISTEN", "127.0.0.1:8081"), Handler: handler(db), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	log.Printf("listening on %s", server.Addr)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func openDB(path string) (*sql.DB, error) {
	db, err := store.Open(path)
	if err != nil {
		return nil, err
	}
	if err := store.Migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func handler(db *sql.DB) http.Handler {
	mux := http.NewServeMux()
	api.New(db).Register(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	// Unknown API calls must never fall through to a future SPA handler.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "API not implemented", http.StatusNotImplemented)
	})
	mux.Handle("/", frontend.Handler())
	return mux
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
