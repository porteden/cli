//go:build !windows

package output

// enableVirtualTerminalProcessing is a no-op on non-Windows platforms, where
// terminals interpret ANSI escape sequences natively. The Windows-tagged
// build provides the real implementation.
func enableVirtualTerminalProcessing() bool {
	return true
}
