// Pomme opens the packaged player with its environment and dependency checks.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"strings"
	"time"
)

const (
	startupColumns = 140
	startupRows    = 45
)

func main() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	appArgs, terminalChild := launcherArgs(os.Args[1:])
	if reopenInPommeTerminal(exe, appArgs, terminalChild) {
		return
	}
	args := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(filepath.Dir(exe), "Run-Pomme.ps1")}
	args = append(args, appArgs...)
	cmd := exec.Command("powershell.exe", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err = cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Pomme could not start:", err)
		if launcherShouldPause(appArgs) {
			fmt.Fprintln(os.Stderr, "Press Enter to close.")
			fmt.Scanln()
		}
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() > 0 {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

// Use a Pomme-specific Terminal profile so its tab displays the Pomme icon.
// Other terminal hosts and non-player CLI commands keep their normal behavior.
func reopenInPommeTerminal(exe string, appArgs []string, child bool) bool {
	if runtime.GOOS != "windows" || !shouldOpenPommeTerminal(appArgs, child, os.Getenv("WT_PROFILE_ID")) {
		return false
	}
	for _, arg := range append([]string{exe}, appArgs...) {
		if strings.ContainsAny(arg, ";\r\n") {
			return false
		}
	}
	wt, err := exec.LookPath("wt.exe")
	if err != nil {
		return false
	}
	changed, err := ensureTerminalProfile(os.Getenv("LOCALAPPDATA"), exe)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Pomme: could not register Terminal logo:", err)
		return false
	}
	if changed {
		time.Sleep(750 * time.Millisecond)
	}
	cmd := exec.Command(wt, pommeTerminalArgs(exe, appArgs)...)
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}
