package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	brand "github.com/k1y0miiii/applemusic-tui/assets"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const pommeProfileGUID = "{d92dc45d-0481-50e7-b1e2-c4b1d70ffd05}"
const terminalChildFlag = "--pomme-terminal-child"

func launcherArgs(args []string) ([]string, bool) {
	child := len(args) > 0 && args[0] == terminalChildFlag
	if child {
		args = args[1:]
	}
	if len(args) == 1 && (args[0] == "--version" || args[0] == "-v") {
		args = []string{"version"}
	}
	return args, child
}
func shouldOpenPommeTerminal(args []string, child bool, currentProfile string) bool {
	if child || strings.EqualFold(currentProfile, pommeProfileGUID) {
		return false
	}
	if len(args) > 0 {
		switch args[0] {
		case "version", "doctor", "help", "lastfm-auth", "--version", "-v":
			return false
		}
	}
	return true
}
func pommeTerminalArgs(exe string, args []string) []string {
	result := []string{"--window", "new", "--size", "140,45", "new-tab", "--profile", pommeProfileGUID, "--title", "Pomme", "--", exe, terminalChildFlag}
	return append(result, args...)
}

// Register only an application-owned fragment. Never parse or rewrite the user's
// main Terminal settings; a timestamp notification asks an existing host to reload.
func ensureTerminalProfile(localData, exe string) (bool, error) {
	if !filepath.IsAbs(localData) || !filepath.IsAbs(exe) || strings.ContainsAny(exe, "\";\r\n") {
		return false, fmt.Errorf("invalid terminal profile location")
	}
	folder := filepath.Join(localData, "Microsoft", "Windows Terminal", "Fragments", "PommeTUI")
	if err := os.MkdirAll(folder, 0700); err != nil {
		return false, err
	}
	profile := map[string]any{"guid": pommeProfileGUID, "name": "Pomme", "icon": filepath.Join(folder, "pomme.png"), "commandline": "\"" + exe + "\" " + terminalChildFlag, "startingDirectory": filepath.Dir(exe), "closeOnExit": "graceful"}
	data, err := json.MarshalIndent(map[string]any{"profiles": []any{profile}}, "", "  ")
	if err != nil {
		return false, err
	}
	data = append(data, '\n')
	iconChanged, err := writeProfileFile(filepath.Join(folder, "pomme.png"), brand.Logo)
	if err != nil {
		return false, err
	}
	jsonChanged, err := writeProfileFile(filepath.Join(folder, "pomme.json"), data)
	if err != nil {
		return false, err
	}
	changed := iconChanged || jsonChanged
	if changed {
		for _, path := range []string{
			filepath.Join(localData, "Packages", "Microsoft.WindowsTerminal_8wekyb3d8bbwe", "LocalState", "settings.json"),
			filepath.Join(localData, "Packages", "Microsoft.WindowsTerminalPreview_8wekyb3d8bbwe", "LocalState", "settings.json"),
			filepath.Join(localData, "Microsoft", "Windows Terminal", "settings.json"),
		} {
			if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
				_ = os.Chtimes(path, time.Now(), time.Now())
			}
		}
	}
	return changed, nil
}
func writeProfileFile(path string, data []byte) (bool, error) {
	old, err := os.ReadFile(path)
	if err == nil && bytes.Equal(old, data) {
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".pomme-*.tmp")
	if err != nil {
		return false, err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err = file.Write(data); err != nil {
		file.Close()
		return false, err
	}
	if err = file.Close(); err != nil {
		return false, err
	}
	if err = os.Rename(name, path); err != nil {
		return false, err
	}
	return true, nil
}

func launcherShouldPause(args []string) bool {
	if len(args) == 0 {
		return true
	}
	return args[0] == "play" || strings.HasPrefix(args[0], "-Output") || args[0] == "-DefaultOutput"
}
