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
		privateKey   = flag.String("private-key", os.Getenv("HYPERLIQUID_PRIVATE_KEY"), "private key; can also be set with HYPERLIQUID_PRIVATE_KEY")
		baseURL      = flag.String("base-url", os.Getenv("HYPERLIQUID_BASE_URL"), "Hyperliquid API base URL")
		testnet      = flag.Bool("testnet", false, "use Hyperliquid testnet")
		coin         = flag.String("coin", "", "perp coin")
		dex          = flag.String("dex", "", "perp dex name; empty string is the default perp dex")
		oid          = flag.Int("oid", 0, "order id to modify")
		oidCloidRaw  = flag.String("oid-cloid", "", "existing order cloid instead of oid")
		side         = flag.String("side", "buy", "buy or sell")
		size         = flag.Float64("size", 0, "new order size")
		price        = flag.Float64("price", 0, "new limit price")
		tif          = flag.String("tif", "Gtc", "time in force: Gtc, Ioc, or Alo")
		reduceOnly   = flag.Bool("reduce-only", false, "new reduce-only value")
		newCloidRaw  = flag.String("new-cloid", "", "optional new client order id, 16-byte hex string")
		vaultAddress = flag.String("vault-address", os.Getenv("HYPERLIQUID_VAULT_ADDRESS"), "optional vault/subaccount address")
		confirm      = flag.Bool("confirm", false, "actually modify the order")
		timeout      = flag.Duration("timeout", 20*time.Second, "HTTP timeout")
	)
	flag.Parse()
	runModify(*privateKey, *baseURL, *testnet, *coin, *dex, *oid, *oidCloidRaw, *side, *size, *price, *tif, *reduceOnly, *newCloidRaw, clientutil.OptionalString(vaultAddress), *confirm, *timeout)
}

func runModify(privateKey string, baseURL string, testnet bool, coin string, dex string, oid int, oidCloidRaw string, side string, size float64, price float64, tif string, reduceOnly bool, newCloidRaw string, vault *string, confirm bool, timeout time.Duration) {
	clientutil.RequirePrivateKey(privateKey)
	clientutil.RequireCoin(coin)
	if oid == 0 && oidCloidRaw == "" {
		clientutil.ExitUsage("pass -oid or -oid-cloid")
	}
	if size <= 0 {
		clientutil.ExitUsage("-size must be greater than 0")
	}
	if price <= 0 {
		clientutil.ExitUsage("-price must be greater than 0")
	}
	isBuy, err := clientutil.ParseSide(side)
	if err != nil {
		clientutil.ExitUsage(err.Error())
	}
	if !clientutil.ValidTIF(tif) {
		clientutil.ExitUsage("-tif must be one of Gtc, Ioc, Alo")
	}
	if !confirm {
		fmt.Fprintln(os.Stderr, "refusing to modify without -confirm")
		os.Exit(2)
	}
	orderID, err := execution.ParseOrderID(oid, oidCloidRaw)
	if err != nil {
		clientutil.ExitErr("order cloid", err)
	}
	var newCloid *execution.Cloid
	if newCloidRaw != "" {
		parsed, err := execution.NewCloid(newCloidRaw)
		if err != nil {
			clientutil.ExitErr("new cloid", err)
		}
		newCloid = &parsed
	}
	base := clientutil.ResolveBaseURL(baseURL, testnet)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	client := execution.New(execution.Config{
		BaseURL:      base,
		Timeout:      timeout,
		PrivateKey:   privateKey,
		Dex:          dex,
		VaultAddress: vault,
	})
	response, err := client.ModifyPerpOrder(ctx, execution.ModifyOrderRequest{
		OrderID:    orderID,
		Coin:       coin,
		IsBuy:      isBuy,
		Size:       size,
		Price:      price,
		TIF:        tif,
		ReduceOnly: reduceOnly,
		Cloid:      newCloid,
	})
	if err != nil {
		clientutil.ExitErr("perp modify order", err)
	}
	clientutil.PrintJSON(response)
}
