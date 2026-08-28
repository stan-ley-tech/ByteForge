package httpclient

import (
	"io"
	"net/http"
	"time"
)

// Response is a fully buffered HTTP response, including the timing
// information the assertion engine and UI need.
type Response struct {
	StatusCode int
	Status     string
	Proto      string
	Header     http.Header
	Body       []byte
	// Truncated is true when Body was cut short by the client's
	// MaxResponseBytes limit.
	Truncated bool
	Duration  time.Duration
}

// StreamResponse is a response whose body has not been read yet. It is
// returned by Client.Stream for large or long-lived payloads where buffering
// the whole thing would be wasteful or unbounded.
type StreamResponse struct {
	StatusCode int
	Status     string
	Proto      string
	Header     http.Header
	Started    time.Time
	Body       io.ReadCloser
}

// Close releases the underlying connection. It is safe to call more than
// once.
func (s *StreamResponse) Close() error {
	if s.Body == nil {
		return nil
	}
	return s.Body.Close()
}

// readLimited reads r up to limit bytes. If limit is 0, it reads until EOF
// with no bound. The second return value reports whether the stream had more
// data than the limit allowed.
func readLimited(r io.Reader, limit int64) (data []byte, truncated bool, err error) {
	if limit <= 0 {
		data, err = io.ReadAll(r)
		return data, false, err
	}

	limited := io.LimitReader(r, limit+1)
	data, err = io.ReadAll(limited)
	if err != nil {
		return nil, false, err
	}
	if int64(len(data)) > limit {
		return data[:limit], true, nil
	}
	return data, false, nil
}
