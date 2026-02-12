package system

import (
	"os"
	"path/filepath"
	"strings"
)

// InstallMethod represents how the CLI was installed.
type InstallMethod string

const (
	InstallHomebrew InstallMethod = "homebrew"
	InstallGo       InstallMethod = "go"
	InstallScript   InstallMethod = "script"
)

// DetectInstallMethod determines how the CLI was installed by examining the binary path.
func DetectInstallMethod() InstallMethod {
	exe, err := os.Executable()
	if err != nil {
		return InstallScript
	}

	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return InstallScript
	}

	lower := strings.ToLower(exe)

	// Homebrew installs land in Cellar or under the homebrew prefix
	if strings.Contains(lower, "cellar") || strings.Contains(lower, "homebrew") {
		return InstallHomebrew
	}

	// go install puts binaries in GOPATH/bin or ~/go/bin
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}
	if strings.HasPrefix(exe, filepath.Join(gopath, "bin")) {
		return InstallGo
	}

	return InstallScript
}
