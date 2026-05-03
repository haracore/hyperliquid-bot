package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"hyperliquid-bot/sdk/constants"
	hlinfo "hyperliquid-bot/sdk/info"
)

func main() {
	var (
		address = flag.String("address", os.Getenv("HYPERLIQUID_ADDRESS"), "user address; can also be set with HYPERLIQUID_ADDRESS")
		baseURL = flag.String("base-url", os.Getenv("HYPERLIQUID_BASE_URL"), "Hyperliquid API base URL")
		testnet = flag.Bool("testnet", false, "use Hyperliquid testnet")
		timeout = flag.Duration("timeout", 15*time.Second, "HTTP timeout")
	)
	flag.Parse()

	if strings.TrimSpace(*address) == "" {
		fmt.Fprintln(os.Stderr, "missing address: pass -address or set HYPERLIQUID_ADDRESS")
		os.Exit(2)
	}
	if *baseURL == "" {
		*baseURL = constants.MainnetAPIURL
	}
	if *testnet {
		*baseURL = constants.TestnetAPIURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	info := hlinfo.New(*baseURL, *timeout)

	var perpState map[string]any
	if err := info.UserState(ctx, *address, "", &perpState); err != nil {
		exitErr("perp state", err)
	}
	var spotState map[string]any
	if err := info.SpotUserState(ctx, *address, &spotState); err != nil {
		exitErr("spot state", err)
	}

	printHeader("Hyperliquid balances")
	fmt.Printf("Address: %s\n", *address)
	fmt.Printf("API:     %s\n", *baseURL)

	printPerpSummary(perpState)
	printSpotBalances(spotState)
	printPerpPositions(perpState)
}

func exitErr(label string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", label, err)
	os.Exit(1)
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

	positions, ok := state["assetPositions"].([]any)
	if !ok || len(positions) == 0 {
		fmt.Println("(none)")
		return
	}

	printed := false
	for _, raw := range positions {
		wrapper, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		position, ok := wrapper["position"].(map[string]any)
		if !ok {
			continue
		}
		szi := fmt.Sprint(position["szi"])
		if szi == "0" || szi == "0.0" {
			continue
		}
		printed = true
		fmt.Printf(
			"%-12s szi=%s value=%s unrealizedPnl=%s marginUsed=%s\n",
			position["coin"],
			position["szi"],
			position["positionValue"],
			position["unrealizedPnl"],
			position["marginUsed"],
		)
	}
	if !printed {
		fmt.Println("(none)")
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
