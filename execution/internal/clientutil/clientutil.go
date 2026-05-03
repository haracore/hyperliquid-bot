package clientutil

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"hyperliquid-bot/execution/credentials"
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
		ExitUsage("missing private key: pass -private-key or configure execution secrets")
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

type SecretFlags struct {
	Provider       *string
	Account        *string
	Prefix         *string
	VaultAddress   *string
	VaultToken     *string
	VaultNamespace *string
	VaultMount     *string
	VaultPrefix    *string
}

func AddSecretFlags() SecretFlags {
	return SecretFlags{
		Provider:       flag.String("secret-provider", envDefault("HYPERLIQUID_SECRET_PROVIDER", credentials.ProviderEnv), "secret provider: env or vault"),
		Account:        flag.String("account", envDefault("HYPERLIQUID_ACCOUNT", "main"), "secret account name"),
		Prefix:         flag.String("secret-prefix", envDefault("HYPERLIQUID_SECRET_PREFIX", "accounts"), "secret account path prefix"),
		VaultAddress:   flag.String("vault-addr", os.Getenv("VAULT_ADDR"), "Vault address"),
		VaultToken:     flag.String("vault-token", os.Getenv("VAULT_TOKEN"), "Vault token"),
		VaultNamespace: flag.String("vault-namespace", os.Getenv("VAULT_NAMESPACE"), "Vault namespace"),
		VaultMount:     flag.String("vault-mount", envDefault("VAULT_MOUNT", "secret"), "Vault KV v2 mount"),
		VaultPrefix:    flag.String("vault-prefix", os.Getenv("VAULT_PREFIX"), "Vault path prefix"),
	}
}

func ResolveAccount(ctx context.Context, flags SecretFlags, privateKeyOverride string, addressOverride string, vaultOverride string, timeout time.Duration) credentials.Account {
	resolveCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	config := credentials.ProviderConfig{
		Name:           value(flags.Provider),
		Account:        value(flags.Account),
		Prefix:         value(flags.Prefix),
		VaultAddress:   value(flags.VaultAddress),
		VaultToken:     value(flags.VaultToken),
		VaultNamespace: value(flags.VaultNamespace),
		VaultMount:     value(flags.VaultMount),
		VaultPrefix:    value(flags.VaultPrefix),
	}
	account, err := credentials.ResolveAccountFields(resolveCtx, config)
	if err != nil {
		ExitErr("resolve execution secrets", err)
	}
	account = credentials.ApplyOverrides(account, privateKeyOverride, addressOverride, vaultOverride)
	RequirePrivateKey(account.PrivateKey)
	return account
}

func ResolveAccountFields(ctx context.Context, flags SecretFlags, privateKeyOverride string, addressOverride string, vaultOverride string, timeout time.Duration) credentials.Account {
	resolveCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	account, err := credentials.ResolveAccountFields(resolveCtx, credentials.ProviderConfig{
		Name:           value(flags.Provider),
		Account:        value(flags.Account),
		Prefix:         value(flags.Prefix),
		VaultAddress:   value(flags.VaultAddress),
		VaultToken:     value(flags.VaultToken),
		VaultNamespace: value(flags.VaultNamespace),
		VaultMount:     value(flags.VaultMount),
		VaultPrefix:    value(flags.VaultPrefix),
	})
	if err != nil {
		ExitErr("resolve execution secrets", err)
	}
	return credentials.ApplyOverrides(account, privateKeyOverride, addressOverride, vaultOverride)
}

func envDefault(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func value(flagValue *string) string {
	if flagValue == nil {
		return ""
	}
	return *flagValue
}
