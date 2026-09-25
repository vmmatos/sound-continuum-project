package spotify

import "encoding/json"

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

// Image is a Spotify image reference (playlist cover art, etc.), smallest
// useful subset of Spotify's image object.
type Image struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}

// Playlist is the subset of a Spotify playlist object this MVP needs to
// list and inspect the curator's playlists. Confirmed live: the playlist's
// item-count summary is returned under the "items" key, not the historical
// "tracks" — the same /tracks-to-/items rename applies to this field, not
// just the endpoint path.
type Playlist struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	URI           string `json:"uri"`
	Href          string `json:"href"`
	Public        bool   `json:"public"`
	Collaborative bool   `json:"collaborative"`
	SnapshotID    string `json:"snapshot_id"`
	Owner         struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"owner"`
	Items struct {
		Total int `json:"total"`
	} `json:"items"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
	Images []Image `json:"images"`
}

// Episode is the subset of a Spotify episode object this MVP needs when a
// playlist item is a podcast episode rather than a track.
type Episode struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	URI        string `json:"uri"`
	DurationMS int    `json:"duration_ms"`
}

// PlaylistItem is one entry from GET /playlists/{id}/items. Confirmed
// live: the payload is nested under the "item" key, not "track". A
// playlist item can be a track, an episode, or unavailable (item is
// null — e.g. removed content) — ItemType records which, and only the
// matching field is populated. Never assume Track is set.
type PlaylistItem struct {
	AddedAt string
	AddedBy struct {
		ID string `json:"id"`
	}
	IsLocal  bool
	ItemType string // "track", "episode", or "unavailable"
	Track    *Track
	Episode  *Episode
}

func (p *PlaylistItem) UnmarshalJSON(data []byte) error {
	var raw struct {
		AddedAt string `json:"added_at"`
		AddedBy struct {
			ID string `json:"id"`
		} `json:"added_by"`
		IsLocal bool            `json:"is_local"`
		Item    json.RawMessage `json:"item"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	p.AddedAt = raw.AddedAt
	p.AddedBy = raw.AddedBy
	p.IsLocal = raw.IsLocal

	if len(raw.Item) == 0 || string(raw.Item) == "null" {
		p.ItemType = "unavailable"
		return nil
	}

	var typed struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw.Item, &typed); err != nil {
		return err
	}

	switch typed.Type {
	case "episode":
		var ep Episode
		if err := json.Unmarshal(raw.Item, &ep); err != nil {
			return err
		}
		p.ItemType = "episode"
		p.Episode = &ep
	default:
		var tr Track
		if err := json.Unmarshal(raw.Item, &tr); err != nil {
			return err
		}
		p.ItemType = "track"
		p.Track = &tr
	}
	return nil
}

func (p PlaylistItem) MarshalJSON() ([]byte, error) {
	out := struct {
		AddedAt string `json:"added_at"`
		AddedBy struct {
			ID string `json:"id"`
		} `json:"added_by"`
		IsLocal  bool     `json:"is_local"`
		ItemType string   `json:"type"`
		Track    *Track   `json:"track,omitempty"`
		Episode  *Episode `json:"episode,omitempty"`
	}{
		AddedAt:  p.AddedAt,
		AddedBy:  p.AddedBy,
		IsLocal:  p.IsLocal,
		ItemType: p.ItemType,
		Track:    p.Track,
		Episode:  p.Episode,
	}
	return json.Marshal(out)
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
