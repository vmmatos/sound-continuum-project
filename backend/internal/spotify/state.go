package spotify

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// stateExpiry bounds how long a generated OAuth state stays valid — long
// enough for a human to complete Spotify's consent screen, short enough
// that a stale state can't be replayed much later.
const stateExpiry = 10 * time.Minute

// stateGuard tracks the single pending OAuth state for Sound Continuum's one
// curator. Only one authorization flow is ever in flight at a time, so a
// single mutex-guarded value is enough — no map, no generic store.
type stateGuard struct {
	mu      sync.Mutex
	value   string
	expires time.Time
}

// generate creates a cryptographically secure state value, stores it as the
// pending value (replacing any prior one), and returns it.
func (g *stateGuard) generate() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	value := hex.EncodeToString(buf)

	g.mu.Lock()
	g.value = value
	g.expires = time.Now().Add(stateExpiry)
	g.mu.Unlock()

	return value, nil
}

// consume reports whether candidate matches the current pending state and
// hasn't expired. It clears the pending state either way, so a state value
// can only ever be used once.
func (g *stateGuard) consume(candidate string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	value, expires := g.value, g.expires
	g.value = ""
	g.expires = time.Time{}

	if candidate == "" || value == "" {
		return false
	}
	if time.Now().After(expires) {
		return false
	}
	return candidate == value
}
