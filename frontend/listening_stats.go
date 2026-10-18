package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/k1y0miiii/applemusic-tui/engine"
)

const listeningStatsFile = "listening-stats.json"

type listeningRecord struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Artist  string  `json:"artist"`
	Album   string  `json:"album"`
	Plays   int     `json:"plays"`
	Seconds float64 `json:"seconds"`
}

type listeningHistory struct {
	Days map[string]map[string]listeningRecord `json:"days"`
}

type statsRange int

const (
	statsLast7 statsRange = iota
	statsLast30
	statsThisMonth
	statsThisYear
	statsAllTime
)

var statsRangeNames = [...]string{"Last 7 Days", "Last 30 Days", "This Month", "This Year", "All Time"}

type statsSummary struct {
	Plays, Seconds int
	TodaySeconds   int
	Streak, Best   int
	Tracks         []listeningRecord
	Artists        []listeningRecord
	Albums         []listeningRecord
	Days           map[string]int
}

func loadListeningHistory() *listeningHistory {
	history := &listeningHistory{Days: make(map[string]map[string]listeningRecord)}
	path := listeningStatsPath()
	if path == "" {
		return history
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 8<<20 || json.Unmarshal(data, history) != nil || history.Days == nil {
		return &listeningHistory{Days: make(map[string]map[string]listeningRecord)}
	}
	return history
}

func listeningStatsPath() string {
	dir := configDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, listeningStatsFile)
}

func (h *listeningHistory) record(track engine.Track, at time.Time, elapsed time.Duration, newPlay bool) {
	if h == nil || track.ID == "" || elapsed <= 0 {
		return
	}
	if h.Days == nil {
		h.Days = make(map[string]map[string]listeningRecord)
	}
	day := at.Local().Format("2006-01-02")
	if h.Days[day] == nil {
		h.Days[day] = make(map[string]listeningRecord)
	}
	record := h.Days[day][track.ID]
	record.ID, record.Title, record.Artist, record.Album = track.ID, track.Title, track.Artist, track.Album
	if newPlay {
		record.Plays++
	}
	record.Seconds += elapsed.Seconds()
	h.Days[day][track.ID] = record
}

func (h *listeningHistory) save() {
	path := listeningStatsPath()
	if h == nil || path == "" {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0o700) != nil {
		return
	}
	data, err := json.Marshal(h)
	if err != nil || len(data) > 8<<20 {
		return
	}
	tmp := path + ".tmp"
	if os.WriteFile(tmp, data, 0o600) == nil {
		_ = os.Rename(tmp, path)
	}
}

func statsRangeStart(period statsRange, now time.Time) time.Time {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch period {
	case statsLast7:
		return today.AddDate(0, 0, -6)
	case statsLast30:
		return today.AddDate(0, 0, -29)
	case statsThisMonth:
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	case statsThisYear:
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	default:
		return time.Time{}
	}
}

func (h *listeningHistory) summary(period statsRange, now time.Time) statsSummary {
	result := statsSummary{Days: make(map[string]int)}
	if h == nil {
		return result
	}
	start := statsRangeStart(period, now)
	tracks := make(map[string]listeningRecord)
	artists := make(map[string]listeningRecord)
	albums := make(map[string]listeningRecord)
	for day, entries := range h.Days {
		date, err := time.ParseInLocation("2006-01-02", day, now.Location())
		if err != nil || (!start.IsZero() && date.Before(start)) {
			continue
		}
		for _, entry := range entries {
			seconds := int(entry.Seconds)
			result.Plays += entry.Plays
			result.Seconds += seconds
			result.Days[day] += seconds
			mergeListenRecord(tracks, entry.ID, entry)
			if entry.Artist != "" {
				artist := entry
				artist.Title, artist.Artist = entry.Artist, ""
				mergeListenRecord(artists, entry.Artist, artist)
			}
			if entry.Album != "" {
				album := entry
				album.Title, album.Artist = entry.Album, ""
				mergeListenRecord(albums, entry.Album, album)
			}
		}
	}
	today := now.Local().Format("2006-01-02")
	result.TodaySeconds = result.Days[today]
	result.Tracks = topListenRecords(tracks)
	result.Artists = topListenRecords(artists)
	result.Albums = topListenRecords(albums)
	result.Streak, result.Best = h.streaks(now)
	return result
}

func mergeListenRecord(dst map[string]listeningRecord, key string, value listeningRecord) {
	current := dst[key]
	if current.ID == "" {
		current = listeningRecord{ID: key, Title: value.Title, Artist: value.Artist, Album: value.Album}
	}
	current.Plays += value.Plays
	current.Seconds += value.Seconds
	dst[key] = current
}

func topListenRecords(records map[string]listeningRecord) []listeningRecord {
	result := make([]listeningRecord, 0, len(records))
	for _, record := range records {
		result = append(result, record)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Plays == result[j].Plays {
			return result[i].Seconds > result[j].Seconds
		}
		return result[i].Plays > result[j].Plays
	})
	if len(result) > 5 {
		result = result[:5]
	}
	return result
}

func (h *listeningHistory) streaks(now time.Time) (current, best int) {
	if h == nil {
		return 0, 0
	}
	var dates []time.Time
	for day, entries := range h.Days {
		listened := false
		for _, item := range entries {
			if item.Seconds > 0 {
				listened = true
				break
			}
		}
		if date, err := time.ParseInLocation("2006-01-02", day, now.Location()); err == nil && listened {
			dates = append(dates, date)
		}
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	for i, date := range dates {
		if i == 0 || !dates[i-1].AddDate(0, 0, 1).Equal(date) {
			current = 1
		} else {
			current++
		}
		if current > best {
			best = current
		}
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	anchor := today
	if _, ok := h.Days[today.Format("2006-01-02")]; !ok {
		anchor = today.AddDate(0, 0, -1)
	}
	current = 0
	for day := anchor; ; day = day.AddDate(0, 0, -1) {
		entries, ok := h.Days[day.Format("2006-01-02")]
		if !ok || !hasListening(entries) {
			break
		}
		current++
	}
	return current, best
}

func hasListening(entries map[string]listeningRecord) bool {
	for _, entry := range entries {
		if entry.Seconds > 0 {
			return true
		}
	}
	return false
}
