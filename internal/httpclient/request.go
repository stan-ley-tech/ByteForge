package httpclient

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"strings"
)

// Request describes an HTTP request in a transport-agnostic way so it can be
// built once (from a saved collection entry, a CLI flag set, or an API
// payload) and executed by any Client.
type Request struct {
	Method  string
	URL     string
	Headers map[string][]string
	Query   map[string][]string
	Body    []byte
}

// NewRequest creates a Request for method and rawURL with an empty body.
func NewRequest(method, rawURL string) *Request {
	return &Request{
		Method:  strings.ToUpper(method),
		URL:     rawURL,
		Headers: map[string][]string{},
		Query:   map[string][]string{},
	}
}

// WithHeader appends a header value, preserving any existing values for the
// same key (headers like Accept can legitimately repeat).
func (r *Request) WithHeader(key, value string) *Request {
	r.Headers[key] = append(r.Headers[key], value)
	return r
}

// WithQuery appends a query parameter value.
func (r *Request) WithQuery(key, value string) *Request {
	r.Query[key] = append(r.Query[key], value)
	return r
}

// WithBody sets a raw request body and returns the Request for chaining.
func (r *Request) WithBody(body []byte) *Request {
	r.Body = body
	return r
}

// toHTTP builds a real *http.Request bound to ctx, merging Query into the
// URL and copying Headers across.
func (r *Request) toHTTP(ctx context.Context) (*http.Request, error) {
	parsed, err := url.Parse(r.URL)
	if err != nil {
		return nil, err
	}

	if len(r.Query) > 0 {
		q := parsed.Query()
		for key, values := range r.Query {
			for _, v := range values {
				q.Add(key, v)
			}
		}
		parsed.RawQuery = q.Encode()
	}

	var body *bytes.Reader
	if len(r.Body) > 0 {
		body = bytes.NewReader(r.Body)
	} else {
		body = bytes.NewReader(nil)
	}

	httpReq, err := http.NewRequestWithContext(ctx, r.Method, parsed.String(), body)
	if err != nil {
		return nil, err
	}

	for key, values := range r.Headers {
		for _, v := range values {
			httpReq.Header.Add(key, v)
		}
	}

	return httpReq, nil
}
