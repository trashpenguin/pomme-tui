package main

import (
	"testing"
	"time"

	"github.com/k1y0miiii/applemusic-tui/engine"
)

func TestListeningHistoryAggregatesAndFiltersRanges(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.Local)
	history := &listeningHistory{Days: make(map[string]map[string]listeningRecord)}
	track := engine.Track{ID: "one", Title: "Blah Blah", Artist: "Joji", Album: "Piss In The Wind"}
	history.record(track, now.Add(-time.Hour), 90*time.Second, true)
	history.record(track, now.Add(-30*time.Minute), 60*time.Second, false)
	history.record(engine.Track{ID: "two", Title: "Old Song", Artist: "Other"}, now.AddDate(0, 0, -40), 4*time.Minute, true)

	got := history.summary(statsLast30, now)
	if got.Plays != 1 || got.Seconds != 150 || got.TodaySeconds != 150 {
		t.Fatalf("last 30-day totals = plays %d, seconds %d, today %d; want 1, 150, 150", got.Plays, got.Seconds, got.TodaySeconds)
	}
	if len(got.Tracks) != 1 || got.Tracks[0].Title != "Blah Blah" || got.Tracks[0].Plays != 1 {
		t.Fatalf("top tracks = %+v, want the recent song", got.Tracks)
	}
	if len(got.Artists) != 1 || got.Artists[0].Title != "Joji" {
		t.Fatalf("top artists = %+v, want Joji", got.Artists)
	}
	if len(got.Albums) != 1 || got.Albums[0].Title != "Piss In The Wind" {
		t.Fatalf("top albums = %+v, want Piss In The Wind", got.Albums)
	}
	all := history.summary(statsAllTime, now)
	if all.Plays != 2 || all.Seconds != 390 {
		t.Fatalf("all-time totals = plays %d, seconds %d; want 2, 390", all.Plays, all.Seconds)
	}
}

func TestListeningHistoryStreakAndPersistence(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.Local)
	history := &listeningHistory{Days: make(map[string]map[string]listeningRecord)}
	track := engine.Track{ID: "one", Title: "Song", Artist: "Artist"}
	for i := 0; i < 4; i++ {
		history.record(track, now.AddDate(0, 0, -i), time.Minute, i == 0)
	}
	current, best := history.streaks(now)
	if current != 4 || best != 4 {
		t.Fatalf("streaks = current %d, best %d; want 4 and 4", current, best)
	}

	t.Setenv("AMTUI_CONFIG_DIR", t.TempDir())
	history.save()
	loaded := loadListeningHistory()
	if got := loaded.summary(statsLast7, now); got.Plays != 1 || got.Seconds != 240 {
		t.Fatalf("persisted history = %+v, want one play over four minutes", got)
	}
}
