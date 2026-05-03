package secrets

import (
	"context"
	"errors"
	"testing"
)

func TestEnvProvider(t *testing.T) {
	env := map[string]string{
		"HYPERLIQUID_ADDRESS":     "0xabc",
		"HYPERLIQUID_PRIVATE_KEY": "0xsecret",
	}
	provider := NewEnvProvider(EnvConfig{
		PathFields: map[string]EnvFields{
			"accounts/main": {
				"address":     "HYPERLIQUID_ADDRESS",
				"private_key": "HYPERLIQUID_PRIVATE_KEY",
			},
		},
		Lookup: func(name string) (string, bool) {
			value, ok := env[name]
			return value, ok
		},
	})

	bundle, err := provider.Get(context.Background(), Ref{Path: "/accounts/main"})
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Get("private_key") != "0xsecret" {
		t.Fatalf("expected private key, got %q", bundle.Get("private_key"))
	}
}

func TestEnvProviderNotFound(t *testing.T) {
	provider := NewEnvProvider(EnvConfig{})

	_, err := provider.Get(context.Background(), Ref{Path: "missing"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
