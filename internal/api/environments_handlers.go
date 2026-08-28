package api

import (
	"net/http"

	"github.com/stan-ley-tech/ByteForge/internal/environments"
)

// handleListEnvironments returns every environment with secrets masked.
// The raw secret values never leave the server on a listing call; only
// handleRunCollection and handleSendRequest, which need real values to
// actually make requests, read them unredacted from storage.
func (s *Server) handleListEnvironments(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListEnvironments(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	redacted := make([]*environments.Environment, len(list))
	for i, env := range list {
		redacted[i] = env.Redacted()
	}
	writeJSON(w, http.StatusOK, redacted)
}

func (s *Server) handleCreateEnvironment(w http.ResponseWriter, r *http.Request) {
	var env environments.Environment
	if err := readJSON(r, &env); err != nil {
		writeError(w, r, badRequest(err))
		return
	}
	if env.Name == "" {
		writeError(w, r, badRequest(errEnvironmentNameRequired))
		return
	}
	if err := s.store.SaveEnvironment(r.Context(), &env); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, env.Redacted())
}

func (s *Server) handleGetEnvironment(w http.ResponseWriter, r *http.Request) {
	env, err := s.store.GetEnvironment(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, env.Redacted())
}

func (s *Server) handleUpdateEnvironment(w http.ResponseWriter, r *http.Request) {
	var env environments.Environment
	if err := readJSON(r, &env); err != nil {
		writeError(w, r, badRequest(err))
		return
	}
	env.ID = r.PathValue("id")
	if env.Name == "" {
		writeError(w, r, badRequest(errEnvironmentNameRequired))
		return
	}
	if err := s.store.SaveEnvironment(r.Context(), &env); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, env.Redacted())
}

func (s *Server) handleDeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteEnvironment(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
