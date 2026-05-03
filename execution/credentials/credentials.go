package credentials

import (
	"context"
	"fmt"
	"strings"

	rootsecrets "hyperliquid-bot/secrets"
)

const (
	ProviderEnv   = "env"
	ProviderVault = "vault"
)

type ProviderConfig struct {
	Name           string
	Account        string
	Prefix         string
	VaultAddress   string
	VaultToken     string
	VaultNamespace string
	VaultMount     string
	VaultPrefix    string
}

type Account struct {
	Name         string
	Address      string
	PrivateKey   string
	VaultAddress string
}

func ResolveAccount(ctx context.Context, config ProviderConfig) (Account, error) {
	accountName := defaultString(config.Account, "main")
	prefix := defaultString(config.Prefix, "accounts")

	provider, err := NewProvider(config)
	if err != nil {
		return Account{}, err
	}
	resolver := rootsecrets.NewAccountResolver(provider, prefix)
	account, err := resolver.Account(ctx, accountName)
	if err != nil {
		return Account{}, err
	}
	return Account{
		Name:         account.Name,
		Address:      account.Address,
		PrivateKey:   account.PrivateKey,
		VaultAddress: account.VaultAddress,
	}, nil
}

func ResolveAccountFields(ctx context.Context, config ProviderConfig) (Account, error) {
	accountName := defaultString(config.Account, "main")
	prefix := defaultString(config.Prefix, "accounts")

	provider, err := NewProvider(config)
	if err != nil {
		return Account{}, err
	}
	bundle, err := provider.Get(ctx, rootsecrets.Ref{Path: joinPath(prefix, accountName)})
	if err != nil {
		return Account{}, err
	}
	return Account{
		Name:         accountName,
		Address:      bundle.Get("address"),
		PrivateKey:   bundle.Get("private_key"),
		VaultAddress: bundle.Get("vault_address"),
	}, nil
}

func NewProvider(config ProviderConfig) (rootsecrets.Provider, error) {
	switch defaultString(config.Name, ProviderEnv) {
	case ProviderEnv:
		accountName := defaultString(config.Account, "main")
		prefix := defaultString(config.Prefix, "accounts")
		secretPath := joinPath(prefix, accountName)
		return rootsecrets.NewEnvProvider(rootsecrets.EnvConfig{
			PathFields: map[string]rootsecrets.EnvFields{
				secretPath: {
					"address":       "HYPERLIQUID_ADDRESS",
					"private_key":   "HYPERLIQUID_PRIVATE_KEY",
					"vault_address": "HYPERLIQUID_VAULT_ADDRESS",
				},
			},
		}), nil
	case ProviderVault:
		return rootsecrets.NewVaultProvider(rootsecrets.VaultConfig{
			Address:   config.VaultAddress,
			Token:     config.VaultToken,
			Namespace: config.VaultNamespace,
			Mount:     defaultString(config.VaultMount, "secret"),
			Prefix:    config.VaultPrefix,
		}), nil
	default:
		return nil, fmt.Errorf("secret provider must be env or vault")
	}
}

func ApplyOverrides(account Account, privateKey string, address string, vaultAddress string) Account {
	if strings.TrimSpace(privateKey) != "" {
		account.PrivateKey = privateKey
	}
	if strings.TrimSpace(address) != "" {
		account.Address = address
	}
	if strings.TrimSpace(vaultAddress) != "" {
		account.VaultAddress = vaultAddress
	}
	return account
}

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func joinPath(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(strings.TrimSpace(part), "/")
		if part != "" && part != "." {
			cleaned = append(cleaned, part)
		}
	}
	return strings.Join(cleaned, "/")
}
