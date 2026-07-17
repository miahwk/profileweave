package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	browserapp "github.com/miahwk/profileweave/internal/browser/application"
	browserdomain "github.com/miahwk/profileweave/internal/browser/domain"
	fingerprint "github.com/miahwk/profileweave/internal/fingerprint/domain"
	profileapp "github.com/miahwk/profileweave/internal/profile/application"
	profiledomain "github.com/miahwk/profileweave/internal/profile/domain"
)

type memoryRepository struct {
	mu    sync.Mutex
	items map[string]profiledomain.Profile
}

func (m *memoryRepository) List(context.Context) ([]profiledomain.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	items := make([]profiledomain.Profile, 0, len(m.items))
	for _, item := range m.items {
		items = append(items, item)
	}
	return items, nil
}

func (m *memoryRepository) Get(_ context.Context, id string) (profiledomain.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.items[id]
	if !ok {
		return profiledomain.Profile{}, profiledomain.ErrNotFound
	}
	return item, nil
}

func (m *memoryRepository) Save(_ context.Context, item profiledomain.Profile, expected uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.items[item.ID]
	if ok && current.Revision != expected || !ok && expected != 0 {
		return profiledomain.ErrConflict
	}
	m.items[item.ID] = item
	return nil
}

func (m *memoryRepository) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return profiledomain.ErrNotFound
	}
	delete(m.items, id)
	return nil
}

type fakeRuntime struct {
	mu      sync.Mutex
	started map[string]chan error
}

func (f *fakeRuntime) Discover(context.Context) ([]browserdomain.BrowserDescriptor, error) {
	return []browserdomain.BrowserDescriptor{{ID: "chrome", Name: "Chrome", Path: "C:/Chrome/chrome.exe", Available: true}}, nil
}

func (f *fakeRuntime) Launch(_ context.Context, spec browserdomain.LaunchSpec) (browserdomain.Process, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.started[spec.ProfileID]; exists {
		return browserdomain.Process{}, errors.New("already started")
	}
	done := make(chan error, 1)
	f.started[spec.ProfileID] = done
	return browserdomain.Process{PID: 4321, Done: done}, nil
}

func (f *fakeRuntime) Stop(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	done, ok := f.started[id]
	if !ok {
		return errors.New("not owned")
	}
	delete(f.started, id)
	done <- nil
	close(done)
	return nil
}

func newTestAPI() *API {
	repo := &memoryRepository{items: make(map[string]profiledomain.Profile)}
	runtime := &fakeRuntime{started: make(map[string]chan error)}
	browsers := browserapp.NewService(repo, runtime)
	profiles := profileapp.NewService(repo, browsers, nil)
	return newAPI(profiles, browsers, "test-control-token")
}

func testInput() profiledomain.Input {
	return profiledomain.Input{
		Name: "API profile", StartURL: "https://example.test", Browser: profiledomain.Browser{Kind: "auto"},
		Fingerprint: fingerprint.Default(), Proxy: fingerprint.Proxy{Mode: "direct"},
	}
}

func request(t *testing.T, api *API, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Host = "127.0.0.1:3210"
	req.Header.Set("Content-Type", "application/json")
	if method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions {
		req.Header.Set(controlTokenHeader, "test-control-token")
	}
	recorder := httptest.NewRecorder()
	api.ServeHTTP(recorder, req)
	return recorder
}

func createProfile(t *testing.T, api *API) profiledomain.Profile {
	t.Helper()
	response := request(t, api, http.MethodPost, "/api/v1/profiles", testInput())
	if response.Code != http.StatusCreated {
		t.Fatalf("create status %d: %s", response.Code, response.Body.String())
	}
	var profile profiledomain.Profile
	if err := json.Unmarshal(response.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	return profile
}

func TestProfileCRUDDuplicateAndValidate(t *testing.T) {
	api := newTestAPI()
	profile := createProfile(t, api)
	if !profiledomain.ValidID(profile.ID) || profile.Revision != 1 {
		t.Fatalf("unexpected created profile %#v", profile)
	}

	list := request(t, api, http.MethodGet, "/api/v1/profiles?search=api", nil)
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte(profile.ID)) {
		t.Fatalf("list response %d: %s", list.Code, list.Body.String())
	}
	validate := request(t, api, http.MethodPost, "/api/v1/profiles/"+profile.ID+"/validate", nil)
	if validate.Code != http.StatusOK || !bytes.Contains(validate.Body.Bytes(), []byte(`"score"`)) {
		t.Fatalf("validate response %d: %s", validate.Code, validate.Body.String())
	}
	duplicate := request(t, api, http.MethodPost, "/api/v1/profiles/"+profile.ID+"/duplicate", nil)
	if duplicate.Code != http.StatusCreated || !bytes.Contains(duplicate.Body.Bytes(), []byte("copy")) {
		t.Fatalf("duplicate response %d: %s", duplicate.Code, duplicate.Body.String())
	}

	profile.Name = "Updated"
	update := request(t, api, http.MethodPut, "/api/v1/profiles/"+profile.ID, profile)
	if update.Code != http.StatusOK || !bytes.Contains(update.Body.Bytes(), []byte(`"revision":2`)) {
		t.Fatalf("update response %d: %s", update.Code, update.Body.String())
	}
	stale := request(t, api, http.MethodPut, "/api/v1/profiles/"+profile.ID, profile)
	if stale.Code != http.StatusConflict {
		t.Fatalf("expected stale conflict, got %d: %s", stale.Code, stale.Body.String())
	}
	deleted := request(t, api, http.MethodDelete, "/api/v1/profiles/"+profile.ID, nil)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete response %d: %s", deleted.Code, deleted.Body.String())
	}
}

func TestSessionLifecycleAndCapabilities(t *testing.T) {
	api := newTestAPI()
	profile := createProfile(t, api)
	launch := request(t, api, http.MethodPost, "/api/v1/profiles/"+profile.ID+"/launch", nil)
	if launch.Code != http.StatusOK || !bytes.Contains(launch.Body.Bytes(), []byte(`"pid":4321`)) {
		t.Fatalf("launch response %d: %s", launch.Code, launch.Body.String())
	}
	conflict := request(t, api, http.MethodPost, "/api/v1/profiles/"+profile.ID+"/launch", nil)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("expected session conflict, got %d", conflict.Code)
	}
	profile.Name = "Unsafe while running"
	update := request(t, api, http.MethodPut, "/api/v1/profiles/"+profile.ID, profile)
	if update.Code != http.StatusConflict || !bytes.Contains(update.Body.Bytes(), []byte(`"code":"profile_running"`)) {
		t.Fatalf("running update response %d: %s", update.Code, update.Body.String())
	}
	deleted := request(t, api, http.MethodDelete, "/api/v1/profiles/"+profile.ID, nil)
	if deleted.Code != http.StatusConflict {
		t.Fatalf("running delete response %d: %s", deleted.Code, deleted.Body.String())
	}
	sessions := request(t, api, http.MethodGet, "/api/v1/sessions", nil)
	if sessions.Code != http.StatusOK || !bytes.Contains(sessions.Body.Bytes(), []byte(`"status":"running"`)) {
		t.Fatalf("sessions response %d: %s", sessions.Code, sessions.Body.String())
	}
	capabilities := request(t, api, http.MethodGet, "/api/v1/capabilities", nil)
	if capabilities.Code != http.StatusOK || !bytes.Contains(capabilities.Body.Bytes(), []byte(`"unsupported"`)) {
		t.Fatalf("capabilities response %d: %s", capabilities.Code, capabilities.Body.String())
	}
	stop := request(t, api, http.MethodPost, "/api/v1/profiles/"+profile.ID+"/stop", nil)
	if stop.Code != http.StatusOK || !bytes.Contains(stop.Body.Bytes(), []byte(`"status":"stopped"`)) {
		t.Fatalf("stop response %d: %s", stop.Code, stop.Body.String())
	}
}

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
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"version":"dev"`)) {
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

func TestBootstrapReturnsEphemeralControlToken(t *testing.T) {
	response := request(t, newTestAPI(), http.MethodGet, "/api/v1/bootstrap", nil)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"controlToken":"test-control-token"`)) {
		t.Fatalf("bootstrap response %d: %s", response.Code, response.Body.String())
	}
}
