package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/porteden/cli/internal/config"
	"github.com/porteden/cli/internal/output"
	"github.com/porteden/cli/internal/system"
)

const (
	versionCheckURL    = "https://api.github.com/repos/porteden/cli/releases/latest"
	checkCacheFile     = "version-check"
	checkIntervalHours = 24
)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// CheckForUpdate checks if a newer version is available (once per day).
// Prints a subtle notification to stderr if an update is available.
// This function is non-blocking and fails silently on errors.
func CheckForUpdate() {
	if config.Version == "dev" {
		return // Don't check for updates on dev builds
	}

	cacheFile := filepath.Join(configDir(), checkCacheFile)

	// Check if we've checked recently
	if stat, err := os.Stat(cacheFile); err == nil {
		if time.Since(stat.ModTime()) < checkIntervalHours*time.Hour {
			return // Checked recently, skip
		}
	}

	// Perform check in background
	go func() {
		latestVersion, err := FetchLatestVersion()
		if err != nil {
			return // Fail silently
		}

		// Update cache file timestamp
		_ = os.MkdirAll(configDir(), 0700)
		_ = os.WriteFile(cacheFile, []byte(latestVersion), 0600)

		// Compare versions (simple string comparison - assumes semver)
		if latestVersion != config.Version && latestVersion > config.Version {
			fmt.Fprintf(os.Stderr, "\n%s\n",
				output.ColorYellow(fmt.Sprintf(
					"A new version of porteden is available (%s). %s",
					latestVersion, updateHint())))
		}
	}()
}

func updateHint() string {
	switch system.DetectInstallMethod() {
	case system.InstallHomebrew:
		return "Run 'brew upgrade porteden' to update."
	case system.InstallGo:
		return "Run 'porteden update' or 'go install github.com/porteden/cli/cmd/porteden@latest' to update."
	default:
		return "Run 'porteden update' to update."
	}
}

// FetchLatestVersion fetches the latest release version from GitHub.
func FetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(versionCheckURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return "", err
	}

	// Strip 'v' prefix if present
	version := release.TagName
	if len(version) > 0 && version[0] == 'v' {
		version = version[1:]
	}

	return version, nil
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "porteden")
}
