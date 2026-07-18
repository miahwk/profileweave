package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/miahwk/profileweave/internal/browser/domain"
)

func TestDoctorReportsProviderAndBrowserAvailability(t *testing.T) {
	response := request(t, newTestAPI(), http.MethodGet, "/api/v1/doctor", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("doctor response %d: %s", response.Code, response.Body.String())
	}

	var report domain.DoctorReport
	if err := json.Unmarshal(response.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if !report.Healthy || report.Provider.ID != "test-runtime" {
		t.Fatalf("unexpected doctor report %#v", report)
	}
	if report.InspectedBrowsers != 1 || report.AvailableBrowsers != 1 {
		t.Fatalf("unexpected browser counts %#v", report)
	}
	if report.Issues == nil || report.Browsers == nil {
		t.Fatal("doctor arrays must be serialized as arrays, not null")
	}
}

func TestCapabilitiesUseProviderMetadata(t *testing.T) {
	response := request(t, newTestAPI(), http.MethodGet, "/api/v1/capabilities", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("capabilities response %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Provider domain.ProviderInfo `json:"provider"`
		Features []featureCapability `json:"features"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Provider.ID != "test-runtime" || len(payload.Features) != 2 {
		t.Fatalf("capabilities did not come from provider: %#v", payload)
	}
	if payload.Features[1].Status != domain.CapabilityUnsupported {
		t.Fatalf("unexpected capability status %#v", payload.Features[1])
	}
	if bytes.Contains(response.Body.Bytes(), []byte(`"path"`)) || bytes.Contains(response.Body.Bytes(), []byte(`C:/Chrome`)) {
		t.Fatalf("capabilities exposed a browser install path: %s", response.Body.String())
	}
}
