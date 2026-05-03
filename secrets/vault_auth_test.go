package secrets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVaultUserpassAuthenticatorLoginWithMFA(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/userpass/login/trader" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("X-Vault-MFA") != "totp-main:123456" {
			t.Fatalf("unexpected mfa header %q", r.Header.Get("X-Vault-MFA"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"auth": {
				"client_token": "hvs.token",
				"accessor": "accessor",
				"policies": ["default", "hyperliquid-read"],
				"lease_duration": 3600,
				"renewable": true
			}
		}`))
	}))
	defer server.Close()

	auth := NewVaultUserpassAuthenticator(VaultUserpassConfig{
		Address:   server.URL,
		Username:  "trader",
		Password:  "pass",
		MFAMethod: "totp-main",
		OTP:       "123456",
	})

	token, err := auth.Login(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if token.ClientToken != "hvs.token" {
		t.Fatalf("expected token, got %q", token.ClientToken)
	}
	if token.LeaseDuration != 3600 {
		t.Fatalf("expected lease duration 3600, got %d", token.LeaseDuration)
	}
}

func TestVaultUserpassProviderLogsInThenReadsSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/auth/userpass/login/trader":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"auth":{"client_token":"hvs.token"}}`))
		case "/v1/secret/data/hyperliquid/accounts/main":
			if r.Header.Get("X-Vault-Token") != "hvs.token" {
				t.Fatalf("unexpected vault token %q", r.Header.Get("X-Vault-Token"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": {
					"data": {"private_key": "0xsecret"},
					"metadata": {"version": 1}
				}
			}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	provider := NewVaultUserpassProvider(
		VaultUserpassConfig{
			Address:  server.URL,
			Username: "trader",
			Password: "pass",
		},
		VaultConfig{
			Mount:  "secret",
			Prefix: "hyperliquid",
		},
	)

	bundle, err := provider.Get(context.Background(), Ref{Path: "accounts/main"})
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Get("private_key") != "0xsecret" {
		t.Fatalf("expected private key, got %q", bundle.Get("private_key"))
	}
}
