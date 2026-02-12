package auth

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/porteden/cli/internal/output"
	"golang.org/x/term"
)

// ExportDestination represents where to export the API key beyond the keyring.
type ExportDestination string

const (
	ExportOpenClaw ExportDestination = "openclaw"
	ExportShell    ExportDestination = "shell"
	ExportNone     ExportDestination = "none"
)

// IsInteractiveTerminal returns true if stdin is a terminal.
func IsInteractiveTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// PromptExportDestination shows an interactive menu and returns the user's choice.
func PromptExportDestination(in io.Reader, out io.Writer) ExportDestination {
	shellProfile, _ := detectShellProfile()

	fmt.Fprintln(out)
	fmt.Fprintln(out, output.ColorBold("  Where would you also like to save your API key?"))
	fmt.Fprintln(out)

	home, _ := os.UserHomeDir()
	openclawPath := filepath.Join("~", ".openclaw", "openclaw.json")
	if home != "" {
		openclawPath = filepath.Join(home, ".openclaw", "openclaw.json")
	}

	fmt.Fprintf(out, "        %s OpenClaw gateway  %s\n", output.ColorCyan("[1]"), output.ColorGray("("+openclawPath+")"))
	fmt.Fprintf(out, "        %s Shell profile     %s\n", output.ColorCyan("[2]"), output.ColorGray("("+shellProfile+")"))
	fmt.Fprintf(out, "        %s Skip\n", output.ColorCyan("[3]"))
	fmt.Fprintln(out)

	reader := bufio.NewReader(in)
	for attempts := 0; attempts < 3; attempts++ {
		fmt.Fprint(out, "        Choice [3]: ")
		line, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(line)

		switch choice {
		case "", "3":
			return ExportNone
		case "1":
			return ExportOpenClaw
		case "2":
			return ExportShell
		}

		fmt.Fprintln(out, "        Invalid choice. Please enter 1-3.")
	}

	return ExportNone
}

// ExportAPIKey exports the API key to the specified destination.
func ExportAPIKey(apiKey string, dest ExportDestination) error {
	switch dest {
	case ExportOpenClaw:
		return exportToOpenClaw(apiKey)
	case ExportShell:
		return exportToShellProfile(apiKey)
	case ExportNone:
		return nil
	default:
		return fmt.Errorf("invalid export destination %q: must be openclaw, shell, or none", dest)
	}
}

func exportToOpenClaw(apiKey string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	dir := filepath.Join(home, ".openclaw")
	filePath := filepath.Join(dir, "openclaw.json")

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	data := make(map[string]interface{})
	existing, err := os.ReadFile(filePath)
	if err == nil && len(existing) > 0 {
		if err := json.Unmarshal(existing, &data); err != nil {
			return fmt.Errorf("failed to parse %s: %w", filePath, err)
		}
	}

	setNestedKey(data, []string{"skills", "entries", "porteden", "env", "PE_API_KEY"}, apiKey)

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	if err := os.WriteFile(filePath, append(out, '\n'), 0600); err != nil {
		return fmt.Errorf("failed to write %s: %w", filePath, err)
	}

	output.PrintSuccess(fmt.Sprintf("API key saved to %s", filePath))
	return nil
}

func setNestedKey(m map[string]interface{}, keys []string, value interface{}) {
	for i, key := range keys {
		if i == len(keys)-1 {
			m[key] = value
			return
		}
		next, ok := m[key]
		if !ok {
			next = map[string]interface{}{}
			m[key] = next
		}
		nextMap, ok := next.(map[string]interface{})
		if !ok {
			nextMap = map[string]interface{}{}
			m[key] = nextMap
		}
		m = nextMap
	}
}

var (
	peAPIKeyExportRe = regexp.MustCompile(`(?m)^export PE_API_KEY=.*$`)
	peAPIKeyPSRe     = regexp.MustCompile(`(?m)^\$env:PE_API_KEY\s*=.*$`)
)

func exportToShellProfile(apiKey string) error {
	profilePath, err := detectShellProfile()
	if err != nil {
		return err
	}
	if profilePath == "" {
		return fmt.Errorf("could not detect shell profile")
	}

	// Use PowerShell syntax on Windows (unless Git Bash)
	isPowerShell := runtime.GOOS == "windows" && !isGitBash()
	var exportLine string
	var matchRe *regexp.Regexp
	if isPowerShell {
		exportLine = fmt.Sprintf(`$env:PE_API_KEY = "%s"`, apiKey)
		matchRe = peAPIKeyPSRe
	} else {
		exportLine = fmt.Sprintf("export PE_API_KEY=%s", apiKey)
		matchRe = peAPIKeyExportRe
	}

	// Read existing content
	var content string
	var perm os.FileMode = 0644
	existing, err := os.ReadFile(profilePath)
	if err == nil {
		content = string(existing)
		if info, statErr := os.Stat(profilePath); statErr == nil {
			perm = info.Mode().Perm()
		}
	}

	if matchRe.MatchString(content) {
		// Replace existing line
		content = matchRe.ReplaceAllString(content, exportLine)
	} else {
		// Append
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += exportLine + "\n"
	}

	if err := os.WriteFile(profilePath, []byte(content), perm); err != nil {
		return fmt.Errorf("failed to write %s: %w", profilePath, err)
	}

	output.PrintSuccess(fmt.Sprintf("API key saved to %s", profilePath))
	if isPowerShell {
		output.PrintInfo("Restart PowerShell or run '. $PROFILE' to apply.")
	} else {
		output.PrintInfo("Run 'source " + profilePath + "' or open a new terminal to apply.")
	}
	return nil
}

func isGitBash() bool {
	return os.Getenv("MSYSTEM") != "" || strings.Contains(os.Getenv("SHELL"), "bash")
}

func detectShellProfile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	shell := os.Getenv("SHELL")

	switch runtime.GOOS {
	case "darwin":
		if strings.Contains(shell, "bash") {
			return filepath.Join(home, ".bash_profile"), nil
		}
		return filepath.Join(home, ".zshrc"), nil

	case "linux":
		if strings.Contains(shell, "zsh") {
			return filepath.Join(home, ".zshrc"), nil
		}
		return filepath.Join(home, ".bashrc"), nil

	case "windows":
		// Git Bash
		if os.Getenv("MSYSTEM") != "" || strings.Contains(shell, "bash") {
			return filepath.Join(home, ".bashrc"), nil
		}
		// PowerShell profile
		return filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"), nil

	default:
		if strings.Contains(shell, "zsh") {
			return filepath.Join(home, ".zshrc"), nil
		}
		return filepath.Join(home, ".bashrc"), nil
	}
}
