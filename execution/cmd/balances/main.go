package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	execution "hyperliquid-bot/execution/client"
	"hyperliquid-bot/execution/internal/clientutil"
)

func main() {
	var (
		address = flag.String("address", "", "user address; overrides execution secrets")
		baseURL = flag.String("base-url", os.Getenv("HYPERLIQUID_BASE_URL"), "Hyperliquid API base URL")
		testnet = flag.Bool("testnet", false, "use Hyperliquid testnet")
		timeout = flag.Duration("timeout", 15*time.Second, "HTTP timeout")
	)
	secretFlags := clientutil.AddSecretFlags()
	flag.Parse()

	account := clientutil.ResolveAccountFields(context.Background(), secretFlags, "", *address, "", *timeout)
	clientutil.RequireAddress(account.Address)
	base := clientutil.ResolveBaseURL(*baseURL, *testnet)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	client := execution.New(execution.Config{BaseURL: base, Timeout: *timeout})
	result, err := client.Balances(ctx, account.Address)
	if err != nil {
		clientutil.ExitErr("balances", err)
	}

	printHeader("Hyperliquid balances")
	fmt.Printf("Address: %s\n", account.Address)
	fmt.Printf("API:     %s\n", base)

	printPerpSummary(result.PerpState)
	printSpotBalances(result.SpotState)
	printPerpPositions(result.PerpState)
}

func printHeader(title string) {
	fmt.Println(title)
	fmt.Println(strings.Repeat("=", len(title)))
}

func printPerpSummary(state map[string]any) {
	fmt.Println()
	fmt.Println("Perp account")
	fmt.Println("------------")
	printKeyValue("withdrawable", state["withdrawable"])

	if summary, ok := state["marginSummary"].(map[string]any); ok {
		keys := []string{"accountValue", "totalRawUsd", "totalMarginUsed", "totalNtlPos"}
		for _, key := range keys {
			printKeyValue(key, summary[key])
		}
	}
}

func printSpotBalances(state map[string]any) {
	fmt.Println()
	fmt.Println("Spot balances")
	fmt.Println("-------------")

	balances, ok := state["balances"].([]any)
	if !ok || len(balances) == 0 {
		fmt.Println("(none)")
		return
	}

	rows := make([]map[string]any, 0, len(balances))
	for _, raw := range balances {
		row, ok := raw.(map[string]any)
		if ok {
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		return fmt.Sprint(rows[i]["coin"]) < fmt.Sprint(rows[j]["coin"])
	})

	for _, row := range rows {
		coin := fmt.Sprint(row["coin"])
		total := firstPresent(row, "total", "balance")
		hold := firstPresent(row, "hold", "holdBalance")
		entry := fmt.Sprintf("%-12s total=%s", coin, total)
		if hold != "" && hold != "<nil>" {
			entry += " hold=" + hold
		}
		fmt.Println(entry)
	}
}

func printPerpPositions(state map[string]any) {
	fmt.Println()
	fmt.Println("Perp positions")
	fmt.Println("--------------")

	positions := execution.ExtractPerpPositions(state, false)
	if len(positions) == 0 {
		fmt.Println("(none)")
		return
	}

	for _, position := range positions {
		fmt.Printf(
			"%-12s szi=%s value=%s unrealizedPnl=%s marginUsed=%s\n",
			position.Coin,
			position.Szi,
			position.PositionValue,
			position.UnrealizedPnl,
			position.MarginUsed,
		)
	}
}

func printKeyValue(key string, value any) {
	if value == nil {
		return
	}
	fmt.Printf("%-16s %v\n", key+":", value)
}

func firstPresent(row map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := row[key]; ok && value != nil {
			return fmt.Sprint(value)
		}
	}
	return ""
}
