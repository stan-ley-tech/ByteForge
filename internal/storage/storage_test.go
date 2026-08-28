package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stan-ley-tech/ByteForge/internal/collections"
	"github.com/stan-ley-tech/ByteForge/internal/environments"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open("")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCollections_SaveGetListDelete(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	c := collections.New("Demo")
	c.AddRequest(collections.Request{Name: "Ping", Method: "GET", URL: "https://example.com"})

	if err := s.SaveCollection(ctx, c); err != nil {
		t.Fatalf("SaveCollection returned error: %v", err)
	}

	got, err := s.GetCollection(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetCollection returned error: %v", err)
	}
	if got.Name != "Demo" || len(got.Requests) != 1 {
		t.Fatalf("GetCollection() = %+v, want a round-tripped Demo collection", got)
	}

	c.Name = "Demo Renamed"
	if err := s.SaveCollection(ctx, c); err != nil {
		t.Fatalf("SaveCollection (update) returned error: %v", err)
	}
	got, err = s.GetCollection(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetCollection after update returned error: %v", err)
	}
	if got.Name != "Demo Renamed" {
		t.Fatalf("GetCollection().Name = %q, want %q (update should overwrite in place)", got.Name, "Demo Renamed")
	}

	list, err := s.ListCollections(ctx)
	if err != nil {
		t.Fatalf("ListCollections returned error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}

	if err := s.DeleteCollection(ctx, c.ID); err != nil {
		t.Fatalf("DeleteCollection returned error: %v", err)
	}
	if _, err := s.GetCollection(ctx, c.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetCollection after delete: err = %v, want ErrNotFound", err)
	}
}

func TestEnvironments_SaveGetByNameListDelete(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	env := environments.New("Staging")
	env.Set("BASE_URL", "https://staging.example.com", false)
	env.Set("API_KEY", "sk-secret", true)

	if err := s.SaveEnvironment(ctx, env); err != nil {
		t.Fatalf("SaveEnvironment returned error: %v", err)
	}

	got, err := s.GetEnvironmentByName(ctx, "Staging")
	if err != nil {
		t.Fatalf("GetEnvironmentByName returned error: %v", err)
	}
	if got.Variables["API_KEY"].Value != "sk-secret" || !got.Variables["API_KEY"].Secret {
		t.Fatalf("round-tripped secret variable mismatch: %+v", got.Variables["API_KEY"])
	}

	list, err := s.ListEnvironments(ctx)
	if err != nil {
		t.Fatalf("ListEnvironments returned error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}

	if err := s.DeleteEnvironment(ctx, env.ID); err != nil {
		t.Fatalf("DeleteEnvironment returned error: %v", err)
	}
	if _, err := s.GetEnvironment(ctx, env.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetEnvironment after delete: err = %v, want ErrNotFound", err)
	}
}

func TestRuns_SaveAndListFilteredByCollection(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	id, err := s.SaveRun(ctx, RunRecord{
		CollectionID:   "coll-a",
		CollectionName: "A",
		Report:         []byte(`{"passed":1,"failed":0}`),
		Passed:         1,
		StartedAt:      time.Now(),
		DurationMS:     42,
	})
	if err != nil {
		t.Fatalf("SaveRun returned error: %v", err)
	}
	if id == "" {
		t.Fatal("SaveRun did not return an ID")
	}

	if _, err := s.SaveRun(ctx, RunRecord{
		CollectionID:   "coll-b",
		CollectionName: "B",
		Report:         []byte(`{"passed":0,"failed":1}`),
		Failed:         1,
		StartedAt:      time.Now(),
	}); err != nil {
		t.Fatalf("SaveRun returned error: %v", err)
	}

	runsA, err := s.ListRuns(ctx, "coll-a", 0)
	if err != nil {
		t.Fatalf("ListRuns returned error: %v", err)
	}
	if len(runsA) != 1 || runsA[0].CollectionName != "A" {
		t.Fatalf("ListRuns(coll-a) = %+v, want exactly the A run", runsA)
	}

	all, err := s.ListRuns(ctx, "", 0)
	if err != nil {
		t.Fatalf("ListRuns returned error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("len(all) = %d, want 2", len(all))
	}
}

func TestHistory_AddAndList(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := s.AddHistory(ctx, HistoryEntry{
			RequestName: "Ping",
			Method:      "GET",
			URL:         "https://example.com",
			Status:      200,
			DurationMS:  int64(10 + i),
		}); err != nil {
			t.Fatalf("AddHistory returned error: %v", err)
		}
	}

	list, err := s.ListHistory(ctx, 2)
	if err != nil {
		t.Fatalf("ListHistory returned error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2 (limit should be respected)", len(list))
	}
}
