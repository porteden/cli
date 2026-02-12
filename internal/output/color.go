package output

import (
	"os"
	"runtime"

	"golang.org/x/term"
)

// ANSI color codes
const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	Gray    = "\033[90m"
	Bold    = "\033[1m"
)

var colorsEnabled = true

func init() {
	colorsEnabled = supportsColor()
}

// supportsColor checks if the terminal supports colors
func supportsColor() bool {
	// Respect NO_COLOR environment variable (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check FORCE_COLOR for explicit enable
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}

	// Check if stdout is a terminal
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}

	// On Windows, check for modern terminal support
	if runtime.GOOS == "windows" {
		// Windows Terminal and modern PowerShell support ANSI
		if os.Getenv("WT_SESSION") != "" || os.Getenv("TERM_PROGRAM") != "" {
			return true
		}
		// Check for ConEmu, cmder, etc.
		if os.Getenv("ConEmuANSI") == "ON" {
			return true
		}
		// Default: assume modern Windows 10+ with VT support
		return true
	}

	// Unix-like systems generally support colors in terminals
	return true
}

// SetColorEnabled allows overriding color detection
func SetColorEnabled(enabled bool) {
	colorsEnabled = enabled
}

// Colorize wraps text with color codes if colors are enabled
func Colorize(color, text string) string {
	if !colorsEnabled {
		return text
	}
	return color + text + Reset
}

// Helper functions for common colors
func ColorRed(text string) string    { return Colorize(Red, text) }
func ColorGreen(text string) string  { return Colorize(Green, text) }
func ColorYellow(text string) string { return Colorize(Yellow, text) }
func ColorBlue(text string) string   { return Colorize(Blue, text) }
func ColorCyan(text string) string   { return Colorize(Cyan, text) }
func ColorGray(text string) string   { return Colorize(Gray, text) }
func ColorBold(text string) string   { return Colorize(Bold, text) }

// ColorStatus colors event statuses
func ColorStatus(status string) string {
	switch status {
	case "confirmed":
		return ColorGreen(status)
	case "tentative":
		return ColorYellow(status)
	case "cancelled":
		return ColorRed(status)
	default:
		return status
	}
}
