package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	browserapp "github.com/miahwk/profileweave/internal/browser/application"
)

func TestWriteErrorMapsApplicationShutdown(t *testing.T) {
	response := httptest.NewRecorder()
	writeError(response, browserapp.ErrShuttingDown)
	if response.Code != http.StatusConflict ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"code":"application_shutting_down"`)) {
		t.Fatalf("shutdown error response %d: %s", response.Code, response.Body.String())
	}
}
