package secrets

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
)

var (
	ErrNotFound     = errors.New("secret not found")
	ErrMissingField = errors.New("secret field missing")
)

type Provider interface {
	Get(ctx context.Context, ref Ref) (Bundle, error)
}

type Ref struct {
	Path    string
	Version int
}

type Bundle struct {
	Path    string
	Version int
	Fields  map[string]string
}

func (b Bundle) Get(field string) string {
	if b.Fields == nil {
		return ""
	}
	return b.Fields[field]
}

func (b Bundle) Require(field string) (string, error) {
	value := strings.TrimSpace(b.Get(field))
	if value == "" {
		return "", fmt.Errorf("%w: %s", ErrMissingField, field)
	}
	return value, nil
}

type Account struct {
	Name         string
	Address      string
	PrivateKey   string
	VaultAddress string
	Fields       map[string]string
}

type AccountResolver struct {
	provider Provider
	prefix   string
}

func NewAccountResolver(provider Provider, prefix string) *AccountResolver {
	return &AccountResolver{
		provider: provider,
		prefix:   cleanPath(prefix),
	}
}

func (r *AccountResolver) Account(ctx context.Context, name string) (Account, error) {
	name = cleanPath(name)
	if name == "" {
		return Account{}, fmt.Errorf("account name is required")
	}
	bundle, err := r.provider.Get(ctx, Ref{Path: joinPath(r.prefix, name)})
	if err != nil {
		return Account{}, err
	}
	privateKey, err := bundle.Require("private_key")
	if err != nil {
		return Account{}, err
	}
	return Account{
		Name:         name,
		Address:      bundle.Get("address"),
		PrivateKey:   privateKey,
		VaultAddress: bundle.Get("vault_address"),
		Fields:       cloneMap(bundle.Fields),
	}, nil
}

func cleanPath(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "/")
	return path.Clean(value)
}

func joinPath(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = cleanPath(part)
		if part != "." && part != "" {
			cleaned = append(cleaned, part)
		}
	}
	if len(cleaned) == 0 {
		return ""
	}
	return path.Join(cleaned...)
}

func cloneMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
