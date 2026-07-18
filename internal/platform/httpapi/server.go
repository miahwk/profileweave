package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	browserapp "github.com/miahwk/profileweave/internal/browser/application"
	"github.com/miahwk/profileweave/internal/browser/domain"
	"github.com/miahwk/profileweave/internal/buildinfo"
	profileapp "github.com/miahwk/profileweave/internal/profile/application"
	profiledomain "github.com/miahwk/profileweave/internal/profile/domain"
)

const maxBodyBytes = 1 << 20

type API struct {
	profiles     *profileapp.Service
	browsers     *browserapp.Service
	controlToken string
	handler      http.Handler
}

func New(profiles *profileapp.Service, browsers *browserapp.Service) *API {
	return newAPI(profiles, browsers, newControlToken())
}

func newAPI(profiles *profileapp.Service, browsers *browserapp.Service, token string) *API {
	api := &API{profiles: profiles, browsers: browsers, controlToken: token}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", api.health)
	mux.HandleFunc("GET /api/v1/bootstrap", api.bootstrap)
	mux.HandleFunc("GET /api/v1/capabilities", api.capabilities)
	mux.HandleFunc("GET /api/v1/runtime/capabilities", api.capabilities)
	mux.HandleFunc("GET /api/v1/doctor", api.doctor)
	mux.HandleFunc("GET /api/v1/profiles", api.listProfiles)
	mux.HandleFunc("POST /api/v1/profiles", api.createProfile)
	mux.HandleFunc("GET /api/v1/profiles/{id}", api.getProfile)
	mux.HandleFunc("PUT /api/v1/profiles/{id}", api.updateProfile)
	mux.HandleFunc("DELETE /api/v1/profiles/{id}", api.deleteProfile)
	mux.HandleFunc("POST /api/v1/profiles/{id}/duplicate", api.duplicateProfile)
	mux.HandleFunc("POST /api/v1/profiles/{id}/validate", api.validateProfile)
	mux.HandleFunc("POST /api/v1/profiles/{id}/launch", api.launchProfile)
	mux.HandleFunc("POST /api/v1/profiles/{id}/stop", api.stopProfile)
	mux.HandleFunc("GET /api/v1/sessions", api.listSessions)
	mux.HandleFunc("GET /api/v1/trash", api.listTrash)
	mux.HandleFunc("POST /api/v1/trash/{id}/restore", api.restoreTrash)
	mux.HandleFunc("DELETE /api/v1/trash/{id}", api.purgeTrash)
	api.handler = securityHeaders(recoverMiddleware(originGuard(requireControlToken(mux, token))))
	return api
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) { a.handler.ServeHTTP(w, r) }

type listResponse[T any] struct {
	Items []T `json:"items"`
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, struct {
		Status string `json:"status"`
		buildinfo.Info
	}{Status: "ok", Info: buildinfo.Current()})
}

func (a *API) listProfiles(w http.ResponseWriter, r *http.Request) {
	items, err := a.profiles.List(r.Context(), r.URL.Query().Get("search"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listResponse[profiledomain.Profile]{Items: items})
}

func (a *API) getProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := a.profiles.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (a *API) createProfile(w http.ResponseWriter, r *http.Request) {
	var input profiledomain.Input
	if err := decodeJSON(w, r, &input); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error(), nil)
		return
	}
	profile, err := a.profiles.Create(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, profile)
}

func (a *API) updateProfile(w http.ResponseWriter, r *http.Request) {
	var request profiledomain.Profile
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error(), nil)
		return
	}
	if request.ID != "" && request.ID != r.PathValue("id") {
		writeAPIError(w, http.StatusUnprocessableEntity, "profile_invalid", "profile ID does not match route", []profiledomain.FieldError{{Field: "id", Message: "profile ID does not match route"}})
		return
	}
	profile, err := a.profiles.Update(r.Context(), r.PathValue("id"), request.Revision, request.Input())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (a *API) deleteProfile(w http.ResponseWriter, r *http.Request) {
	if err := a.profiles.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) duplicateProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := a.profiles.Duplicate(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, profile)
}

func (a *API) validateProfile(w http.ResponseWriter, r *http.Request) {
	report, err := a.profiles.Validate(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (a *API) launchProfile(w http.ResponseWriter, r *http.Request) {
	session, err := a.browsers.Launch(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (a *API) stopProfile(w http.ResponseWriter, r *http.Request) {
	session, err := a.browsers.Stop(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (a *API) listSessions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, listResponse[domain.Session]{Items: a.browsers.List()})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			return errors.New("request body exceeds 1 MiB")
		}
		return errors.New("request body must be a valid JSON object with known fields")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func originGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !loopbackHost(r.Host) {
			writeAPIError(w, http.StatusForbidden, "origin_forbidden", "requests require a loopback Host", nil)
			return
		}
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		if origin != "" {
			u, err := url.Parse(origin)
			if err != nil || u.Scheme != "http" || !loopbackHost(u.Host) {
				writeAPIError(w, http.StatusForbidden, "origin_forbidden", "cross-origin write request rejected", nil)
				return
			}
		}
		if strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
			writeAPIError(w, http.StatusForbidden, "origin_forbidden", "cross-site write request rejected", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loopbackHost(hostPort string) bool {
	host := hostPort
	if parsed, _, err := net.SplitHostPort(hostPort); err == nil {
		host = parsed
	} else if strings.Contains(hostPort, ":") {
		return false
	}
	host = strings.Trim(host, "[]")
	return strings.EqualFold(host, "localhost") || net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback()
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				writeAPIError(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
