// Package jsonpath evaluates a deliberately small subset of JSONPath
// against already-decoded JSON (the output of encoding/json's Unmarshal
// into any). It supports dot-separated field access, a single numeric or
// wildcard index per segment, and an optional leading "$." root — enough
// to express the paths that show up in real API responses ("data.id",
// "items[0].name", "items[*].id") without carrying the weight of the full
// JSONPath grammar.
package jsonpath

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ErrNotFound is returned, often wrapped, when a path segment doesn't match
// the shape of the data: a missing field, an out-of-range index, or an
// index applied to something that isn't an array.
var ErrNotFound = errors.New("jsonpath: not found")

type segment struct {
	name     string
	hasIndex bool
	wildcard bool
	index    int
}

var segmentPattern = regexp.MustCompile(`^([A-Za-z0-9_-]*)(?:\[(\d+|\*)\])?$`)

// Query evaluates path against data. When path contains a wildcard index,
// the result is a []any collecting one value per matching element (elements
// where the rest of the path doesn't resolve are silently skipped, the same
// way a database query skips rows that don't match a filter).
func Query(data any, path string) (any, error) {
	segments, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	return evalSegments(data, segments)
}

func parsePath(path string) ([]segment, error) {
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")
	if path == "" {
		return nil, nil
	}

	parts := strings.Split(path, ".")
	segments := make([]segment, 0, len(parts))
	for _, part := range parts {
		m := segmentPattern.FindStringSubmatch(part)
		if m == nil {
			return nil, fmt.Errorf("jsonpath: invalid path segment %q in %q", part, path)
		}

		seg := segment{name: m[1]}
		switch m[2] {
		case "":
			// no index
		case "*":
			seg.hasIndex = true
			seg.wildcard = true
		default:
			idx, err := strconv.Atoi(m[2])
			if err != nil {
				return nil, fmt.Errorf("jsonpath: invalid index %q in %q", m[2], path)
			}
			seg.hasIndex = true
			seg.index = idx
		}
		segments = append(segments, seg)
	}
	return segments, nil
}

func evalSegments(current any, segments []segment) (any, error) {
	if len(segments) == 0 {
		return current, nil
	}
	seg, rest := segments[0], segments[1:]

	if seg.name != "" {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%w: %q is not an object", ErrNotFound, seg.name)
		}
		val, ok := obj[seg.name]
		if !ok {
			return nil, fmt.Errorf("%w: field %q not found", ErrNotFound, seg.name)
		}
		current = val
	}

	if !seg.hasIndex {
		return evalSegments(current, rest)
	}

	arr, ok := current.([]any)
	if !ok {
		return nil, fmt.Errorf("%w: expected an array to index into", ErrNotFound)
	}

	if seg.wildcard {
		results := make([]any, 0, len(arr))
		for _, elem := range arr {
			val, err := evalSegments(elem, rest)
			if err != nil {
				continue
			}
			results = append(results, val)
		}
		return results, nil
	}

	if seg.index < 0 || seg.index >= len(arr) {
		return nil, fmt.Errorf("%w: index %d out of range (length %d)", ErrNotFound, seg.index, len(arr))
	}
	return evalSegments(arr[seg.index], rest)
}
