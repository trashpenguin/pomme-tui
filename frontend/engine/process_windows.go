//go:build windows

package engine

import (
	"context"
	"os/exec"
	"syscall"
)

// CREATE_NO_WINDOW suppresses a helper console while allowing the explicit GUI sign-in window.
func nativeCommand(path string, args ...string) *exec.Cmd {
	cmd := exec.Command(path, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	return cmd
}
func nativeCommandContext(ctx context.Context, path string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	return cmd
}
