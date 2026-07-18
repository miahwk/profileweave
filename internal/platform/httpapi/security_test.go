package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPGuardsAndStructuredErrors(t *testing.T) {
	api := newTestAPI()
	badOrigin := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader([]byte(`{}`)))
	badOrigin.Host = "127.0.0.1:3210"
	badOrigin.Header.Set("Origin", "https://evil.example")
	badOrigin.Header.Set(controlTokenHeader, "test-control-token")
	recorder := httptest.NewRecorder()
	api.ServeHTTP(recorder, badOrigin)
	if recorder.Code != http.StatusForbidden || !bytes.Contains(recorder.Body.Bytes(), []byte(`"code":"origin_forbidden"`)) {
		t.Fatalf("origin guard response %d: %s", recorder.Code, recorder.Body.String())
	}
	badHost := httptest.NewRequest(http.MethodGet, "/api/v1/profiles", nil)
	badHost.Host = "attacker.example"
	badHostResponse := httptest.NewRecorder()
	api.ServeHTTP(badHostResponse, badHost)
	if badHostResponse.Code != http.StatusForbidden {
		t.Fatalf("host guard response %d: %s", badHostResponse.Code, badHostResponse.Body.String())
	}
	missingToken := httptest.NewRequest(http.MethodDelete, "/api/v1/profiles/p_00000000000000000000000000000000", nil)
	missingToken.Host = "127.0.0.1:3210"
	missingTokenResponse := httptest.NewRecorder()
	api.ServeHTTP(missingTokenResponse, missingToken)
	if missingTokenResponse.Code != http.StatusForbidden || !bytes.Contains(missingTokenResponse.Body.Bytes(), []byte(`"code":"control_token_invalid"`)) {
		t.Fatalf("control token response %d: %s", missingTokenResponse.Code, missingTokenResponse.Body.String())
	}

	unknown := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader([]byte(`{"unknown":true}`)))
	unknown.Host = "127.0.0.1:3210"
	unknown.Header.Set(controlTokenHeader, "test-control-token")
	badJSON := httptest.NewRecorder()
	api.ServeHTTP(badJSON, unknown)
	if badJSON.Code != http.StatusBadRequest || !bytes.Contains(badJSON.Body.Bytes(), []byte(`"error"`)) {
		t.Fatalf("JSON guard response %d: %s", badJSON.Code, badJSON.Body.String())
	}
}

func TestHealthIncludesBuildInfoAndSecurityHeaders(t *testing.T) {
	response := request(t, newTestAPI(), http.MethodGet, "/api/v1/health", nil)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"version":"dev"`)) ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"product":"ProfileWeave"`)) {
		t.Fatalf("health response %d: %s", response.Code, response.Body.String())
	}
	for name, want := range map[string]string{
		"Cache-Control":           "no-store",
		"Content-Security-Policy": "frame-ancestors 'none'",
		"Referrer-Policy":         "no-referrer",
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
	} {
		if got := response.Header().Get(name); !bytes.Contains([]byte(got), []byte(want)) {
			t.Errorf("%s=%q, want containing %q", name, got, want)
		}
	}
}

func TestShutdownRequiresTokenAndInvokesCallback(t *testing.T) {
	api := newTestAPI()
	called := make(chan struct{}, 1)
	api.shutdown = func() { called <- struct{}{} }

	req := httptest.NewRequest(http.MethodPost, "/api/v1/shutdown", nil)
	req.Host = "127.0.0.1:3210"
	denied := httptest.NewRecorder()
	api.ServeHTTP(denied, req)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("missing token status = %d", denied.Code)
	}
	select {
	case <-called:
		t.Fatal("unauthorized request invoked shutdown")
	default:
	}

	accepted := request(t, api, http.MethodPost, "/api/v1/shutdown", nil)
	if accepted.Code != http.StatusAccepted {
		t.Fatalf("shutdown status %d: %s", accepted.Code, accepted.Body.String())
	}
	select {
	case <-called:
	default:
		t.Fatal("authorized request did not invoke shutdown")
	}
	second := request(t, api, http.MethodPost, "/api/v1/shutdown", nil)
	if second.Code != http.StatusAccepted {
		t.Fatalf("second shutdown status %d", second.Code)
	}
	select {
	case <-called:
	default:
		t.Fatal("a later shutdown request did not allow cleanup retry")
	}
}

func TestBootstrapReturnsEphemeralControlToken(t *testing.T) {
	response := request(t, newTestAPI(), http.MethodGet, "/api/v1/bootstrap", nil)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"controlToken":"test-control-token"`)) {
		t.Fatalf("bootstrap response %d: %s", response.Code, response.Body.String())
	}
}
