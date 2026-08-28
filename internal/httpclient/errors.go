package httpclient

import "fmt"

// RequestError wraps a transport-level failure (DNS, connect, TLS, timeout,
// context cancellation) with the URL that was being requested, so callers
// several layers up the stack don't need to thread that context through
// manually.
type RequestError struct {
	URL string
	Err error
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("httpclient: request to %s failed: %v", e.URL, e.Err)
}

func (e *RequestError) Unwrap() error {
	return e.Err
}
