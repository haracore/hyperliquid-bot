package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	execution "hyperliquid-bot/execution/client"
	"hyperliquid-bot/execution/internal/clientutil"
)

func main() {
	var (
		privateKey  = flag.String("private-key", os.Getenv("HYPERLIQUID_PRIVATE_KEY"), "private key; can also be set with HYPERLIQUID_PRIVATE_KEY")
		baseURL     = flag.String("base-url", os.Getenv("HYPERLIQUID_BASE_URL"), "Hyperliquid API base URL")
		testnet     = flag.Bool("testnet", false, "use Hyperliquid testnet")
		coin        = flag.String("coin", "", "spot pair, for example PURR/USDC, or indexed spot asset like @8")
		oid         = flag.Int("oid", 0, "order id to modify")
		oidCloidRaw = flag.String("oid-cloid", "", "existing order cloid instead of oid")
		side        = flag.String("side", "buy", "buy or sell")
		size        = flag.Float64("size", 0, "new order size")
		price       = flag.Float64("price", 0, "new limit price")
		tif         = flag.String("tif", "Gtc", "time in force: Gtc, Ioc, or Alo")
		newCloidRaw = flag.String("new-cloid", "", "optional new client order id, 16-byte hex string")
		confirm     = flag.Bool("confirm", false, "actually modify the order")
		timeout     = flag.Duration("timeout", 20*time.Second, "HTTP timeout")
	)
	flag.Parse()

	clientutil.RequirePrivateKey(*privateKey)
	clientutil.RequireCoin(*coin)
	if *oid == 0 && *oidCloidRaw == "" {
		clientutil.ExitUsage("pass -oid or -oid-cloid")
	}
	if *size <= 0 {
		clientutil.ExitUsage("-size must be greater than 0")
	}
	if *price <= 0 {
		clientutil.ExitUsage("-price must be greater than 0")
	}
	isBuy, err := clientutil.ParseSide(*side)
	if err != nil {
		clientutil.ExitUsage(err.Error())
	}
	if !clientutil.ValidTIF(*tif) {
		clientutil.ExitUsage("-tif must be one of Gtc, Ioc, Alo")
	}
	if !*confirm {
		fmt.Fprintln(os.Stderr, "refusing to modify without -confirm")
		os.Exit(2)
	}
	orderID, err := execution.ParseOrderID(*oid, *oidCloidRaw)
	if err != nil {
		clientutil.ExitErr("order cloid", err)
	}
	var newCloid *execution.Cloid
	if *newCloidRaw != "" {
		parsed, err := execution.NewCloid(*newCloidRaw)
		if err != nil {
			clientutil.ExitErr("new cloid", err)
		}
		newCloid = &parsed
	}
	base := clientutil.ResolveBaseURL(*baseURL, *testnet)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	client := execution.New(execution.Config{BaseURL: base, Timeout: *timeout, PrivateKey: *privateKey})
	response, err := client.ModifySpotOrder(ctx, execution.ModifyOrderRequest{
		OrderID: orderID,
		Coin:    *coin,
		IsBuy:   isBuy,
		Size:    *size,
		Price:   *price,
		TIF:     *tif,
		Cloid:   newCloid,
	})
	if err != nil {
		clientutil.ExitErr("spot modify order", err)
	}
	clientutil.PrintJSON(response)
}
