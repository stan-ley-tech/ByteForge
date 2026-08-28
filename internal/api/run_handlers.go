package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/stan-ley-tech/ByteForge/internal/collections"
	"github.com/stan-ley-tech/ByteForge/internal/environments"
	"github.com/stan-ley-tech/ByteForge/internal/runner"
	"github.com/stan-ley-tech/ByteForge/internal/storage"
)

type runRequest struct {
	EnvironmentID string `json:"environmentId"`
	StopOnFailure bool   `json:"stopOnFailure"`
}

// resolveEnvironment loads the environment for a run when one was
// requested. A blank ID is valid — it means "no environment", not an
// error — since a collection with fully literal URLs doesn't need one.
func (s *Server) resolveEnvironment(r *http.Request, id string) (*environments.Environment, error) {
	if id == "" {
		return nil, nil
	}
	return s.store.GetEnvironment(r.Context(), id)
}

func (s *Server) handleRunCollection(w http.ResponseWriter, r *http.Request) {
	coll, err := s.store.GetCollection(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}

	var body runRequest
	if r.ContentLength != 0 {
		if err := readJSON(r, &body); err != nil {
			writeError(w, r, badRequest(err))
			return
		}
	}

	env, err := s.resolveEnvironment(r, body.EnvironmentID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	report, runErr := s.runner.Run(r.Context(), coll, env, runner.Options{StopOnFailure: body.StopOnFailure})
	if runErr != nil && report == nil {
		writeError(w, r, runErr)
		return
	}

	if err := s.persistRun(r, coll.ID, report); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) persistRun(r *http.Request, collectionID string, report *runner.Report) error {
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}
	_, err = s.store.SaveRun(r.Context(), storage.RunRecord{
		CollectionID:    collectionID,
		CollectionName:  report.CollectionName,
		EnvironmentName: report.Environment,
		Report:          data,
		Passed:          report.Passed,
		Failed:          report.Failed,
		StartedAt:       report.Started,
		DurationMS:      report.Duration.Milliseconds(),
	})
	return err
}

// runSummary mirrors storage.RunRecord for the API surface, with Report
// embedded as raw JSON instead of the base64 string Go's encoding/json
// would otherwise produce for a []byte field.
type runSummary struct {
	ID              string          `json:"id"`
	CollectionID    string          `json:"collectionId"`
	CollectionName  string          `json:"collectionName"`
	EnvironmentName string          `json:"environmentName"`
	Report          json.RawMessage `json:"report"`
	Passed          int             `json:"passed"`
	Failed          int             `json:"failed"`
	StartedAt       time.Time       `json:"startedAt"`
	DurationMS      int64           `json:"durationMs"`
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := s.store.ListRuns(r.Context(), r.PathValue("id"), 50)
	if err != nil {
		writeError(w, r, err)
		return
	}

	summaries := make([]runSummary, len(runs))
	for i, run := range runs {
		summaries[i] = runSummary{
			ID:              run.ID,
			CollectionID:    run.CollectionID,
			CollectionName:  run.CollectionName,
			EnvironmentName: run.EnvironmentName,
			Report:          json.RawMessage(run.Report),
			Passed:          run.Passed,
			Failed:          run.Failed,
			StartedAt:       run.StartedAt,
			DurationMS:      run.DurationMS,
		}
	}
	writeJSON(w, http.StatusOK, summaries)
}

type sendRequestInput struct {
	Request       collections.Request `json:"request"`
	EnvironmentID string              `json:"environmentId"`
}

func (s *Server) handleSendRequest(w http.ResponseWriter, r *http.Request) {
	var input sendRequestInput
	if err := readJSON(r, &input); err != nil {
		writeError(w, r, badRequest(err))
		return
	}

	env, err := s.resolveEnvironment(r, input.EnvironmentID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	step := s.runner.SendOne(r.Context(), input.Request, env, runner.Options{})

	_ = s.store.AddHistory(r.Context(), storage.HistoryEntry{
		RequestName: input.Request.Name,
		Method:      input.Request.Method,
		URL:         step.URL,
		Status:      step.Status,
		DurationMS:  step.Duration.Milliseconds(),
		ExecutedAt:  time.Now(),
	})

	writeJSON(w, http.StatusOK, step)
}

func (s *Server) handleListHistory(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListHistory(r.Context(), 100)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}
