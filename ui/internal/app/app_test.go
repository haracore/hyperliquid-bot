package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetPagesRender(t *testing.T) {
	ui := New(Config{DefaultAddress: "0xabc"})
	server := httptest.NewServer(ui.Routes())
	defer server.Close()

	for _, path := range []string{"/balances", "/positions", "/perp/orders", "/spot/orders"} {
		t.Run(path, func(t *testing.T) {
			resp, err := http.Get(server.URL + path)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200, got %d", resp.StatusCode)
			}
		})
	}
}

func TestWriteActionRequiresPrivateKey(t *testing.T) {
	ui := New(Config{})
	req := httptest.NewRequest(http.MethodPost, "/spot/orders", strings.NewReader("action=cancel-oid&coin=PURR%2FUSDC&oid=123&confirm=on"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	ui.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "private key is not configured") {
		t.Fatalf("expected private key error, got body: %s", rec.Body.String())
	}
}
