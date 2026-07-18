package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	browserapp "github.com/miahwk/profileweave/internal/browser/application"
	profileapp "github.com/miahwk/profileweave/internal/profile/application"
	profiledomain "github.com/miahwk/profileweave/internal/profile/domain"
)

type errorBody struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string                     `json:"code"`
	Message string                     `json:"message"`
	Details []profiledomain.FieldError `json:"details,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string, details []profiledomain.FieldError) {
	writeJSON(w, status, errorBody{Error: apiError{Code: code, Message: message, Details: details}})
}

func writeError(w http.ResponseWriter, err error) {
	var validation *profiledomain.ValidationError
	switch {
	case errors.As(err, &validation):
		writeAPIError(w, http.StatusUnprocessableEntity, "profile_invalid", "profile configuration is invalid", validation.Details)
	case errors.Is(err, profiledomain.ErrNotFound):
		writeAPIError(w, http.StatusNotFound, "profile_not_found", "profile was not found", nil)
	case errors.Is(err, profiledomain.ErrTrashNotFound):
		writeAPIError(w, http.StatusNotFound, "trash_not_found", "trashed profile was not found", nil)
	case errors.Is(err, profiledomain.ErrConflict):
		writeAPIError(w, http.StatusConflict, "revision_conflict", "profile was changed by another request", nil)
	case errors.Is(err, profileapp.ErrProfileRunning):
		writeAPIError(w, http.StatusConflict, "profile_running", "stop the profile before changing or deleting it", nil)
	case errors.Is(err, browserapp.ErrAlreadyRunning):
		writeAPIError(w, http.StatusConflict, "session_conflict", "profile already has an active browser session", nil)
	case errors.Is(err, browserapp.ErrNotRunning):
		writeAPIError(w, http.StatusConflict, "session_not_running", "profile has no active browser session", nil)
	case errors.Is(err, browserapp.ErrShuttingDown):
		writeAPIError(w, http.StatusConflict, "application_shutting_down", "application is shutting down", nil)
	case errors.Is(err, browserapp.ErrInvalidProfile):
		writeAPIError(w, http.StatusUnprocessableEntity, "profile_invalid", "profile has blocking consistency errors", nil)
	default:
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
	}
}
