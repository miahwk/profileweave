package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWithWebDirServesSPAAndPreservesAPI(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<main>app</main>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("window.app=true"), 0o600); err != nil {
		t.Fatal(err)
	}
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("api")) })
	handler := WithWebDir(api, dir)

	for path, wanted := range map[string]string{"/": "<main>app</main>", "/profiles/edit": "<main>app</main>", "/app.js": "window.app=true", "/api/v1/health": "api"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), wanted) {
			t.Errorf("GET %s: %d %q", path, recorder.Code, recorder.Body.String())
		}
		if recorder.Header().Get("X-Frame-Options") != "DENY" {
			t.Errorf("GET %s missing static security headers", path)
		}
	}
}

func TestWithWebDirKeepsAPIWhenBuildMissing(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := WithWebDir(api, filepath.Join(t.TempDir(), "missing"))

	apiResponse := httptest.NewRecorder()
	handler.ServeHTTP(apiResponse, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if apiResponse.Code != http.StatusNoContent {
		t.Fatalf("API status %d", apiResponse.Code)
	}
	webResponse := httptest.NewRecorder()
	handler.ServeHTTP(webResponse, httptest.NewRequest(http.MethodGet, "/", nil))
	if webResponse.Code != http.StatusNotFound || !strings.Contains(webResponse.Body.String(), "frontend build unavailable") {
		t.Fatalf("web status %d: %s", webResponse.Code, webResponse.Body.String())
	}
}
