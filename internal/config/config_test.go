package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestKeysAndProfilesRoundTrip(t *testing.T) {
	keyring.MockInit()
	t.Setenv("MERCYPAY_CONFIG_DIR", t.TempDir())

	config, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	config.Profile("default").MerchantName = "Boutique Ada"
	if inKeychain, err := config.StoreKey("default", "mp_sandbox_abc"); err != nil || !inKeychain {
		t.Fatalf("%v %v", inKeychain, err)
	}

	reloaded, _ := Load()
	if reloaded.Key("default") != "mp_sandbox_abc" || reloaded.Profiles["default"].MerchantName != "Boutique Ada" {
		t.Fatalf("not reloaded: %+v", reloaded.Profiles["default"])
	}
	if reloaded.Profiles["default"].APIKey != "" {
		t.Fatal("the key must not be written to the file when the keychain works")
	}

	info, _ := os.Stat(filepath.Join(os.Getenv("MERCYPAY_CONFIG_DIR"), "config.json"))
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode %v", info.Mode().Perm())
	}

	if err := reloaded.Forget("default"); err != nil || reloaded.Key("default") != "" {
		t.Fatal("forget must remove the key")
	}
}
