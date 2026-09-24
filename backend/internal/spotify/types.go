package spotify

// Paging mirrors Spotify's current pagination envelope, shared by
// GET /me/playlists, GET /playlists/{id}/items, and each object type
// inside GET /search.
type Paging[T any] struct {
	Items    []T    `json:"items"`
	Total    int    `json:"total"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
}

// Playlist is the subset of a Spotify playlist object this MVP needs to
// list the curator's playlists. Confirmed live: the playlist's item-count
// summary is returned under the "items" key, not the historical "tracks" —
// the same /tracks-to-/items rename applies to this field, not just the
// endpoint path.
type Playlist struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	URI         string `json:"uri"`
	Public      bool   `json:"public"`
	Owner       struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"owner"`
	Items struct {
		Total int `json:"total"`
	} `json:"items"`
}

// PlaylistItem is one entry from GET /playlists/{id}/items. Confirmed
// live: the track payload is nested under the "item" key, not "track".
type PlaylistItem struct {
	AddedAt string `json:"added_at"`
	Track   Track  `json:"item"`
}

// Track is the subset of a Spotify track object this MVP needs. Fields
// removed from the current API (popularity, available_markets) are
// intentionally omitted rather than left unused.
type Track struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	URI        string   `json:"uri"`
	DurationMS int      `json:"duration_ms"`
	Artists    []Artist `json:"artists"`
}

// Artist is the subset of a Spotify artist object referenced from a track.
type Artist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URI  string `json:"uri"`
}

// SearchResult carries only the object types Sound Continuum's curation
// workflow needs. Album/show/episode/audiobook are omitted — add if a
// future card needs them.
type SearchResult struct {
	Tracks    *Paging[Track]    `json:"tracks,omitempty"`
	Artists   *Paging[Artist]   `json:"artists,omitempty"`
	Playlists *Paging[Playlist] `json:"playlists,omitempty"`
}
