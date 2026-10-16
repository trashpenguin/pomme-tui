package engine

import "time"

type Track struct {
	ID, Title, Artist, Album string
	Kind                     string
	Duration                 time.Duration
	Art                      string
}
type State struct {
	Authed, Playing, Initializing bool
	Pos, Dur                      time.Duration
	Volume                        int
	Shuffle                       bool
	Repeat                        int
	Now                           Track
	Queue                         []Track
	QueuePos                      int
	Err                           string
}
type SearchResults struct{ Songs, Albums, Playlists, Recent, Artists []Track }
type Account struct {
	Name       string `json:"name"`
	Handle     string `json:"handle"`
	Storefront string `json:"storefront"`
	Country    string `json:"country"`
}
type wireTrack struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
	Kind   string `json:"kind"`
	DurMs  uint64 `json:"durMs"`
	Art    string `json:"art"`
}

func (t wireTrack) track() Track {
	return Track{ID: t.ID, Title: t.Title, Artist: t.Artist, Album: t.Album, Kind: t.Kind, Duration: time.Duration(t.DurMs) * time.Millisecond, Art: t.Art}
}
func tracks(items []wireTrack) []Track {
	out := make([]Track, len(items))
	for i, t := range items {
		out[i] = t.track()
	}
	return out
}
