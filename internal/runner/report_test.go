package runner

import (
	"encoding/json"
	"testing"
	"time"
)

func TestReport_MarshalJSON_DurationIsMilliseconds(t *testing.T) {
	report := &Report{
		CollectionName: "Demo",
		Duration:       1500 * time.Millisecond,
		Steps: []StepResult{
			{Request: "Ping", Duration: 250 * time.Millisecond, Passed: true},
		},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if decoded["durationMs"] != float64(1500) {
		t.Fatalf("report durationMs = %v, want 1500 (a raw time.Duration would give 1500000000 nanoseconds)", decoded["durationMs"])
	}

	steps, ok := decoded["steps"].([]any)
	if !ok || len(steps) != 1 {
		t.Fatalf("decoded steps = %v", decoded["steps"])
	}
	step := steps[0].(map[string]any)
	if step["durationMs"] != float64(250) {
		t.Fatalf("step durationMs = %v, want 250", step["durationMs"])
	}
}
