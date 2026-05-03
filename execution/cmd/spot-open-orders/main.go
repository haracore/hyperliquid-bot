package main

import (
	"context"
	"flag"
	"os"
	"time"

	"hyperliquid-bot/execution/internal/clientutil"
	hlinfo "hyperliquid-bot/sdk/info"
)

func main() {
	var (
		address  = flag.String("address", os.Getenv("HYPERLIQUID_ADDRESS"), "user address; can also be set with HYPERLIQUID_ADDRESS")
		baseURL  = flag.String("base-url", os.Getenv("HYPERLIQUID_BASE_URL"), "Hyperliquid API base URL")
		testnet  = flag.Bool("testnet", false, "use Hyperliquid testnet")
		frontend = flag.Bool("frontend", true, "use frontendOpenOrders response")
		timeout  = flag.Duration("timeout", 15*time.Second, "HTTP timeout")
	)
	flag.Parse()
	clientutil.RequireAddress(*address)
	base := clientutil.ResolveBaseURL(*baseURL, *testnet)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	info := hlinfo.New(base, *timeout)
	var response any
	var err error
	if *frontend {
		err = info.FrontendOpenOrders(ctx, *address, "", &response)
	} else {
		err = info.OpenOrders(ctx, *address, "", &response)
	}
	if err != nil {
		clientutil.ExitErr("spot open orders", err)
	}
	clientutil.PrintJSON(response)
}
