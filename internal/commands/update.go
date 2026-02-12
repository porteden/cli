package commands

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/porteden/cli/internal/config"
	"github.com/porteden/cli/internal/output"
	"github.com/porteden/cli/internal/system"
	"github.com/porteden/cli/internal/version"
	"github.com/spf13/cobra"
)

var selfUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update to the latest version",
	Long: `Update the PortEden CLI to the latest release.

The update method is automatically detected based on how you installed the CLI:
  - Homebrew:  runs 'brew upgrade porteden/tap/porteden'
  - Go:        runs 'go install github.com/porteden/cli/cmd/porteden@latest'
  - Script:    downloads the latest binary from GitHub releases`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate()
	},
}

func runUpdate() error {
	method := system.DetectInstallMethod()

	// Check latest version
	fmt.Println("Checking for updates...")
	latest, err := version.FetchLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if config.Version != "dev" && latest == config.Version {
		fmt.Printf("Already up to date (v%s).\n", config.Version)
		return nil
	}

	if config.Version != "dev" {
		fmt.Printf("Current version: v%s\n", config.Version)
	}
	fmt.Printf("Latest version:  v%s\n", latest)
	fmt.Println()

	switch method {
	case system.InstallHomebrew:
		return updateViaHomebrew()
	case system.InstallGo:
		return updateViaGo()
	default:
		return updateViaScript()
	}
}

func updateViaHomebrew() error {
	fmt.Println("Updating via Homebrew...")
	cmd := exec.Command("brew", "upgrade", "porteden/tap/porteden")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew upgrade failed: %w", err)
	}
	output.PrintSuccess("Updated successfully!")
	return nil
}

func updateViaGo() error {
	fmt.Println("Updating via go install...")
	cmd := exec.Command("go", "install", "github.com/porteden/cli/cmd/porteden@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go install failed: %w", err)
	}
	output.PrintSuccess("Updated successfully!")
	return nil
}

func updateViaScript() error {
	fmt.Println("Downloading latest release...")

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine binary path: %w", err)
	}

	// Fetch the latest release info from GitHub
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/porteden/cli/releases/latest")
	if err != nil {
		return fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read release info: %w", err)
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return fmt.Errorf("failed to parse release info: %w", err)
	}

	// Find the right asset for this OS/arch
	osName := runtime.GOOS
	archName := runtime.GOARCH
	// goreleaser uses Darwin/Linux and x86_64/arm64
	osMap := map[string]string{"darwin": "Darwin", "linux": "Linux", "windows": "Windows"}
	archMap := map[string]string{"amd64": "x86_64", "arm64": "arm64"}

	wantOS := osMap[osName]
	wantArch := archMap[archName]
	if wantOS == "" || wantArch == "" {
		return fmt.Errorf("unsupported platform: %s/%s", osName, archName)
	}

	var downloadURL string
	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, wantOS) && strings.Contains(asset.Name, wantArch) && strings.HasSuffix(asset.Name, ".tar.gz") {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no release found for %s/%s", wantOS, wantArch)
	}

	// Download the tarball
	dlResp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download release: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", dlResp.StatusCode)
	}

	// Extract the binary from the tarball
	gz, err := gzip.NewReader(dlResp.Body)
	if err != nil {
		return fmt.Errorf("failed to decompress: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var binaryData []byte
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tarball: %w", err)
		}
		// Look for the porteden binary
		name := header.Name
		if strings.HasSuffix(name, "/porteden") || name == "porteden" {
			binaryData, err = io.ReadAll(tr)
			if err != nil {
				return fmt.Errorf("failed to extract binary: %w", err)
			}
			break
		}
	}

	if binaryData == nil {
		return fmt.Errorf("binary not found in release tarball")
	}

	// Write the new binary, replacing the old one
	// Write to a temp file first, then rename for atomicity
	tmpFile := exePath + ".tmp"
	if err := os.WriteFile(tmpFile, binaryData, 0755); err != nil {
		return fmt.Errorf("failed to write new binary: %w", err)
	}

	if err := os.Rename(tmpFile, exePath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	output.PrintSuccess("Updated successfully!")
	return nil
}
