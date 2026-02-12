package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const credentialsFile = "credentials.json"

// credentialStore is the on-disk JSON format.
type credentialStore struct {
	ActiveProfile string            `json:"active_profile"`
	Profiles      map[string]string `json:"profiles"`
}

var store *credentialStore

// InitStore initializes the file-based credential store.
func InitStore() error {
	if store != nil {
		return nil
	}
	return loadStore()
}

func loadStore() error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, credentialsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			store = &credentialStore{
				ActiveProfile: "default",
				Profiles:      make(map[string]string),
			}
			return nil
		}
		return fmt.Errorf("failed to read credentials file: %w", err)
	}

	var s credentialStore
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("failed to parse credentials file %s: %w", path, err)
	}
	if s.Profiles == nil {
		s.Profiles = make(map[string]string)
	}
	if s.ActiveProfile == "" {
		s.ActiveProfile = "default"
	}
	store = &s
	return nil
}

func saveStore() error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode credentials: %w", err)
	}

	path := filepath.Join(dir, credentialsFile)
	if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}
	return nil
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "porteden"), nil
}

func ensureStore() error {
	if store == nil {
		return fmt.Errorf("credential store not initialized - run 'porteden auth login' first")
	}
	return nil
}

// StoreAPIKey stores an API key for a profile.
func StoreAPIKey(apiKey, profile string) error {
	if err := ensureStore(); err != nil {
		return err
	}
	if profile == "" {
		profile = "default"
	}
	store.Profiles[profile] = apiKey
	return saveStore()
}

// GetAPIKey retrieves the API key for a profile, checking PE_API_KEY first.
func GetAPIKey(profile string) (string, error) {
	if envKey := os.Getenv("PE_API_KEY"); envKey != "" {
		return envKey, nil
	}
	return GetStoredAPIKey(profile)
}

// GetStoredAPIKey retrieves the API key from the credential store only, ignoring PE_API_KEY.
func GetStoredAPIKey(profile string) (string, error) {
	if err := ensureStore(); err != nil {
		return "", err
	}
	if profile == "" {
		profile = GetActiveProfile()
	}
	key, ok := store.Profiles[profile]
	if !ok || key == "" {
		return "", fmt.Errorf("no API key found for profile %q", profile)
	}
	return key, nil
}

// DeleteAPIKey removes the API key for a profile.
func DeleteAPIKey(profile string) error {
	if err := ensureStore(); err != nil {
		return err
	}
	if profile == "" {
		profile = "default"
	}
	delete(store.Profiles, profile)
	return saveStore()
}

// GetActiveProfile returns the currently active profile name.
func GetActiveProfile() string {
	if store == nil {
		return "default"
	}
	return store.ActiveProfile
}

// SetActiveProfile sets the active profile.
func SetActiveProfile(profile string) error {
	if err := ensureStore(); err != nil {
		return err
	}
	store.ActiveProfile = profile
	return saveStore()
}

// ListProfiles returns all stored profile names and the active profile.
func ListProfiles() (profiles []string, activeProfile string, err error) {
	if err := ensureStore(); err != nil {
		return nil, "", err
	}
	for name := range store.Profiles {
		profiles = append(profiles, name)
	}
	sort.Strings(profiles)
	return profiles, store.ActiveProfile, nil
}
