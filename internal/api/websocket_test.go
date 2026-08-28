package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestRunCollectionWS_StreamsStepsThenDone(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": 1}`))
	}))
	defer upstream.Close()

	srv := newTestServer(t)
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	create := doJSON(t, srv, http.MethodPost, "/api/collections", map[string]any{
		"name": "WS Demo",
		"requests": []map[string]any{
			{"name": "Get", "method": "GET", "url": upstream.URL, "assertions": []string{"status == 200"}},
		},
	})
	var created map[string]any
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	id := created["id"].(string)

	wsURL := "ws" + strings.TrimPrefix(httpSrv.URL, "http") + "/api/ws/collections/" + id + "/run"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	var sawStep, sawDone bool
	for i := 0; i < 5 && !sawDone; i++ {
		var evt wsEvent
		if err := conn.ReadJSON(&evt); err != nil {
			t.Fatalf("read websocket message %d: %v", i, err)
		}
		switch evt.Type {
		case "step":
			if evt.Step == nil {
				t.Fatal("step event carried no step")
			}
			sawStep = true
		case "done":
			if evt.Report == nil || evt.Report.Passed != 1 || evt.Report.Failed != 0 {
				t.Fatalf("done event carried unexpected report: %+v", evt.Report)
			}
			sawDone = true
		case "error":
			t.Fatalf("run reported an error: %s", evt.Error)
		}
	}

	if !sawStep {
		t.Fatal("never received a step event")
	}
	if !sawDone {
		t.Fatal("never received a done event")
	}

	// The run should have been persisted, same as a synchronous run.
	runs := doJSON(t, srv, http.MethodGet, "/api/collections/"+id+"/runs", nil)
	var runList []map[string]any
	json.Unmarshal(runs.Body.Bytes(), &runList)
	if len(runList) != 1 {
		t.Fatalf("len(runList) = %d, want 1 (WS run should persist like a synchronous one)", len(runList))
	}
}
