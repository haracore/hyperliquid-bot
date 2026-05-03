package secrets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVaultProviderKV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/secret/data/hyperliquid/accounts/main" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("version") != "3" {
			t.Fatalf("unexpected version %q", r.URL.Query().Get("version"))
		}
		if r.Header.Get("X-Vault-Token") != "token" {
			t.Fatalf("missing vault token")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"data": {
					"address": "0xabc",
					"private_key": "0xsecret",
					"vault_address": "0xvault"
				},
				"metadata": {"version": 3}
			}
		}`))
	}))
	defer server.Close()

	provider := NewVaultProvider(VaultConfig{
		Address: server.URL,
		Token:   "token",
		Mount:   "secret",
		Prefix:  "hyperliquid",
	})

	bundle, err := provider.Get(context.Background(), Ref{Path: "accounts/main", Version: 3})
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Version != 3 {
		t.Fatalf("expected version 3, got %d", bundle.Version)
	}
	if bundle.Get("private_key") != "0xsecret" {
		t.Fatalf("expected private key, got %q", bundle.Get("private_key"))
	}
}

func TestVaultProviderNotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	provider := NewVaultProvider(VaultConfig{Address: server.URL, Token: "token"})
	_, err := provider.Get(context.Background(), Ref{Path: "missing"})
	if err == nil {
		t.Fatal("expected error")
	}
}
