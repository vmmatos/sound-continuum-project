package spotifymock

import (
	"context"
	"testing"
)

func TestSearchEchoesQueryAsName(t *testing.T) {
	c := NewCatalogue()
	result, err := c.Search(context.Background(), "Kate Bush", "artist", 5, 0)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if result.Artists == nil || len(result.Artists.Items) != 1 {
		t.Fatalf("expected exactly one artist, got %+v", result.Artists)
	}
	if result.Artists.Items[0].Name != "Kate Bush" {
		t.Errorf("Name = %q, want %q (resolveArtist requires an exact match)", result.Artists.Items[0].Name, "Kate Bush")
	}
}

func TestSearchIsDeterministic(t *testing.T) {
	c := NewCatalogue()
	r1, err := c.Search(context.Background(), "Some Artist", "artist", 5, 0)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	r2, err := c.Search(context.Background(), "Some Artist", "artist", 5, 0)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if r1.Artists.Items[0].ID != r2.Artists.Items[0].ID {
		t.Errorf("Search(%q) returned different IDs across calls: %q vs %q", "Some Artist", r1.Artists.Items[0].ID, r2.Artists.Items[0].ID)
	}
}

func TestSearchDifferentQueriesYieldDifferentArtists(t *testing.T) {
	c := NewCatalogue()
	r1, _ := c.Search(context.Background(), "Artist One", "artist", 5, 0)
	r2, _ := c.Search(context.Background(), "Artist Two", "artist", 5, 0)
	if r1.Artists.Items[0].ID == r2.Artists.Items[0].ID {
		t.Error("different queries produced the same artist ID")
	}
}

func TestArtistAlbumsIsDeterministicAndBounded(t *testing.T) {
	c := NewCatalogue()
	ctx := context.Background()

	page1, err := c.ArtistAlbums(ctx, "mock-artist-aaaa", 10, 0)
	if err != nil {
		t.Fatalf("ArtistAlbums returned error: %v", err)
	}
	page2, err := c.ArtistAlbums(ctx, "mock-artist-aaaa", 10, 0)
	if err != nil {
		t.Fatalf("ArtistAlbums returned error: %v", err)
	}
	if len(page1.Items) == 0 {
		t.Fatal("expected at least one album")
	}
	if len(page1.Items) > 5 {
		t.Errorf("len(Items) = %d, expected a small bounded set", len(page1.Items))
	}
	for i := range page1.Items {
		if page1.Items[i].ID != page2.Items[i].ID {
			t.Errorf("ArtistAlbums is not deterministic: %q vs %q", page1.Items[i].ID, page2.Items[i].ID)
		}
	}
}

func TestAlbumTracksIsDeterministicAndBounded(t *testing.T) {
	c := NewCatalogue()
	ctx := context.Background()

	page1, err := c.AlbumTracks(ctx, "mock-album-aaaa", 50, 0)
	if err != nil {
		t.Fatalf("AlbumTracks returned error: %v", err)
	}
	page2, err := c.AlbumTracks(ctx, "mock-album-aaaa", 50, 0)
	if err != nil {
		t.Fatalf("AlbumTracks returned error: %v", err)
	}
	if len(page1.Items) == 0 {
		t.Fatal("expected at least one track")
	}
	if len(page1.Items) > 10 {
		t.Errorf("len(Items) = %d, expected a small bounded set", len(page1.Items))
	}
	for i := range page1.Items {
		if page1.Items[i].ID != page2.Items[i].ID {
			t.Errorf("AlbumTracks is not deterministic: %q vs %q", page1.Items[i].ID, page2.Items[i].ID)
		}
	}
}

func TestTrackMatchesWhatAlbumTracksProduced(t *testing.T) {
	c := NewCatalogue()
	ctx := context.Background()

	page, err := c.AlbumTracks(ctx, "mock-album-bbbb", 50, 0)
	if err != nil {
		t.Fatalf("AlbumTracks returned error: %v", err)
	}
	if len(page.Items) == 0 {
		t.Fatal("expected at least one track")
	}
	want := page.Items[0]

	got, err := c.Track(ctx, want.ID)
	if err != nil {
		t.Fatalf("Track returned error: %v", err)
	}
	if got.ID != want.ID || got.Name != want.Name || got.DurationMS != want.DurationMS {
		t.Errorf("Track(%q) = %+v, want it to match AlbumTracks' own entry %+v", want.ID, got, want)
	}
}

func TestTrackIsDeterministic(t *testing.T) {
	c := NewCatalogue()
	ctx := context.Background()

	t1, err := c.Track(ctx, "mock-track-cccc")
	if err != nil {
		t.Fatalf("Track returned error: %v", err)
	}
	t2, err := c.Track(ctx, "mock-track-cccc")
	if err != nil {
		t.Fatalf("Track returned error: %v", err)
	}
	if t1.Name != t2.Name || t1.DurationMS != t2.DurationMS {
		t.Errorf("Track is not deterministic: %+v vs %+v", t1, t2)
	}
}

func TestOfficialPlaylistIsStable(t *testing.T) {
	c := NewCatalogue()
	ctx := context.Background()

	p1, err := c.OfficialPlaylist(ctx)
	if err != nil {
		t.Fatalf("OfficialPlaylist returned error: %v", err)
	}
	p2, err := c.OfficialPlaylist(ctx)
	if err != nil {
		t.Fatalf("OfficialPlaylist returned error: %v", err)
	}
	if *p1 != *p2 {
		t.Errorf("OfficialPlaylist is not stable: %+v vs %+v", p1, p2)
	}
	if p1.SpotifyPlaylistID == "" {
		t.Error("expected a non-empty playlist ID")
	}
}

func TestPlaylistItemsIsEmpty(t *testing.T) {
	c := NewCatalogue()
	page, err := c.PlaylistItems(context.Background(), "mock-playlist-001", 50, 0)
	if err != nil {
		t.Fatalf("PlaylistItems returned error: %v", err)
	}
	if len(page.Items) != 0 || page.Total != 0 {
		t.Errorf("expected an empty playlist, got %+v", page)
	}
}
