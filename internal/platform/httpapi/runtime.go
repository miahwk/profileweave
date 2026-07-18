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

type browserCapability struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
}

type capabilityResponse struct {
	Provider domain.ProviderInfo `json:"provider"`
	Browsers []browserCapability `json:"browsers"`
	Features []featureCapability `json:"features"`
}

func (a *API) capabilities(w http.ResponseWriter, r *http.Request) {
	browsers, err := a.browsers.Discover(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	provider := a.browsers.RuntimeInfo()
	publicBrowsers := make([]browserCapability, 0, len(browsers))
	for _, browser := range browsers {
		publicBrowsers = append(publicBrowsers, browserCapability{
			ID: browser.ID, Name: browser.Name, Available: browser.Available,
		})
	}
	features := make([]featureCapability, 0, len(provider.Capabilities))
	for _, item := range provider.Capabilities {
		features = append(features, featureCapability{
			Key: item.ID, Label: item.Name, Status: item.Status, Detail: item.Detail,
		})
	}
	writeJSON(w, http.StatusOK, capabilityResponse{
		Provider: provider, Browsers: publicBrowsers, Features: features,
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
