package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func rememberSearch(history []string, query string) []string {
	query = strings.TrimSpace(query)
	if query == "" || len(query) > 512 {
		return history
	}
	result := []string{query}
	for _, old := range history {
		if !strings.EqualFold(old, query) && len(result) < 30 {
			result = append(result, old)
		}
	}
	return result
}
func loadSearchHistory() []string {
	file, err := os.Open(filepath.Join(configDir(), "search-history.json"))
	if err != nil {
		return nil
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.Size() > 32768 {
		return nil
	}
	var history []string
	if json.NewDecoder(file).Decode(&history) != nil {
		return nil
	}
	var clean []string
	for i := len(history) - 1; i >= 0; i-- {
		clean = rememberSearch(clean, history[i])
	}
	return clean
}
func saveSearchHistory(history []string) {
	dir := configDir()
	if dir == "" {
		return
	}
	if os.MkdirAll(dir, 0700) != nil {
		return
	}
	data, err := json.Marshal(history)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, "search-history.json"), data, 0600)
}
