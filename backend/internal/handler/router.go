package handler

import (
	"net/http"
	"os"
)

// NewMux returns an HTTP mux with all routes registered.
func NewMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", HealthCheck)

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./static"
	}
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		fs := http.FileServer(http.Dir(staticDir))
		mux.Handle("/", fs)
	}

	return mux
}
