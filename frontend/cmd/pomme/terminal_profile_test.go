package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTerminalProfileHasPommeIconAndPreservesOtherSettings(t *testing.T) {
	root := t.TempDir()
	settings := filepath.Join(root, "Packages", "Microsoft.WindowsTerminal_8wekyb3d8bbwe", "LocalState", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0700); err != nil {
		t.Fatal(err)
	}
	original := []byte("// user settings\n{\"defaultProfile\":\"keep\"}\n")
	if err := os.WriteFile(settings, original, 0600); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(root, "App With Spaces", "Pomme.exe")
	changed, err := ensureTerminalProfile(root, exe)
	if err != nil || !changed {
		t.Fatalf("profile registration failed: %v", err)
	}
	folder := filepath.Join(root, "Microsoft", "Windows Terminal", "Fragments", "PommeTUI")
	raw, err := os.ReadFile(filepath.Join(folder, "pomme.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fragment struct {
		Profiles []struct{ Guid, Name, Icon, Commandline string }
	}
	if err = json.Unmarshal(raw, &fragment); err != nil {
		t.Fatal(err)
	}
	if len(fragment.Profiles) != 1 || fragment.Profiles[0].Guid != pommeProfileGUID || fragment.Profiles[0].Name != "Pomme" {
		t.Fatal("missing stable Pomme profile")
	}
	profile := fragment.Profiles[0]
	if profile.Icon != filepath.Join(folder, "pomme.png") || !strings.Contains(profile.Commandline, exe) || !strings.Contains(profile.Commandline, terminalChildFlag) {
		t.Fatal("profile missing icon or safe launcher command")
	}
	icon, err := os.ReadFile(profile.Icon)
	if err != nil || len(icon) < 8 || string(icon[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("profile icon is not bundled PNG")
	}
	after, _ := os.ReadFile(settings)
	if string(after) != string(original) {
		t.Fatal("registration overwrote user Terminal settings")
	}
	changed, err = ensureTerminalProfile(root, exe)
	if err != nil || changed {
		t.Fatal("unchanged profile was rewritten")
	}
}
func TestTerminalProfileLaunchArgumentsPreserveUserArguments(t *testing.T) {
	exe := `C:\App With Spaces\Pomme.exe`
	args := pommeTerminalArgs(exe, []string{"play", "-Output", "Speakers With Spaces"})
	found := strings.Join(args, "|")
	if !strings.Contains(found, "new-tab|--profile|"+pommeProfileGUID) || !strings.Contains(found, exe+"|"+terminalChildFlag+"|play|-Output|Speakers With Spaces") {
		t.Fatalf("invalid terminal arguments: %#v", args)
	}
	clean, child := launcherArgs([]string{terminalChildFlag, "doctor"})
	if !child || len(clean) != 1 || clean[0] != "doctor" {
		t.Fatal("internal terminal marker leaked to command")
	}
	for _, command := range []string{"version", "doctor", "help", "lastfm-auth"} {
		if shouldOpenPommeTerminal([]string{command}, false, "") {
			t.Fatal("non-player command opened a Terminal window")
		}
	}
	if shouldOpenPommeTerminal(nil, true, "") || shouldOpenPommeTerminal(nil, false, pommeProfileGUID) {
		t.Fatal("launcher would recurse")
	}
}

// Explicit opt-in: verifies the real installed host accepts the Pomme profile.
// Registers only Pomme's fragment and runs a no-account, no-audio probe that exits.
func TestTerminalProfileHostIntegration(t *testing.T) {
	exe := os.Getenv("POMME_TERMINAL_ICON_EXE")
	if exe == "" {
		t.Skip("explicit Terminal host integration opt-in")
	}
	if _, err := os.Stat(exe); err != nil {
		t.Fatal(err)
	}
	changed, err := ensureTerminalProfile(os.Getenv("LOCALAPPDATA"), exe)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		time.Sleep(750 * time.Millisecond)
	}
	dir := filepath.Join(t.TempDir(), "Probe With Spaces")
	if err = os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "icon probe.ps1")
	output := filepath.Join(dir, "profile.txt")
	code := `param([string]$Destination,[string]$Launcher,[string]$ArgumentProbe)
$version = & $Launcher --pomme-terminal-child version
if ($LASTEXITCODE -ne 0) { exit 1 }
[IO.File]::WriteAllText($Destination,($env:WT_PROFILE_ID+[char]10+$ArgumentProbe+[char]10+$version))
exit 0
`
	if err = os.WriteFile(script, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("wt.exe", "--window", "new", "--size", "80,12", "new-tab", "--profile", pommeProfileGUID, "--title", "Pomme icon check", "--", "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script, output, exe, "Argument With Spaces")
	if err = cmd.Run(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(output)
		if err == nil && len(data) > 0 {
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			if len(lines) != 3 || lines[1] != "Argument With Spaces" || !strings.Contains(lines[2], "Pomme TUI") {
				t.Fatalf("host lost arguments or child launcher failed: %s", data)
			}
			if !strings.EqualFold(strings.TrimSpace(lines[0]), pommeProfileGUID) {
				t.Fatalf("host selected wrong profile: %s", data)
			}
			t.Log("Windows Terminal accepted the Pomme profile with its existing apple PNG icon; probe exited without account/audio.")
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("Terminal profile probe did not finish; no existing Terminal window was closed")
}

func TestDiagnosticLauncherCommandsNeverPause(t *testing.T) {
	for _, command := range []string{"version", "doctor", "help", "lastfm-auth"} {
		if launcherShouldPause([]string{command}) {
			t.Fatal("diagnostic command waits for keyboard input")
		}
	}
	args, child := launcherArgs([]string{"--version"})
	if child || len(args) != 1 || args[0] != "version" {
		t.Fatal("version alias not normalized")
	}
}
