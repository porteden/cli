//go:build windows

package output

import (
	"os"

	"golang.org/x/sys/windows"
)

// enableVirtualTerminalProcessing turns on ENABLE_VIRTUAL_TERMINAL_PROCESSING
// for stdout so classic Windows consoles interpret ANSI escape sequences
// instead of printing them literally. Returns false when the mode cannot be
// queried or set, in which case the caller disables color.
func enableVirtualTerminalProcessing() bool {
	h := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return false
	}
	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return true // already enabled
	}
	if err := windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		return false
	}
	return true
}
