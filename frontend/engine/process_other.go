//go:build !windows

package engine

import (
	"context"
	"os/exec"
)

func nativeCommand(path string, args ...string) *exec.Cmd { return exec.Command(path, args...) }
func nativeCommandContext(ctx context.Context, path string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, path, args...)
}
