package secrets

import (
	"context"
	"fmt"
	"os"
)

type EnvConfig struct {
	PathFields map[string]EnvFields
	Lookup     func(string) (string, bool)
}

type EnvFields map[string]string

type EnvProvider struct {
	pathFields map[string]EnvFields
	lookup     func(string) (string, bool)
}

func NewEnvProvider(config EnvConfig) *EnvProvider {
	lookup := config.Lookup
	if lookup == nil {
		lookup = os.LookupEnv
	}
	pathFields := make(map[string]EnvFields, len(config.PathFields))
	for secretPath, fields := range config.PathFields {
		copied := make(EnvFields, len(fields))
		for field, envName := range fields {
			copied[field] = envName
		}
		pathFields[cleanPath(secretPath)] = copied
	}
	return &EnvProvider{pathFields: pathFields, lookup: lookup}
}

func (p *EnvProvider) Get(_ context.Context, ref Ref) (Bundle, error) {
	secretPath := cleanPath(ref.Path)
	fields, ok := p.pathFields[secretPath]
	if !ok {
		return Bundle{}, fmt.Errorf("%w: %s", ErrNotFound, secretPath)
	}
	values := make(map[string]string, len(fields))
	for field, envName := range fields {
		if value, ok := p.lookup(envName); ok {
			values[field] = value
		}
	}
	return Bundle{Path: secretPath, Fields: values}, nil
}

func HyperliquidEnvProvider(account string) *EnvProvider {
	return NewEnvProvider(EnvConfig{
		PathFields: map[string]EnvFields{
			joinPath("accounts", account): {
				"address":       "HYPERLIQUID_ADDRESS",
				"private_key":   "HYPERLIQUID_PRIVATE_KEY",
				"vault_address": "HYPERLIQUID_VAULT_ADDRESS",
			},
		},
	})
}
