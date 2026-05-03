package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"hyperliquid-bot/execution/credentials"
)

func TestLoadVaultConfig(t *testing.T) {
	t.Setenv("VAULT_TOKEN", "token-from-env")
	path := writeConfig(t, `
[server]
listen = ":9090"

[hyperliquid]
base_url = "https://example.test"
testnet = true
timeout = "7s"

[secrets]
provider = "vault"
account = "trader"
prefix = "accounts"

[secrets.vault]
addr = "http://vault.test:8200"
token_env = "VAULT_TOKEN"
namespace = "team"
mount = "secret"
prefix = "hyperliquid"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	provider := cfg.ProviderConfig()
	if cfg.Server.Listen != ":9090" {
		t.Fatalf("expected listen :9090, got %q", cfg.Server.Listen)
	}
	if !cfg.Hyperliquid.Testnet {
		t.Fatal("expected testnet true")
	}
	if cfg.Hyperliquid.Timeout != 7*time.Second {
		t.Fatalf("expected 7s timeout, got %s", cfg.Hyperliquid.Timeout)
	}
	if provider.Name != credentials.ProviderVault {
		t.Fatalf("expected vault provider, got %q", provider.Name)
	}
	if provider.VaultToken != "token-from-env" {
		t.Fatalf("expected token from env, got %q", provider.VaultToken)
	}
	if provider.VaultPrefix != "hyperliquid" {
		t.Fatalf("expected vault prefix, got %q", provider.VaultPrefix)
	}
}

func TestLoadVaultUserpassConfig(t *testing.T) {
	t.Setenv("VAULT_USERNAME", "alice")
	t.Setenv("VAULT_PASSWORD", "secret")
	t.Setenv("VAULT_OTP", "123456")
	path := writeConfig(t, `
[secrets]
provider = "vault-userpass"
account = "main"

[secrets.vault_userpass]
addr = "http://vault.test:8200"
username_env = "VAULT_USERNAME"
password_env = "VAULT_PASSWORD"
otp_env = "VAULT_OTP"
auth_mount = "userpass"
mount = "secret"
prefix = "hyperliquid"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	provider := cfg.ProviderConfig()
	if provider.Name != credentials.ProviderVaultUserpass {
		t.Fatalf("expected vault-userpass provider, got %q", provider.Name)
	}
	if provider.VaultUsername != "alice" {
		t.Fatalf("expected username from env, got %q", provider.VaultUsername)
	}
	if provider.VaultPassword != "secret" {
		t.Fatalf("expected password from env, got %q", provider.VaultPassword)
	}
	if provider.VaultOTP != "123456" {
		t.Fatalf("expected otp from env, got %q", provider.VaultOTP)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
