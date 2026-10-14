// Pomme opens the packaged player with its environment and dependency checks.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
