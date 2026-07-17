package httpapi

import (
	"net/http"
	"time"

	profiledomain "github.com/miahwk/profileweave/internal/profile/domain"
)

type trashItemResponse struct {
	Profile        profiledomain.Profile `json:"profile"`
	DeletedAt      time.Time             `json:"deletedAt"`
	HasBrowserData bool                  `json:"hasBrowserData"`
}

func (a *API) listTrash(w http.ResponseWriter, r *http.Request) {
	items, err := a.profiles.ListTrash(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	response := make([]trashItemResponse, 0, len(items))
	for _, item := range items {
		response = append(response, trashItemResponse{
			Profile: item.Profile, DeletedAt: item.DeletedAt, HasBrowserData: item.DataRestoreToken != "",
		})
	}
	writeJSON(w, http.StatusOK, listResponse[trashItemResponse]{Items: response})
}

func (a *API) restoreTrash(w http.ResponseWriter, r *http.Request) {
	profile, err := a.profiles.Restore(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (a *API) purgeTrash(w http.ResponseWriter, r *http.Request) {
	if err := a.profiles.Purge(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
