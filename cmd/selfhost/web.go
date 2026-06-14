package main

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// mountWebUI serves the bundled static web SPA from dir as a fallback for any
// request not handled by an API or health route. The container image places the
// build at /app/web (WEB_DIR). If dir does not exist — e.g. the binary is run
// outside that image — this is a no-op and the binary stays API-only.
func mountWebUI(r chi.Router, dir string) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		slog.Info("web UI directory not found; serving API only", slog.String("dir", dir))
		return
	}

	index := filepath.Join(dir, "index.html")
	fileServer := http.FileServer(http.Dir(dir))

	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		// API and health namespaces own their own 404s — never shadow them with
		// the SPA shell, which would return HTML for a missing JSON endpoint.
		if strings.HasPrefix(req.URL.Path, "/api/") || strings.HasPrefix(req.URL.Path, "/health/") {
			http.NotFound(w, req)
			return
		}

		// Serve the requested asset if it exists; otherwise fall back to
		// index.html so client-side routes (e.g. /reading) resolve to the SPA.
		clean := filepath.Clean(req.URL.Path)
		if clean != "/" {
			if _, statErr := os.Stat(filepath.Join(dir, clean)); statErr == nil {
				fileServer.ServeHTTP(w, req)
				return
			}
		}
		http.ServeFile(w, req, index)
	})

	slog.Info("serving bundled web UI", slog.String("dir", dir))
}
