package secrets

import (
	"context"
	"errors"
	"testing"
)

type staticProvider map[string]Bundle

func (p staticProvider) Get(_ context.Context, ref Ref) (Bundle, error) {
	bundle, ok := p[cleanPath(ref.Path)]
	if !ok {
		return Bundle{}, ErrNotFound
	}
	return bundle, nil
}

func TestAccountResolver(t *testing.T) {
	provider := staticProvider{
		"accounts/main": {
			Path: "accounts/main",
			Fields: map[string]string{
				"address":       "0xabc",
				"private_key":   "0xsecret",
				"vault_address": "0xvault",
			},
		},
	}

	account, err := NewAccountResolver(provider, "accounts").Account(context.Background(), "main")
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

func TestAccountResolverRequiresPrivateKey(t *testing.T) {
	provider := staticProvider{
		"accounts/main": {
			Path:   "accounts/main",
			Fields: map[string]string{"address": "0xabc"},
		},
	}

	_, err := NewAccountResolver(provider, "accounts").Account(context.Background(), "main")
	if !errors.Is(err, ErrMissingField) {
		t.Fatalf("expected ErrMissingField, got %v", err)
	}
}
