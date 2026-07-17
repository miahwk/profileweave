package httpapi

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
)

const controlTokenHeader = "X-ProfileWeave-Token"

func newControlToken() string {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		panic("secure random source unavailable")
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func (a *API) bootstrap(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"controlToken": a.controlToken})
}

func requireControlToken(next http.Handler, expected string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		provided := r.Header.Get(controlTokenHeader)
		if len(provided) != len(expected) || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			writeAPIError(w, http.StatusForbidden, "control_token_invalid", "a valid local control token is required", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}
