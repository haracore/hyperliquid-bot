package main

import (
	"context"
	"flag"
	"os"
	"time"

	execution "hyperliquid-bot/execution/client"
	"hyperliquid-bot/execution/internal/clientutil"
)

func main() {
	var (
		address  = flag.String("address", "", "user address; overrides execution secrets")
		baseURL  = flag.String("base-url", os.Getenv("HYPERLIQUID_BASE_URL"), "Hyperliquid API base URL")
		testnet  = flag.Bool("testnet", false, "use Hyperliquid testnet")
		frontend = flag.Bool("frontend", true, "use frontendOpenOrders response")
		timeout  = flag.Duration("timeout", 15*time.Second, "HTTP timeout")
	)
	secretFlags := clientutil.AddSecretFlags()
	flag.Parse()
	account := clientutil.ResolveAccountFields(context.Background(), secretFlags, "", *address, "", *timeout)
	clientutil.RequireAddress(account.Address)
	base := clientutil.ResolveBaseURL(*baseURL, *testnet)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	client := execution.New(execution.Config{BaseURL: base, Timeout: *timeout})
	response, err := client.SpotOpenOrders(ctx, account.Address, *frontend)
	if err != nil {
		clientutil.ExitErr("spot open orders", err)
	}
	clientutil.PrintJSON(response)
}
