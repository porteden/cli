package commands

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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

	current := config.SemverVersion()
	if config.Version != "dev" && latest == current {
		fmt.Printf("Already up to date (v%s).\n", current)
		return nil
	}

	if config.Version != "dev" {
		fmt.Printf("Current version: v%s\n", current)
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
	// Resolve symlinks so we replace the real file, not a symlink into it.
	if resolved, rerr := filepath.EvalSymlinks(exePath); rerr == nil {
		exePath = resolved
	}

	// Small client for the release-info JSON call (bounded, fast).
	infoClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := infoClient.Get("https://api.github.com/repos/porteden/cli/releases/latest")
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

	// goreleaser's name_template is "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
	// with raw lowercase GOOS/GOARCH, and windows ships .zip (everything else .tar.gz).
	// Match on the "_<goos>_<goarch>" infix + the right extension so we never
	// re-introduce the Title-case / x86_64 mismatch that broke this on every OS.
	wantExt := ".tar.gz"
	if runtime.GOOS == "windows" {
		wantExt = ".zip"
	}
	wantInfix := fmt.Sprintf("_%s_%s%s", runtime.GOOS, runtime.GOARCH, wantExt)

	var downloadURL, assetName string
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, wantInfix) {
			downloadURL = asset.BrowserDownloadURL
			assetName = asset.Name
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no release asset found for %s/%s (looked for *%s)", runtime.GOOS, runtime.GOARCH, wantInfix)
	}

	// Download the archive with a generous context deadline instead of a
	// client-wide Timeout, which would otherwise abort a multi-MB download on
	// a slow link mid-stream.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to build download request: %w", err)
	}
	dlResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download release: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", dlResp.StatusCode)
	}

	binaryName := "porteden"
	if runtime.GOOS == "windows" {
		binaryName = "porteden.exe"
	}

	var binaryData []byte
	if strings.HasSuffix(assetName, ".zip") {
		binaryData, err = extractFromZip(dlResp.Body, binaryName)
	} else {
		binaryData, err = extractFromTarGz(dlResp.Body, binaryName)
	}
	if err != nil {
		return err
	}
	if binaryData == nil {
		return fmt.Errorf("binary %q not found in release archive", binaryName)
	}

	return replaceBinary(exePath, binaryData)
}

// extractFromTarGz pulls the named binary out of a gzipped tarball stream.
func extractFromTarGz(r io.Reader, binaryName string) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tarball: %w", err)
		}
		if path.Base(header.Name) == binaryName {
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("failed to extract binary: %w", err)
			}
			return data, nil
		}
	}
	return nil, nil
}

// extractFromZip pulls the named binary out of a zip archive. zip.NewReader
// needs a ReaderAt + size, so the (bounded) archive is buffered first.
func extractFromZip(r io.Reader, binaryName string) ([]byte, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read zip archive: %w", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip archive: %w", err)
	}
	for _, f := range zr.File {
		if path.Base(f.Name) == binaryName {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open binary in zip: %w", err)
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to extract binary: %w", err)
			}
			return data, nil
		}
	}
	return nil, nil
}

// replaceBinary swaps in the new binary. On Windows the running image is
// locked and cannot be overwritten or deleted, but it CAN be renamed aside —
// so move the current binary out of the way, then move the new one into place
// (best-effort cleanup of the old file happens on the next run's temp write).
func replaceBinary(exePath string, data []byte) error {
	tmpFile := exePath + ".new"
	if err := os.WriteFile(tmpFile, data, 0755); err != nil {
		return fmt.Errorf("failed to write new binary: %w", err)
	}

	if runtime.GOOS == "windows" {
		oldFile := exePath + ".old"
		_ = os.Remove(oldFile) // clean up a previous update's leftover
		if err := os.Rename(exePath, oldFile); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("failed to move current binary aside: %w", err)
		}
		if err := os.Rename(tmpFile, exePath); err != nil {
			// Roll back so the user isn't left with no binary.
			_ = os.Rename(oldFile, exePath)
			os.Remove(tmpFile)
			return fmt.Errorf("failed to install new binary: %w", err)
		}
		output.PrintSuccess("Updated successfully!")
		output.PrintInfo(fmt.Sprintf("A leftover %s.old can be deleted once this process exits.", exePath))
		return nil
	}

	if err := os.Rename(tmpFile, exePath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to replace binary: %w", err)
	}
	output.PrintSuccess("Updated successfully!")
	return nil
}
