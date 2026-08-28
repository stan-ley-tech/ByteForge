// Package api exposes ByteForge's engine (collections, environments, the
// runner) over HTTP: a REST API for CRUD and synchronous runs, plus a
// WebSocket endpoint that streams a run's progress step by step.
package api

import (
	"net/http"
	"time"

	"github.com/stan-ley-tech/ByteForge/internal/runner"
	"github.com/stan-ley-tech/ByteForge/internal/storage"
)

// Server wires the HTTP handlers to a Store and a Runner. It holds no other
// state, so it's safe to construct once at startup and share across
// requests.
type Server struct {
	store  *storage.Store
	runner *runner.Runner
	mux    *http.ServeMux
}

// NewServer builds a Server and registers all routes.
func NewServer(store *storage.Store, rn *runner.Runner) *Server {
	s := &Server{store: store, runner: rn, mux: http.NewServeMux()}
	s.routes()
	return s
}

// ServeHTTP satisfies http.Handler so a Server can be passed directly to
// http.ListenAndServe or wrapped in middleware.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)

	s.mux.HandleFunc("GET /api/collections", s.handleListCollections)
	s.mux.HandleFunc("POST /api/collections", s.handleCreateCollection)
	s.mux.HandleFunc("GET /api/collections/{id}", s.handleGetCollection)
	s.mux.HandleFunc("PUT /api/collections/{id}", s.handleUpdateCollection)
	s.mux.HandleFunc("DELETE /api/collections/{id}", s.handleDeleteCollection)
	s.mux.HandleFunc("GET /api/collections/{id}/export", s.handleExportCollection)
	s.mux.HandleFunc("POST /api/collections/{id}/run", s.handleRunCollection)
	s.mux.HandleFunc("GET /api/collections/{id}/runs", s.handleListRuns)

	s.mux.HandleFunc("GET /api/environments", s.handleListEnvironments)
	s.mux.HandleFunc("POST /api/environments", s.handleCreateEnvironment)
	s.mux.HandleFunc("GET /api/environments/{id}", s.handleGetEnvironment)
	s.mux.HandleFunc("PUT /api/environments/{id}", s.handleUpdateEnvironment)
	s.mux.HandleFunc("DELETE /api/environments/{id}", s.handleDeleteEnvironment)

	s.mux.HandleFunc("POST /api/requests/send", s.handleSendRequest)
	s.mux.HandleFunc("GET /api/history", s.handleListHistory)

	s.mux.HandleFunc("GET /api/ws/collections/{id}/run", s.handleRunCollectionWS)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}
