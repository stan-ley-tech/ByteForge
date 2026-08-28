package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stan-ley-tech/ByteForge/internal/runner"
	"github.com/stan-ley-tech/ByteForge/internal/storage"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// The API and the web UI are served from the same origin in production
	// (the Go binary serves the built frontend); during local development
	// the Vite dev server proxies API calls through, so it also arrives
	// same-origin. Nothing legitimate needs cross-origin WebSocket access.
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "" || origin == "http://"+r.Host || origin == "https://"+r.Host
	},
}

// wsEvent is one message in the live run stream. Exactly one of Step or
// Report is set, matching Type.
type wsEvent struct {
	Type   string             `json:"type"` // "step", "done", or "error"
	Step   *runner.StepResult `json:"step,omitempty"`
	Report *runner.Report     `json:"report,omitempty"`
	Error  string             `json:"error,omitempty"`
}

// handleRunCollectionWS runs a collection the same way handleRunCollection
// does, but streams each StepResult over a WebSocket as it completes
// instead of making the client wait for the whole collection to finish —
// this is what lets the UI show test output live rather than as a single
// blocking spinner.
func (s *Server) handleRunCollectionWS(w http.ResponseWriter, r *http.Request) {
	coll, err := s.store.GetCollection(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}

	env, err := s.resolveEnvironment(r, r.URL.Query().Get("environmentId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	stopOnFailure := r.URL.Query().Get("stopOnFailure") == "true"

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	opts := runner.Options{
		StopOnFailure: stopOnFailure,
		OnStep: func(step runner.StepResult) {
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			_ = conn.WriteJSON(wsEvent{Type: "step", Step: &step})
		},
	}

	report, runErr := s.runner.Run(r.Context(), coll, env, opts)
	if runErr != nil && report == nil {
		_ = conn.WriteJSON(wsEvent{Type: "error", Error: runErr.Error()})
		return
	}

	if data, err := json.Marshal(report); err == nil {
		_, _ = s.store.SaveRun(r.Context(), collectRunRecord(coll.ID, report, data))
	}

	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_ = conn.WriteJSON(wsEvent{Type: "done", Report: report})
}

func collectRunRecord(collectionID string, report *runner.Report, data []byte) storage.RunRecord {
	return storage.RunRecord{
		CollectionID:    collectionID,
		CollectionName:  report.CollectionName,
		EnvironmentName: report.Environment,
		Report:          data,
		Passed:          report.Passed,
		Failed:          report.Failed,
		StartedAt:       report.Started,
		DurationMS:      report.Duration.Milliseconds(),
	}
}
