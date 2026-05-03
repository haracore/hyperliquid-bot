package clientutil

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"hyperliquid-bot/sdk/constants"
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
