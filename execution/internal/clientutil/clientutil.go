package clientutil

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"hyperliquid-bot/sdk/constants"
	"hyperliquid-bot/sdk/exchange"
	hlinfo "hyperliquid-bot/sdk/info"
	"hyperliquid-bot/sdk/signing"
	"hyperliquid-bot/sdk/types"
)

func ResolveBaseURL(baseURL string, testnet bool) string {
	if testnet {
		return constants.TestnetAPIURL
	}
	if baseURL != "" {
		return baseURL
	}
	return constants.MainnetAPIURL
}

func ExitUsage(message string) {
	fmt.Fprintln(os.Stderr, message)
	flag.Usage()
	os.Exit(2)
}

func ExitErr(label string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", label, err)
	os.Exit(1)
}

func RequireAddress(address string) {
	if strings.TrimSpace(address) == "" {
		ExitUsage("missing address: pass -address or set HYPERLIQUID_ADDRESS")
	}
}

func RequirePrivateKey(privateKey string) {
	if strings.TrimSpace(privateKey) == "" {
		ExitUsage("missing private key: pass -private-key or set HYPERLIQUID_PRIVATE_KEY")
	}
}

func RequireCoin(coin string) {
	if strings.TrimSpace(coin) == "" {
		ExitUsage("missing -coin")
	}
}

func OptionalString(value *string) *string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return value
}

func ParseSide(side string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "buy", "b":
		return true, nil
	case "sell", "s":
		return false, nil
	default:
		return false, fmt.Errorf("-side must be buy or sell")
	}
}

func ValidTIF(tif string) bool {
	switch tif {
	case "Gtc", "Ioc", "Alo":
		return true
	default:
		return false
	}
}

func PrintJSON(value any) {
	pretty, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		ExitErr("format response", err)
	}
	fmt.Println(string(pretty))
}

func InitializedInfo(ctx context.Context, baseURL string, dex string, timeout time.Duration) *hlinfo.Info {
	perpDexs := []string{""}
	if strings.TrimSpace(dex) != "" {
		perpDexs = []string{dex}
	}
	info, err := hlinfo.NewInitialized(ctx, baseURL, true, nil, nil, perpDexs, timeout)
	if err != nil {
		ExitErr("initialize info metadata", err)
	}
	return info
}

func NewExchange(ctx context.Context, privateKey string, baseURL string, dex string, vaultAddress *string, timeout time.Duration) *exchange.Exchange {
	wallet, err := signing.PrivateKeyFromHex(privateKey)
	if err != nil {
		ExitErr("private key", err)
	}
	info := InitializedInfo(ctx, baseURL, dex, timeout)
	ex := exchange.New(wallet, baseURL, timeout, vaultAddress, nil)
	ex.Info = info
	return ex
}

func ParseCloid(raw string, label string) types.Cloid {
	cloid, err := types.NewCloid(raw)
	if err != nil {
		ExitErr(label, err)
	}
	return cloid
}

func ParseOrderID(oid int, cloidRaw string) exchange.OidOrCloid {
	if strings.TrimSpace(cloidRaw) != "" {
		cloid := ParseCloid(cloidRaw, "order cloid")
		return cloid
	}
	return oid
}
