// Pomme opens the packaged player with its environment and dependency checks.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/x/term"
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
	if reopenInLargerTerminal(exe, os.Args[1:]) {
		return
	}
	args := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(filepath.Dir(exe), "Run-Pomme.ps1")}
	args = append(args, os.Args[1:]...)
	cmd := exec.Command("powershell.exe", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err = cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Pomme could not start:", err)
		fmt.Fprintln(os.Stderr, "Press Enter to close.")
		fmt.Scanln()
		os.Exit(1)
	}
}

// reopenInLargerTerminal gives the album-art-and-lyrics layout room to breathe
// when Pomme is launched from a small console window. If Windows Terminal is
// unavailable, continue in the user's current terminal at its existing size.
func reopenInLargerTerminal(exe string, appArgs []string) bool {
	if runtime.GOOS != "windows" || os.Getenv("POMME_SIZED_TERMINAL") == "1" {
		return false
	}
	cols, rows, err := term.GetSize(os.Stdout.Fd())
	if err == nil && cols >= startupColumns && rows >= startupRows {
		return false
	}
	wt, err := exec.LookPath("wt.exe")
	if err != nil {
		return false
	}
	args := []string{"--window", "new", "--size", "140,45", exe}
	args = append(args, appArgs...)
	cmd := exec.Command(wt, args...)
	cmd.Env = append(os.Environ(), "POMME_SIZED_TERMINAL=1")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}
