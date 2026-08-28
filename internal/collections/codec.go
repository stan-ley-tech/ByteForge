package collections

import (
	"encoding/json"
	"fmt"
	"io"
)

// Decode reads a collection from JSON, fills in any missing IDs, and
// validates its structure. This is the single entry point for both
// "byteforge run collection.json" and the API's collection import endpoint,
// so both paths reject malformed input the same way.
func Decode(r io.Reader) (*Collection, error) {
	var c Collection
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("collections: decode: %w", err)
	}

	c.AssignMissingIDs()
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Encode writes c as indented JSON, safe to hand to a user for export or
// commit to source control. Collections never carry resolved secret values
// (auth and body fields hold {{VARIABLE}} references, not literals), so no
// redaction step is needed here — the safety boundary lives in the
// environments package, at the point secrets actually exist.
func Encode(w io.Writer, c *Collection) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(c); err != nil {
		return fmt.Errorf("collections: encode: %w", err)
	}
	return nil
}
