package jsonpath

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func decode(t *testing.T, raw string) any {
	t.Helper()
	var data any
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("invalid test fixture JSON: %v", err)
	}
	return data
}

func TestQuery_FieldAccess(t *testing.T) {
	data := decode(t, `{"id": 7, "email": "jane@example.com"}`)

	got, err := Query(data, "id")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if got != float64(7) {
		t.Fatalf("Query(id) = %v, want 7", got)
	}

	got, err = Query(data, "email")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if got != "jane@example.com" {
		t.Fatalf("Query(email) = %v", got)
	}
}

func TestQuery_NestedField(t *testing.T) {
	data := decode(t, `{"data": {"user": {"id": 99}}}`)

	got, err := Query(data, "data.user.id")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if got != float64(99) {
		t.Fatalf("Query(data.user.id) = %v, want 99", got)
	}
}

func TestQuery_LeadingDollarSign(t *testing.T) {
	data := decode(t, `{"data": {"id": 1}}`)

	got, err := Query(data, "$.data.id")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if got != float64(1) {
		t.Fatalf("Query($.data.id) = %v, want 1", got)
	}
}

func TestQuery_ArrayIndex(t *testing.T) {
	data := decode(t, `{"items": [{"id": 1}, {"id": 2}, {"id": 3}]}`)

	got, err := Query(data, "items[1].id")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if got != float64(2) {
		t.Fatalf("Query(items[1].id) = %v, want 2", got)
	}
}

func TestQuery_Wildcard(t *testing.T) {
	data := decode(t, `{"items": [{"id": 1}, {"id": 2}, {"id": 3}]}`)

	got, err := Query(data, "items[*].id")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	want := []any{float64(1), float64(2), float64(3)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Query(items[*].id) = %v, want %v", got, want)
	}
}

func TestQuery_WildcardSkipsNonMatchingElements(t *testing.T) {
	data := decode(t, `{"items": [{"id": 1}, {"name": "no id here"}, {"id": 3}]}`)

	got, err := Query(data, "items[*].id")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	want := []any{float64(1), float64(3)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Query(items[*].id) = %v, want %v", got, want)
	}
}

func TestQuery_MissingFieldIsErrNotFound(t *testing.T) {
	data := decode(t, `{"id": 1}`)

	_, err := Query(data, "email")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Query(email) error = %v, want ErrNotFound", err)
	}
}

func TestQuery_IndexOutOfRangeIsErrNotFound(t *testing.T) {
	data := decode(t, `{"items": [1, 2]}`)

	_, err := Query(data, "items[5]")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Query(items[5]) error = %v, want ErrNotFound", err)
	}
}

func TestQuery_IndexOnNonArrayIsErrNotFound(t *testing.T) {
	data := decode(t, `{"id": 1}`)

	_, err := Query(data, "id[0]")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Query(id[0]) error = %v, want ErrNotFound", err)
	}
}

func TestQuery_InvalidSegmentIsError(t *testing.T) {
	data := decode(t, `{"id": 1}`)

	_, err := Query(data, "id[oops]")
	if err == nil {
		t.Fatal("expected an error for a malformed path segment")
	}
}

func TestQuery_EmptyPathReturnsRoot(t *testing.T) {
	data := decode(t, `{"id": 1}`)

	got, err := Query(data, "")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if !reflect.DeepEqual(got, data) {
		t.Fatalf("Query(\"\") = %v, want the root value", got)
	}
}
