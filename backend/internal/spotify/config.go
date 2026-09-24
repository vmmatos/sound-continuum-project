// Package spotify implements Spotify OAuth (Authorization Code flow) and
// token lifecycle management for the single Sound Continuum curator.
package spotify

// Config holds the Spotify application credentials and redirect URI. It is
// loaded from the environment by the caller (cmd/server), not read directly
// from os.Getenv here, so the package stays testable without env mutation.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// valid reports whether every field required to start an OAuth flow is set.
// dev/.secrets.env ships with empty Spotify credentials until a developer
// fills them in, so an invalid Config is an expected, non-fatal state.
func (c Config) valid() bool {
	return c.ClientID != "" && c.ClientSecret != "" && c.RedirectURI != ""
}
