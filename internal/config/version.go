package config

// Build info set at build time via ldflags
// Example: go build -ldflags "-X github.com/porteden/cli/internal/config.Version=1.2.0"
var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

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
