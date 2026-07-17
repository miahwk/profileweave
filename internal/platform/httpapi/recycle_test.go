package httpapi

import (
	"bytes"
	"net/http"
	"testing"
)

func TestRecycleBinAPIListsRestoresAndPurges(t *testing.T) {
	api := newTestAPI()
	profile := createProfile(t, api)
	deleted := request(t, api, http.MethodDelete, "/api/v1/profiles/"+profile.ID, nil)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete response %d: %s", deleted.Code, deleted.Body.String())
	}
	trash := request(t, api, http.MethodGet, "/api/v1/trash", nil)
	if trash.Code != http.StatusOK || !bytes.Contains(trash.Body.Bytes(), []byte(profile.ID)) || bytes.Contains(trash.Body.Bytes(), []byte("dataRestoreToken")) {
		t.Fatalf("trash response %d: %s", trash.Code, trash.Body.String())
	}
	restored := request(t, api, http.MethodPost, "/api/v1/trash/"+profile.ID+"/restore", nil)
	if restored.Code != http.StatusOK || !bytes.Contains(restored.Body.Bytes(), []byte(profile.ID)) {
		t.Fatalf("restore response %d: %s", restored.Code, restored.Body.String())
	}
	if response := request(t, api, http.MethodDelete, "/api/v1/profiles/"+profile.ID, nil); response.Code != http.StatusNoContent {
		t.Fatalf("second delete response %d: %s", response.Code, response.Body.String())
	}
	purged := request(t, api, http.MethodDelete, "/api/v1/trash/"+profile.ID, nil)
	if purged.Code != http.StatusNoContent {
		t.Fatalf("purge response %d: %s", purged.Code, purged.Body.String())
	}
	missing := request(t, api, http.MethodPost, "/api/v1/trash/"+profile.ID+"/restore", nil)
	if missing.Code != http.StatusNotFound || !bytes.Contains(missing.Body.Bytes(), []byte(`"code":"trash_not_found"`)) {
		t.Fatalf("missing restore response %d: %s", missing.Code, missing.Body.String())
	}
}
