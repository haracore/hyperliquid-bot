package credentials

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveAccountFromEnvProvider(t *testing.T) {
	t.Setenv("HYPERLIQUID_ADDRESS", "0xabc")
	t.Setenv("HYPERLIQUID_PRIVATE_KEY", "0xsecret")
	t.Setenv("HYPERLIQUID_VAULT_ADDRESS", "0xvault")

	account, err := ResolveAccount(context.Background(), ProviderConfig{
		Name:    ProviderEnv,
		Account: "main",
		Prefix:  "accounts",
	})
	if err != nil {
		t.Fatal(err)
	}
	if account.Address != "0xabc" {
		t.Fatalf("expected address, got %q", account.Address)
	}
	if account.PrivateKey != "0xsecret" {
		t.Fatalf("expected private key, got %q", account.PrivateKey)
	}
	if account.VaultAddress != "0xvault" {
		t.Fatalf("expected vault address, got %q", account.VaultAddress)
	}
}

func TestResolveAccountFieldsAllowsMissingPrivateKey(t *testing.T) {
	t.Setenv("HYPERLIQUID_ADDRESS", "0xabc")

	account, err := ResolveAccountFields(context.Background(), ProviderConfig{
		Name:    ProviderEnv,
		Account: "main",
		Prefix:  "accounts",
	})
	if err != nil {
		t.Fatal(err)
	}
	if account.Address != "0xabc" {
		t.Fatalf("expected address, got %q", account.Address)
	}
	if account.PrivateKey != "" {
		t.Fatalf("expected empty private key, got %q", account.PrivateKey)
	}
}

func TestResolveAccountFromVaultUserpassProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/auth/userpass/login/trader":
			if r.Header.Get("X-Vault-MFA") != "totp-main:123456" {
				t.Fatalf("expected MFA header, got %q", r.Header.Get("X-Vault-MFA"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"auth":{"client_token":"hvs.token"}}`))
		case "/v1/secret/data/hyperliquid/accounts/main":
			if r.Header.Get("X-Vault-Token") != "hvs.token" {
				t.Fatalf("expected login token, got %q", r.Header.Get("X-Vault-Token"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": {
					"data": {
						"address": "0xabc",
						"private_key": "0xsecret",
						"vault_address": "0xvault"
					},
					"metadata": {"version": 1}
				}
			}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	account, err := ResolveAccount(context.Background(), ProviderConfig{
		Name:           ProviderVaultUserpass,
		Account:        "main",
		Prefix:         "accounts",
		VaultAddress:   server.URL,
		VaultMount:     "secret",
		VaultPrefix:    "hyperliquid",
		VaultUsername:  "trader",
		VaultPassword:  "pass",
		VaultMFAMethod: "totp-main",
		VaultOTP:       "123456",
	})
	if err != nil {
		t.Fatal(err)
	}
	if account.Address != "0xabc" {
		t.Fatalf("expected address, got %q", account.Address)
	}
	if account.PrivateKey != "0xsecret" {
		t.Fatalf("expected private key, got %q", account.PrivateKey)
	}
	if account.VaultAddress != "0xvault" {
		t.Fatalf("expected vault address, got %q", account.VaultAddress)
	}
}

func TestApplyOverrides(t *testing.T) {
	account := ApplyOverrides(Account{
		Address:    "0xenv",
		PrivateKey: "0xsecret",
	}, "0xoverride", "0xaddr", "0xvault")

	if account.PrivateKey != "0xoverride" {
		t.Fatalf("expected private key override, got %q", account.PrivateKey)
	}
	if account.Address != "0xaddr" {
		t.Fatalf("expected address override, got %q", account.Address)
	}
	if account.VaultAddress != "0xvault" {
		t.Fatalf("expected vault override, got %q", account.VaultAddress)
	}
}
