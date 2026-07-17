package httpapi

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// WithWebDir serves a built Vue SPA without coupling the Go build to a generated
// dist directory. API routes always take precedence and remain available when
// the frontend build is absent.
func WithWebDir(api http.Handler, webDir string) http.Handler {
	abs, err := filepath.Abs(webDir)
	available := err == nil && directoryExists(abs) && regularFile(filepath.Join(abs, "index.html"))
	files := http.FileServer(http.Dir(abs))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			api.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !available {
			http.Error(w, "frontend build unavailable; build frontend/dist or configure PROFILEWEAVE_WEB_DIR", http.StatusNotFound)
			return
		}
		cleanURL := filepath.ToSlash(filepath.Clean(r.URL.Path))
		if !strings.HasPrefix(cleanURL, "/") {
			cleanURL = "/" + cleanURL
		}
		rel := strings.TrimPrefix(cleanURL, "/")
		target := filepath.Join(abs, filepath.FromSlash(rel))
		if withinRoot(abs, target) && regularFile(target) {
			clone := r.Clone(r.Context())
			clone.URL.Path = cleanURL
			files.ServeHTTP(w, clone)
			return
		}
		if filepath.Ext(rel) != "" {
			http.NotFound(w, r)
			return
		}
		serveIndex(w, r, filepath.Join(abs, "index.html"))
	})
	return securityHeaders(handler)
}

func serveIndex(w http.ResponseWriter, r *http.Request, indexPath string) {
	file, err := os.Open(indexPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, "index.html", info.ModTime(), file)
}

func withinRoot(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && !errors.Is(err, os.ErrNotExist)
}
