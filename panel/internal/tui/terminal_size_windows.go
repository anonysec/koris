//go:build windows

package tui

import "errors"

// terminalSize is a stub on Windows that always returns an error,
// causing getTerminalSize to fall back to 80x24.
func terminalSize(fd uintptr) (width, height int, err error) {
	return 0, 0, errors.New("terminal size detection not supported on Windows")
}
