package output

import (
	"os"
	"time"

	"github.com/porteden/cli/internal/debug"
)

// GetOutputLocation returns the timezone location for output formatting.
// It checks PE_TIMEZONE environment variable first, falling back to time.Local.
func GetOutputLocation() *time.Location {
	tzName := os.Getenv("PE_TIMEZONE")
	if tzName == "" {
		return time.Local
	}

	loc, err := time.LoadLocation(tzName)
	if err != nil {
		if debug.Verbose {
			debug.Log("Invalid PE_TIMEZONE %q: %v, using local timezone", tzName, err)
		}
		return time.Local
	}
	return loc
}

// FormatLocalTime converts a UTC time to the output timezone and formats it as RFC3339.
func FormatLocalTime(utc time.Time) string {
	if utc.IsZero() {
		return ""
	}
	return utc.In(GetOutputLocation()).Format(time.RFC3339)
}

// GetLocalStart returns the local start time string.
// If startLocal is provided and non-empty, it's returned as-is.
// Otherwise, startUtc is converted to local time.
func GetLocalStart(startLocal string, startUtc time.Time) string {
	if startLocal != "" {
		return startLocal
	}
	return FormatLocalTime(startUtc)
}

// GetLocalEnd returns the local end time string.
// If endLocal is provided and non-empty, it's returned as-is.
// Otherwise, endUtc is converted to local time.
func GetLocalEnd(endLocal string, endUtc time.Time) string {
	if endLocal != "" {
		return endLocal
	}
	return FormatLocalTime(endUtc)
}

// safeDate extracts the date portion (YYYY-MM-DD) from a datetime string.
// Returns empty string if input is too short.
func safeDate(s string) string {
	if len(s) < 10 {
		return ""
	}
	return s[:10]
}

// safeTime extracts the time portion (HH:MM) from a datetime string.
// Expects format like "2024-01-15T09:30:00" where time starts at index 11.
// Returns empty string if input is too short.
func safeTime(s string) string {
	if len(s) < 16 {
		return ""
	}
	return s[11:16]
}
