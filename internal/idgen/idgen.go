// Package idgen generates identifiers for entities stored by ByteForge
// (collections, requests, environments, runs). These only need to be
// unique within a single user's local database, not globally, so a short
// random hex string is sufficient and avoids pulling in a UUID dependency
// for something this small.
package idgen

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns a random 16-character hex identifier.
func New() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		// The OS entropy source failing is not something callers can
		// meaningfully recover from; there is no safe fallback that
		// preserves the "unique enough" guarantee this package exists for.
		panic("idgen: failed to read random bytes: " + err.Error())
	}
	return hex.EncodeToString(buf)
}
