package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"hyperliquid-bot/ui/internal/app"
)

func main() {
	var (
		listen     = flag.String("listen", ":8080", "HTTP listen address")
		address    = flag.String("address", os.Getenv("HYPERLIQUID_ADDRESS"), "default Hyperliquid address")
		privateKey = flag.String("private-key", os.Getenv("HYPERLIQUID_PRIVATE_KEY"), "private key for state-changing actions")
		vault      = flag.String("vault-address", os.Getenv("HYPERLIQUID_VAULT_ADDRESS"), "optional vault/subaccount address")
		baseURL    = flag.String("base-url", os.Getenv("HYPERLIQUID_BASE_URL"), "Hyperliquid API base URL")
		testnet    = flag.Bool("testnet", false, "use Hyperliquid testnet")
		timeout    = flag.Duration("timeout", 20*time.Second, "Hyperliquid request timeout")
	)
	flag.Parse()

	ui := app.New(app.Config{
		DefaultAddress: *address,
		PrivateKey:     *privateKey,
		VaultAddress:   *vault,
		BaseURL:        *baseURL,
		Testnet:        *testnet,
		Timeout:        *timeout,
	})

	fmt.Printf("UI listening on %s\n", displayURL(*listen))
	log.Fatal(http.ListenAndServe(*listen, ui.Routes()))
}

func displayURL(listen string) string {
	if strings.HasPrefix(listen, ":") {
		return "http://localhost" + listen
	}
	return "http://" + listen
}
