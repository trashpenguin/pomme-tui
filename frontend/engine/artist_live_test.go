package engine

import (
	"context"
	"encoding/json"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Explicit opt-in: read-only catalogue checks in a temporary profile. Never
// opens login or plays audio, and never changes the saved account/library.
func TestNativeLiveArtistPicturesAndCatalogue(t *testing.T) {
	if os.Getenv("NATIVE_ARTIST_LIVE_TEST") != "1" {
		t.Skip("requires explicit opt-in and a saved test subscription")
	}
	profile := os.Getenv("POMME_DATA_DIR")
	credentials, err := os.ReadFile(filepath.Join(profile, "cache", "apple", "credentials.json"))
	if err != nil {
		t.Fatal("saved test account is unavailable")
	}
	isolated := t.TempDir()
	account := filepath.Join(isolated, "cache", "apple")
	if err := os.MkdirAll(account, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(account, "credentials.json"), credentials, 0600); err != nil {
		t.Fatal("could not prepare isolated test account")
	}
	t.Setenv("POMME_DATA_DIR", isolated)
	t.Setenv("SONORA_APPLE_MEDIA_USER_TOKEN", "")
	t.Setenv("SONORA_APPLE_BEARER_TOKEN", "")
	path, err := backendPath()
	if err != nil {
		t.Fatal(err)
	}
	eng, err := start(path)
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	var hello struct {
		Protocol int
		Authed   bool
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := eng.rpc.call(ctx, "hello", nil, &hello); err != nil {
		t.Fatal("native startup handshake failed; check the test runtime/CDM environment")
	}
	if !hello.Authed {
		t.Fatal("test account is not signed in; no login window was opened")
	}
	result, err := eng.Search("Modern Baseball")
	if err != nil || len(result.Songs) == 0 {
		t.Fatal("live catalogue song search failed")
	}
	artistFound := false
	for _, candidate := range result.Artists {
		if candidate.ID != "" && candidate.Kind == "artist" && strings.EqualFold(candidate.Title, "Modern Baseball") {
			artistFound = true
		}
	}
	if !artistFound {
		t.Fatal("artist search did not return the requested real catalog artist")
	}
	var song Track
	for _, candidate := range result.Songs {
		if strings.EqualFold(candidate.Artist, "Modern Baseball") {
			song = candidate
			break
		}
	}
	if song.ID == "" {
		t.Fatal("the chosen test artist was not present in catalogue search")
	}
	refs, err := eng.SongArtists(song.ID)
	if err != nil || len(refs.Artists) == 0 {
		t.Fatal("song-credit artist resolution failed")
	}
	var artist Track
	for _, candidate := range refs.Artists {
		if strings.EqualFold(candidate.Title, "Modern Baseball") {
			artist = candidate
			break
		}
	}
	if artist.ID == "" || artist.ID == song.ID || artist.Kind != "artist" {
		t.Fatal("song credits did not return a real artist ID")
	}
	detail, err := eng.ArtistPage(artist.ID)
	if err != nil {
		t.Fatal("live artist catalogue request failed")
	}
	if len(detail.Artists) == 0 || len(detail.Songs) == 0 || len(detail.Albums) == 0 {
		t.Fatal("artist page is missing profile, top songs or albums")
	}
	if detail.Artists[0].Art != "" {
		artist = detail.Artists[0]
	}
	if artist.Art == "" {
		t.Fatal("test artist portrait URL is missing")
	}
	url := strings.NewReplacer("{w}", "96", "{h}", "96", "{f}", "jpg").Replace(artist.Art)
	url = regexp.MustCompile(`\{[^{}]*\}`).ReplaceAllString(url, "")
	imageCtx, imageCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer imageCancel()
	request, err := http.NewRequestWithContext(imageCtx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatal("invalid artist picture URL")
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal("artist portrait download failed")
	}
	defer response.Body.Close()
	bytes, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil || response.StatusCode != http.StatusOK {
		t.Fatal("artist portrait response failed")
	}
	cfg, _, err := image.DecodeConfig(strings.NewReader(string(bytes)))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 {
		t.Fatal("artist portrait is not a valid image")
	}
	albumSongs, err := eng.DiscoveryTracks("album", detail.Albums[0].ID)
	if err != nil || len(albumSongs) == 0 {
		t.Fatal("artist album did not open its songs")
	}

	resolvedAlbum, err := eng.SongAlbum(song.ID)
	if err != nil || len(resolvedAlbum.Albums) != 1 || resolvedAlbum.Albums[0].Kind != "album" || resolvedAlbum.Albums[0].ID == "" {
		t.Fatal("song album reference lookup failed")
	}
	resolvedSongs, err := eng.DiscoveryTracks("album", resolvedAlbum.Albums[0].ID)
	if err != nil {
		t.Fatal("resolved history album tracks failed")
	}
	foundSource := false
	for _, candidate := range resolvedSongs {
		if candidate.ID == song.ID {
			foundSource = true
		}
	}
	if !foundSource {
		t.Fatal("resolved album did not contain the original song")
	}
	t.Logf("History song resolved to its real album and %d album tracks; original song verified. No audio played.", len(resolvedSongs))
	library, err := eng.LibraryPage("artists", 0)
	if err != nil {
		t.Fatal("live Library artist page failed")
	}
	portraits := 0
	for _, item := range library.Artists {
		if item.Art != "" {
			portraits++
		}
	}
	t.Logf("Artist references, portrait (%dx%d), %d top songs, %d albums and %d album tracks verified; library returned %d artists (%d portraits). No audio played.", cfg.Width, cfg.Height, len(detail.Songs), len(detail.Albums), len(albumSongs), len(library.Artists), portraits)
	if folder := os.Getenv("NATIVE_ARTIST_SNAPSHOT_DIR"); folder != "" {
		if err := os.MkdirAll(folder, 0700); err != nil {
			t.Fatal(err)
		}
		snapshot := struct {
			Artist         Track
			Page           SearchResults
			AlbumTracks    []Track
			LibraryArtists []Track
		}{artist, detail, albumSongs, library.Artists[:min(6, len(library.Artists))]}
		data, err := json.Marshal(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, "artist-catalogue.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, "artist-photo.img"), bytes, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
