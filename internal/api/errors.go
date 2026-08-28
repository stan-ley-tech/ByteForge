package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/stan-ley-tech/ByteForge/internal/storage"
)

var errEnvironmentNameRequired = errors.New("environment name is required")

type errorResponse struct {
	Error string `json:"error"`
}

// writeError maps a handler error to an HTTP status and a structured JSON
// body, logging anything that isn't an expected, user-facing condition
// (a bad request, a missing record) so operators can see real failures in
// the server logs without leaking internals to the client.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, storage.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
	case errors.As(err, new(*validationError)):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	default:
		slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

// validationError marks an error as the caller's fault (bad JSON, a failed
// collection.Validate) so writeError can return 400 instead of 500.
type validationError struct{ err error }

func (e *validationError) Error() string { return e.err.Error() }
func (e *validationError) Unwrap() error { return e.err }

func badRequest(err error) error {
	return &validationError{err: err}
}
