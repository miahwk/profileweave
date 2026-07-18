package httpapi

import (
	"net/http"

	"github.com/miahwk/profileweave/internal/browser/domain"
)

type featureCapability struct {
	Key    string                  `json:"key"`
	Label  string                  `json:"label"`
	Status domain.CapabilityStatus `json:"status"`
	Detail string                  `json:"detail,omitempty"`
}

type capabilityResponse struct {
	Provider domain.ProviderInfo        `json:"provider"`
	Browsers []domain.BrowserDescriptor `json:"browsers"`
	Features []featureCapability        `json:"features"`
}

func (a *API) capabilities(w http.ResponseWriter, r *http.Request) {
	browsers, err := a.browsers.Discover(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	provider := a.browsers.RuntimeInfo()
	features := make([]featureCapability, 0, len(provider.Capabilities))
	for _, item := range provider.Capabilities {
		features = append(features, featureCapability{
			Key: item.ID, Label: item.Name, Status: item.Status, Detail: item.Detail,
		})
	}
	writeJSON(w, http.StatusOK, capabilityResponse{
		Provider: provider, Browsers: browsers, Features: features,
	})
}

func (a *API) doctor(w http.ResponseWriter, r *http.Request) {
	report, err := a.browsers.Doctor(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}
