package runner

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stan-ley-tech/ByteForge/internal/collections"
	"github.com/stan-ley-tech/ByteForge/internal/environments"
	"github.com/stan-ley-tech/ByteForge/internal/httpclient"
	"github.com/stan-ley-tech/ByteForge/internal/jsonpath"
)

// renderRequest resolves every {{variable}} placeholder in req against env
// and the current chain variables, and builds the concrete httpclient
// request that will actually go over the wire. It errors out (rather than
// sending a request with a literal "{{TOKEN}}" in it) whenever a
// placeholder can't be resolved.
func renderRequest(req collections.Request, env *environments.Environment, vars map[string]string) (*httpclient.Request, error) {
	lookup := env.Lookup(vars)

	url, err := environments.Render(req.URL, lookup, true)
	if err != nil {
		return nil, fmt.Errorf("url: %w", err)
	}
	hreq := httpclient.NewRequest(req.Method, url)

	for _, h := range req.Headers {
		if !h.Enabled {
			continue
		}
		val, err := environments.Render(h.Value, lookup, true)
		if err != nil {
			return nil, fmt.Errorf("header %q: %w", h.Key, err)
		}
		hreq.WithHeader(h.Key, val)
	}

	for _, q := range req.QueryParams {
		if !q.Enabled {
			continue
		}
		val, err := environments.Render(q.Value, lookup, true)
		if err != nil {
			return nil, fmt.Errorf("query param %q: %w", q.Key, err)
		}
		hreq.WithQuery(q.Key, val)
	}

	if err := applyAuth(hreq, req.Auth, lookup); err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}

	if req.Body.Type != collections.BodyNone && req.Body.Content != "" {
		rendered, err := environments.Render(req.Body.Content, lookup, true)
		if err != nil {
			return nil, fmt.Errorf("body: %w", err)
		}
		hreq.WithBody([]byte(rendered))
		if ct := contentTypeFor(req.Body.Type); ct != "" && len(hreq.Headers["Content-Type"]) == 0 {
			hreq.WithHeader("Content-Type", ct)
		}
	}

	return hreq, nil
}

func applyAuth(hreq *httpclient.Request, auth collections.Auth, lookup func(string) (string, bool)) error {
	switch auth.Type {
	case "", collections.AuthNone:
		return nil

	case collections.AuthBearer:
		token, err := environments.Render(auth.Token, lookup, true)
		if err != nil {
			return err
		}
		hreq.WithHeader("Authorization", "Bearer "+token)
		return nil

	case collections.AuthBasic:
		user, err := environments.Render(auth.Username, lookup, true)
		if err != nil {
			return err
		}
		pass, err := environments.Render(auth.Password, lookup, true)
		if err != nil {
			return err
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		hreq.WithHeader("Authorization", "Basic "+encoded)
		return nil

	case collections.AuthAPIKey:
		val, err := environments.Render(auth.KeyValue, lookup, true)
		if err != nil {
			return err
		}
		if auth.In == "query" {
			hreq.WithQuery(auth.KeyName, val)
		} else {
			hreq.WithHeader(auth.KeyName, val)
		}
		return nil

	default:
		return fmt.Errorf("unsupported auth type %q", auth.Type)
	}
}

func contentTypeFor(t collections.BodyType) string {
	switch t {
	case collections.BodyJSON:
		return "application/json"
	case collections.BodyXML:
		return "application/xml"
	case collections.BodyForm:
		return "application/x-www-form-urlencoded"
	default:
		return ""
	}
}

// extractVariables runs each of a request's Extract rules against the
// response body and stores the results into vars, where they become
// available to every subsequent request in the chain via {{Variable}}.
func extractVariables(body []byte, extract []collections.Extraction, vars map[string]string) error {
	if len(extract) == 0 {
		return nil
	}

	var data any
	if len(body) > 0 {
		if err := json.Unmarshal(body, &data); err != nil {
			return fmt.Errorf("response body is not valid JSON, cannot extract variables: %w", err)
		}
	}

	for _, ex := range extract {
		path := strings.TrimPrefix(strings.TrimPrefix(ex.From, "response."), "body.")
		val, err := jsonpath.Query(data, path)
		if err != nil {
			return fmt.Errorf("extract %q from %q: %w", ex.Variable, ex.From, err)
		}
		vars[ex.Variable] = fmt.Sprint(val)
	}
	return nil
}
