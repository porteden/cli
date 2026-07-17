package config

import "strings"

// Build info set at build time via ldflags
// Example: go build -ldflags "-X github.com/porteden/cli/internal/config.Version=1.2.0"
var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

// SemverVersion returns Version without a leading "v". goreleaser injects the
// v-prefixed git tag (e.g. "v0.2.0") while the GitHub release lookup reports
// the stripped form ("0.2.0"); comparing the two raw strings would never match,
// so the update banner and `porteden update`'s up-to-date check both use this.
func SemverVersion() string {
	return strings.TrimPrefix(Version, "v")
}

// FullVersion returns version string with commit and date if available
func FullVersion() string {
	v := Version
	if Commit != "" {
		v += " (" + Commit + ")"
	}
	if Date != "" {
		v += " built " + Date
	}
	return v
}
