package collections

import (
	"bytes"
	"strings"
	"testing"
)

func TestValidate_RequiresCollectionName(t *testing.T) {
	c := &Collection{}
	if err := c.Validate(); err == nil {
		t.Fatal("expected an error for a collection with no name")
	}
}

func TestValidate_RejectsUnsupportedMethod(t *testing.T) {
	c := &Collection{
		Name:     "Demo",
		Requests: []Request{{Name: "Weird", Method: "FETCH", URL: "https://example.com"}},
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected an error for an unsupported HTTP method")
	}
}

func TestValidate_RejectsMissingURL(t *testing.T) {
	c := &Collection{
		Name:     "Demo",
		Requests: []Request{{Name: "No URL", Method: "GET"}},
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected an error for a request with no URL")
	}
}

func TestValidate_RejectsDuplicateRequestIDs(t *testing.T) {
	c := &Collection{
		Name: "Demo",
		Requests: []Request{
			{ID: "dup", Name: "One", Method: "GET", URL: "https://example.com/1"},
			{ID: "dup", Name: "Two", Method: "GET", URL: "https://example.com/2"},
		},
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected an error for duplicate request IDs")
	}
}

func TestValidate_AcceptsWellFormedCollection(t *testing.T) {
	c := &Collection{
		Name: "Demo",
		Requests: []Request{
			{Name: "Get user", Method: "GET", URL: "https://api.example.com/users/{{USER_ID}}"},
		},
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddRequest_AssignsID(t *testing.T) {
	c := New("Demo")
	req := c.AddRequest(Request{Name: "Ping", Method: "GET", URL: "https://example.com"})
	if req.ID == "" {
		t.Fatal("AddRequest did not assign an ID")
	}
	if len(c.Requests) != 1 {
		t.Fatalf("len(c.Requests) = %d, want 1", len(c.Requests))
	}
}

func TestDecode_AssignsMissingIDsAndValidates(t *testing.T) {
	raw := `{
		"name": "Demo",
		"requests": [
			{"name": "Ping", "method": "GET", "url": "https://example.com"}
		]
	}`

	c, err := Decode(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if c.ID == "" {
		t.Fatal("Decode did not assign a collection ID")
	}
	if c.Requests[0].ID == "" {
		t.Fatal("Decode did not assign a request ID")
	}
}

func TestDecode_RejectsInvalidCollection(t *testing.T) {
	raw := `{"name": "Demo", "requests": [{"name": "Bad", "method": "TRACE", "url": "https://example.com"}]}`
	if _, err := Decode(strings.NewReader(raw)); err == nil {
		t.Fatal("expected Decode to reject an unsupported method")
	}
}

func TestDecode_RejectsUnknownFields(t *testing.T) {
	raw := `{"name": "Demo", "totallyMadeUpField": true, "requests": []}`
	if _, err := Decode(strings.NewReader(raw)); err == nil {
		t.Fatal("expected Decode to reject unknown fields")
	}
}

func TestEncodeDecode_RoundTrip(t *testing.T) {
	original := New("Round Trip")
	original.AddRequest(Request{
		Name:   "Login",
		Method: "POST",
		URL:    "{{BASE_URL}}/login",
		Body:   Body{Type: BodyJSON, Content: `{"user":"a"}`},
		Extract: []Extraction{
			{Variable: "access_token", From: "body.access_token"},
		},
		Assertions: []string{"status == 200"},
	})

	var buf bytes.Buffer
	if err := Encode(&buf, original); err != nil {
		t.Fatalf("Encode returned error: %v", err)
	}

	decoded, err := Decode(&buf)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	if decoded.Name != original.Name {
		t.Fatalf("Name = %q, want %q", decoded.Name, original.Name)
	}
	if len(decoded.Requests) != 1 || decoded.Requests[0].URL != "{{BASE_URL}}/login" {
		t.Fatalf("round-tripped request mismatch: %+v", decoded.Requests)
	}
	if decoded.Requests[0].Extract[0].Variable != "access_token" {
		t.Fatalf("round-tripped extraction mismatch: %+v", decoded.Requests[0].Extract)
	}
}
