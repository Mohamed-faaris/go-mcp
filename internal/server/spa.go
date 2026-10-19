package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// spaHandler serves static assets from dir, falling back to index.html for client-side routing.
func (s *Server) spaHandler(distDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(distDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ignore API routes
		if strings.HasPrefix(r.URL.Path, "/api") {
			http.NotFound(w, r)
			return
		}

		cleanPath := filepath.Clean(r.URL.Path)
		targetPath := filepath.Join(distDir, cleanPath)

		info, err := os.Stat(targetPath)
		if err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fallback to index.html for SPA router
		indexPath := filepath.Join(distDir, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			http.ServeFile(w, r, indexPath)
			return
		}

		// Fallback minimal HTML if dist directory has not been built yet
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>go-mcp Web UI</title><meta charset="utf-8"/></head>
<body style="font-family: sans-serif; background: #0f172a; color: #f8fafc; padding: 2rem;">
  <h2>go-mcp Podman Control Center</h2>
  <p>Backend API is active. Frontend build is loading or can be started via <code>cd web && bun run dev</code>.</p>
</body>
</html>`))
	})
}
