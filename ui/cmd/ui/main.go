package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"hyperliquid-bot/ui/internal/app"
	uiconfig "hyperliquid-bot/ui/internal/config"
)

func main() {
	var (
		configPath     = flag.String("config", "", "TOML config path")
		listenOverride = flag.String("listen", "", "HTTP listen address override")
	)
	flag.Parse()

	cfg, err := uiconfig.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	if strings.TrimSpace(*listenOverride) != "" {
		cfg.Server.Listen = *listenOverride
	}

	ui := app.New(app.Config{
		Credentials:          cfg.ProviderConfig(),
		AddressOverride:      cfg.Overrides.Address,
		PrivateKeyOverride:   cfg.Overrides.PrivateKey,
		VaultAddressOverride: cfg.Overrides.VaultAddress,
		BaseURL:              cfg.Hyperliquid.BaseURL,
		Testnet:              cfg.Hyperliquid.Testnet,
		Timeout:              cfg.Hyperliquid.Timeout,
	})

	fmt.Printf("UI listening on %s\n", displayURL(cfg.Server.Listen))
	log.Fatal(http.ListenAndServe(cfg.Server.Listen, ui.Routes()))
}

func displayURL(listen string) string {
	if strings.HasPrefix(listen, ":") {
		return "http://localhost" + listen
	}
	return "http://" + listen
}
