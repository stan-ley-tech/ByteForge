package assertions

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func mustParse(t *testing.T, expr string) Assertion {
	t.Helper()
	a, err := Parse(expr)
	if err != nil {
		t.Fatalf("Parse(%q) returned error: %v", expr, err)
	}
	return a
}

func TestEvaluate_StatusEquals(t *testing.T) {
	ctx := Context{Status: 200}

	if r := mustParse(t, "status == 200").Evaluate(ctx); !r.Passed {
		t.Fatalf("expected pass, got %+v", r)
	}
	if r := mustParse(t, "status == 404").Evaluate(ctx); r.Passed {
		t.Fatalf("expected fail, got %+v", r)
	}
}

func TestEvaluate_BodyFieldEquals(t *testing.T) {
	ctx := Context{Body: []byte(`{"id": 1, "email": "test@example.com"}`)}

	r := mustParse(t, `response.body.email == "test@example.com"`).Evaluate(ctx)
	if !r.Passed {
		t.Fatalf("expected pass, got %+v", r)
	}

	r = mustParse(t, `response.body.email == "wrong@example.com"`).Evaluate(ctx)
	if r.Passed {
		t.Fatalf("expected fail, got %+v", r)
	}
}

func TestEvaluate_BodyFieldExists(t *testing.T) {
	ctx := Context{Body: []byte(`{"id": 1}`)}

	if r := mustParse(t, "response.body.id exists").Evaluate(ctx); !r.Passed {
		t.Fatalf("expected pass, got %+v", r)
	}
	if r := mustParse(t, "response.body.missing exists").Evaluate(ctx); r.Passed {
		t.Fatalf("expected fail, got %+v", r)
	}
	if r := mustParse(t, "response.body.missing not exists").Evaluate(ctx); !r.Passed {
		t.Fatalf("expected pass for not exists on a missing field, got %+v", r)
	}
}

func TestEvaluate_ResponseTimeUnderThreshold(t *testing.T) {
	fast := Context{Time: 100 * time.Millisecond}
	slow := Context{Time: 900 * time.Millisecond}
	a := mustParse(t, "response.time < 500ms")

	if r := a.Evaluate(fast); !r.Passed {
		t.Fatalf("expected pass for fast response, got %+v", r)
	}
	if r := a.Evaluate(slow); r.Passed {
		t.Fatalf("expected fail for slow response, got %+v", r)
	}
}

func TestEvaluate_HeaderEquals(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	ctx := Context{Headers: h}

	r := mustParse(t, `response.header.Content-Type == "application/json"`).Evaluate(ctx)
	if !r.Passed {
		t.Fatalf("expected pass, got %+v", r)
	}
}

func TestEvaluate_WildcardBodyPathContains(t *testing.T) {
	ctx := Context{Body: []byte(`{"items": [{"id": 1}, {"id": 2}, {"id": 3}]}`)}

	r := mustParse(t, "response.body.items[*].id contains 2").Evaluate(ctx)
	if !r.Passed {
		t.Fatalf("expected pass, got %+v", r)
	}

	r = mustParse(t, "response.body.items[*].id contains 9").Evaluate(ctx)
	if r.Passed {
		t.Fatalf("expected fail, got %+v", r)
	}
}

func TestEvaluate_NonJSONBodyProducesError(t *testing.T) {
	ctx := Context{Body: []byte("not json")}
	r := mustParse(t, "response.body.id exists").Evaluate(ctx)
	if r.Err == nil {
		t.Fatal("expected an error evaluating a body path against a non-JSON response")
	}
}

func TestParse_RejectsUnknownOperand(t *testing.T) {
	if _, err := Parse("bogus.thing == 1"); err == nil {
		t.Fatal("expected an error for an unrecognized operand")
	}
}

func TestParse_RejectsUnknownOperator(t *testing.T) {
	if _, err := Parse("status ~= 200"); err == nil {
		t.Fatal("expected an error for an unrecognized operator")
	}
}

func TestParse_RejectsMissingValue(t *testing.T) {
	if _, err := Parse("status =="); err == nil {
		t.Fatal("expected an error for a missing right-hand value")
	}
}

func TestParseAll_SkipsBlankLines(t *testing.T) {
	list, err := ParseAll([]string{"status == 200", "  ", "", "response.time < 500ms"})
	if err != nil {
		t.Fatalf("ParseAll returned error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}
}

func TestResult_MarshalJSON_UsesLowercaseFieldNames(t *testing.T) {
	// A previous version of Result had no json tags, so it serialized as
	// {"Passed": true, "Message": "..."} — every non-Go client checking
	// result.passed (lowercase) would read undefined and treat every
	// assertion as failed regardless of the actual outcome.
	r := mustParse(t, "status == 200").Evaluate(Context{Status: 200})

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if decoded["passed"] != true {
		t.Fatalf("decoded[\"passed\"] = %v, want true (got keys: %v)", decoded["passed"], decoded)
	}
	if decoded["message"] == nil {
		t.Fatalf("decoded[\"message\"] missing (got keys: %v)", decoded)
	}
}

func TestEvaluate_NumberEqualityIgnoresFloatFormatting(t *testing.T) {
	ctx := Context{Body: []byte(`{"count": 5}`)}
	r := mustParse(t, "response.body.count == 5").Evaluate(ctx)
	if !r.Passed {
		t.Fatalf("expected pass, got %+v", r)
	}
}
