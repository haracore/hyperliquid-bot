package credentials

import (
	"context"
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
