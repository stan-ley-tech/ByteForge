package api

import (
	"net/http"

	"github.com/stan-ley-tech/ByteForge/internal/collections"
)

func (s *Server) handleListCollections(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListCollections(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateCollection(w http.ResponseWriter, r *http.Request) {
	var c collections.Collection
	if err := readJSON(r, &c); err != nil {
		writeError(w, r, badRequest(err))
		return
	}

	c.AssignMissingIDs()
	if err := c.Validate(); err != nil {
		writeError(w, r, badRequest(err))
		return
	}
	if err := s.store.SaveCollection(r.Context(), &c); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, &c)
}

func (s *Server) handleGetCollection(w http.ResponseWriter, r *http.Request) {
	c, err := s.store.GetCollection(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleUpdateCollection(w http.ResponseWriter, r *http.Request) {
	var c collections.Collection
	if err := readJSON(r, &c); err != nil {
		writeError(w, r, badRequest(err))
		return
	}
	c.ID = r.PathValue("id")

	if err := c.Validate(); err != nil {
		writeError(w, r, badRequest(err))
		return
	}
	if err := s.store.SaveCollection(r.Context(), &c); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, &c)
}

func (s *Server) handleDeleteCollection(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteCollection(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleExportCollection returns the collection as a downloadable JSON
// file. Collections never carry resolved secret values (auth and body
// fields hold {{VARIABLE}} references, not literals), so no redaction is
// needed at this boundary — see internal/collections/codec.go.
func (s *Server) handleExportCollection(w http.ResponseWriter, r *http.Request) {
	c, err := s.store.GetCollection(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+c.Name+`.json"`)
	if err := collections.Encode(w, c); err != nil {
		writeError(w, r, err)
		return
	}
}
