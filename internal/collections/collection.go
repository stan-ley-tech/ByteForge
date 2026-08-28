// Package collections models saved requests and the collections that group
// them: everything a user builds in the request editor and persists for
// reuse, chaining, and automated test runs.
package collections

import (
	"fmt"

	"github.com/stan-ley-tech/ByteForge/internal/idgen"
)

// KV is an ordered, individually-toggleable key/value pair used for headers
// and query parameters. A plain map won't do here: header order can matter
// to some servers, and users routinely disable a param without deleting it.
type KV struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

// BodyType identifies how Body.Content should be interpreted and, where
// applicable, which Content-Type header to send if the user hasn't set one
// explicitly.
type BodyType string

const (
	BodyNone BodyType = "none"
	BodyJSON BodyType = "json"
	BodyXML  BodyType = "xml"
	BodyForm BodyType = "form"
	BodyRaw  BodyType = "raw"
)

// Body is a request payload before template rendering.
type Body struct {
	Type    BodyType `json:"type"`
	Content string   `json:"content,omitempty"`
}

// AuthType selects how a request authenticates.
type AuthType string

const (
	AuthNone   AuthType = "none"
	AuthBearer AuthType = "bearer"
	AuthBasic  AuthType = "basic"
	AuthAPIKey AuthType = "apikey"
)

// Auth holds credentials for a request. Values are expected to be
// {{VARIABLE}} references into an environment rather than literal secrets,
// so a saved collection never carries a real API key at rest.
type Auth struct {
	Type AuthType `json:"type"`

	// Bearer
	Token string `json:"token,omitempty"`

	// Basic
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	// API key
	KeyName  string `json:"keyName,omitempty"`
	KeyValue string `json:"keyValue,omitempty"`
	In       string `json:"in,omitempty"` // "header" or "query"
}

// Extraction pulls a value out of a response and saves it under Variable so
// later requests in a chain can reference it via {{Variable}}. From is a
// JSONPath-like expression evaluated against the response, e.g.
// "body.access_token" or "body.data.items[0].id".
type Extraction struct {
	Variable string `json:"variable"`
	From     string `json:"from"`
}

// Request is a single saved HTTP call: everything needed to build and
// execute it, plus the assertions and extractions that turn it into a test
// step rather than just a one-off call.
type Request struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Method      string       `json:"method"`
	URL         string       `json:"url"`
	Headers     []KV         `json:"headers,omitempty"`
	QueryParams []KV         `json:"queryParams,omitempty"`
	Body        Body         `json:"body,omitempty"`
	Auth        Auth         `json:"auth,omitempty"`
	Assertions  []string     `json:"assertions,omitempty"`
	Extract     []Extraction `json:"extract,omitempty"`
}

// Collection is an ordered, named group of requests. Order matters: a run
// executes requests top to bottom, which is what makes chaining ("run
// login, then use its token in profile") predictable.
type Collection struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Requests    []Request `json:"requests"`
}

// New creates an empty, ready-to-use Collection with a fresh ID.
func New(name string) *Collection {
	return &Collection{ID: idgen.New(), Name: name}
}

// AddRequest appends req to the collection, assigning it an ID if it
// doesn't already have one.
func (c *Collection) AddRequest(req Request) *Request {
	if req.ID == "" {
		req.ID = idgen.New()
	}
	c.Requests = append(c.Requests, req)
	return &c.Requests[len(c.Requests)-1]
}

var validMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true,
	"DELETE": true, "HEAD": true, "OPTIONS": true,
}

// Validate checks structural correctness: every request has a name, a
// supported HTTP method, and a non-empty URL. It does not attempt to
// resolve template variables or reach the network.
func (c *Collection) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("collections: collection name is required")
	}

	seen := make(map[string]bool, len(c.Requests))
	for i, req := range c.Requests {
		if req.Name == "" {
			return fmt.Errorf("collections: request %d: name is required", i)
		}
		if !validMethods[req.Method] {
			return fmt.Errorf("collections: request %q: unsupported method %q", req.Name, req.Method)
		}
		if req.URL == "" {
			return fmt.Errorf("collections: request %q: url is required", req.Name)
		}
		if req.ID != "" {
			if seen[req.ID] {
				return fmt.Errorf("collections: duplicate request id %q", req.ID)
			}
			seen[req.ID] = true
		}
	}
	return nil
}

// AssignMissingIDs fills in IDs for the collection and any requests that
// don't already have one. Called after decoding user-authored JSON, which
// commonly omits IDs entirely.
func (c *Collection) AssignMissingIDs() {
	if c.ID == "" {
		c.ID = idgen.New()
	}
	for i := range c.Requests {
		if c.Requests[i].ID == "" {
			c.Requests[i].ID = idgen.New()
		}
	}
}
