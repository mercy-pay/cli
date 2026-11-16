// Package config keeps the CLI's profiles in ~/.config/mercypay/config.json and their keys in
// the system keychain, or in that file (readable by its owner only) when there is none.
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/zalando/go-keyring"
)

const keyringService = "mercypay-cli"

// Profile is one signed-in shop.
type Profile struct {
	MerchantID   string `json:"merchant_id,omitempty"`
	MerchantName string `json:"merchant_name,omitempty"`
	BaseURL      string `json:"base_url,omitempty"`
	// ListenSecret signs forwarded events; kept so the local app's secret stays the same.
	ListenSecret string `json:"listen_secret,omitempty"`
	// APIKey is only set when the keychain was not available.
	APIKey string `json:"api_key,omitempty"`
}

// Config is the whole file.
type Config struct {
	Profiles map[string]*Profile `json:"profiles"`
	path     string
}

// Dir is where the file lives: $XDG_CONFIG_HOME/mercypay, ~/.config/mercypay, or %AppData%\mercypay.
func Dir() (string, error) {
	if dir := os.Getenv("MERCYPAY_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	if runtime.GOOS == "windows" {
		base, err := os.UserConfigDir()
		return filepath.Join(base, "mercypay"), err
	}
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "mercypay"), nil
	}
	home, err := os.UserHomeDir()
	return filepath.Join(home, ".config", "mercypay"), err
}

// Load reads the file, or returns an empty config when there is none yet.
func Load() (*Config, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	config := &Config{Profiles: map[string]*Profile{}, path: filepath.Join(dir, "config.json")}

	content, err := os.ReadFile(config.path)
	if errors.Is(err, fs.ErrNotExist) {
		return config, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(content, config); err != nil {
		return nil, err
	}
	if config.Profiles == nil {
		config.Profiles = map[string]*Profile{}
	}
	return config, nil
}

// Save writes the file with owner-only permissions.
func (c *Config) Save() error {
	if err := os.MkdirAll(filepath.Dir(c.path), 0o700); err != nil {
		return err
	}
	content, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, append(content, '\n'), 0o600)
}

// Profile returns the named profile, created empty when missing.
func (c *Config) Profile(name string) *Profile {
	profile, ok := c.Profiles[name]
	if !ok {
		profile = &Profile{}
		c.Profiles[name] = profile
	}
	return profile
}

// StoreKey keeps the key in the keychain, or in the file when that fails. It reports which.
func (c *Config) StoreKey(name, key string) (inKeychain bool, err error) {
	profile := c.Profile(name)
	if keyring.Set(keyringService, name, key) == nil {
		profile.APIKey = ""
		return true, c.Save()
	}
	profile.APIKey = key
	return false, c.Save()
}

// Key returns the stored key of a profile, or "" when signed out.
func (c *Config) Key(name string) string {
	if profile, ok := c.Profiles[name]; ok && profile.APIKey != "" {
		return profile.APIKey
	}
	key, err := keyring.Get(keyringService, name)
	if err != nil {
		return ""
	}
	return key
}

// Forget removes a profile and its key.
func (c *Config) Forget(name string) error {
	_ = keyring.Delete(keyringService, name)
	delete(c.Profiles, name)
	return c.Save()
}
