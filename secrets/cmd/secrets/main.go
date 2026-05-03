package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"hyperliquid-bot/secrets"
)

const usage = `Usage:
  secrets get [flags]
  secrets account [flags]
  secrets login-vault [flags]

Commands:
  get          Read a generic secret bundle
  account      Read an account bundle via AccountResolver
  login-vault  Login to Vault and print the issued client token

Use "secrets <command> -h" for command-specific flags.
`

type outputBundle struct {
	Path    string            `json:"path"`
	Version int               `json:"version,omitempty"`
	Fields  map[string]string `json:"fields"`
}

type outputAccount struct {
	Name         string            `json:"name"`
	Address      string            `json:"address,omitempty"`
	PrivateKey   string            `json:"private_key"`
	VaultAddress string            `json:"vault_address,omitempty"`
	Fields       map[string]string `json:"fields,omitempty"`
}

type repeatedFlags []string

func (f *repeatedFlags) String() string {
	return strings.Join(*f, ",")
}

func (f *repeatedFlags) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "get":
		err = runGet(os.Args[2:])
	case "account":
		err = runAccount(os.Args[2:])
	case "login-vault":
		err = runLoginVault(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runGet(args []string) error {
	var envFields repeatedFlags
	fs := flag.NewFlagSet("get", flag.ExitOnError)
	providerName := fs.String("provider", "env", "secret provider: env or vault")
	secretPath := fs.String("path", "accounts/main", "secret path")
	version := fs.Int("version", 0, "secret version; only supported by vault")
	timeout := fs.Duration("timeout", 10*time.Second, "provider timeout")
	reveal := fs.Bool("reveal", false, "print sensitive fields without redaction")
	fs.Var(&envFields, "env-field", "env mapping field=ENV_NAME; repeatable and used by env provider")
	auth := addVaultUserpassFlags(fs)
	addVaultFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	provider, err := buildProvider(*providerName, *secretPath, envFields, auth.config())
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	bundle, err := provider.Get(ctx, secrets.Ref{Path: *secretPath, Version: *version})
	if err != nil {
		return err
	}
	return printJSON(outputBundle{
		Path:    bundle.Path,
		Version: bundle.Version,
		Fields:  redactMap(bundle.Fields, *reveal),
	})
}

func runAccount(args []string) error {
	var envFields repeatedFlags
	fs := flag.NewFlagSet("account", flag.ExitOnError)
	providerName := fs.String("provider", "env", "secret provider: env or vault")
	account := fs.String("account", "main", "account name")
	prefix := fs.String("prefix", "accounts", "account path prefix")
	timeout := fs.Duration("timeout", 10*time.Second, "provider timeout")
	reveal := fs.Bool("reveal", false, "print sensitive fields without redaction")
	fs.Var(&envFields, "env-field", "env mapping field=ENV_NAME; repeatable and used by env provider")
	auth := addVaultUserpassFlags(fs)
	addVaultFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	secretPath := joinPath(*prefix, *account)
	provider, err := buildProvider(*providerName, secretPath, envFields, auth.config())
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	resolver := secrets.NewAccountResolver(provider, *prefix)
	resolved, err := resolver.Account(ctx, *account)
	if err != nil {
		return err
	}
	privateKey := resolved.PrivateKey
	if !*reveal {
		privateKey = redact(privateKey)
	}
	return printJSON(outputAccount{
		Name:         resolved.Name,
		Address:      resolved.Address,
		PrivateKey:   privateKey,
		VaultAddress: resolved.VaultAddress,
		Fields:       redactMap(resolved.Fields, *reveal),
	})
}

func runLoginVault(args []string) error {
	fs := flag.NewFlagSet("login-vault", flag.ExitOnError)
	timeout := fs.Duration("timeout", 10*time.Second, "login timeout")
	reveal := fs.Bool("reveal", false, "print the issued client token without redaction")
	auth := addVaultUserpassFlags(fs)
	addVaultFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	token, err := secrets.NewVaultUserpassAuthenticator(auth.config()).Login(ctx)
	if err != nil {
		return err
	}
	if token.ClientToken != "" {
		if !*reveal {
			token.ClientToken = redact(token.ClientToken)
		}
	}
	return printJSON(token)
}

func buildProvider(providerName string, secretPath string, envFields repeatedFlags, authConfig secrets.VaultUserpassConfig) (secrets.Provider, error) {
	switch providerName {
	case "env":
		fields, err := parseEnvFields(envFields)
		if err != nil {
			return nil, err
		}
		if len(fields) == 0 {
			fields = hyperliquidEnvFields()
		}
		return secrets.NewEnvProvider(secrets.EnvConfig{
			PathFields: map[string]secrets.EnvFields{
				secretPath: fields,
			},
		}), nil
	case "vault":
		return secrets.NewVaultProvider(secrets.VaultConfig{
			Address:   os.Getenv("VAULT_ADDR"),
			Token:     os.Getenv("VAULT_TOKEN"),
			Namespace: os.Getenv("VAULT_NAMESPACE"),
			Mount:     envDefault("VAULT_MOUNT", "secret"),
			Prefix:    os.Getenv("VAULT_PREFIX"),
		}), nil
	case "vault-userpass":
		return secrets.NewVaultUserpassProvider(authConfig, secrets.VaultConfig{
			Address:   os.Getenv("VAULT_ADDR"),
			Namespace: os.Getenv("VAULT_NAMESPACE"),
			Mount:     envDefault("VAULT_MOUNT", "secret"),
			Prefix:    os.Getenv("VAULT_PREFIX"),
		}), nil
	default:
		return nil, fmt.Errorf("-provider must be env, vault, or vault-userpass")
	}
}

func addVaultFlags(fs *flag.FlagSet) {
	fs.Func("vault-addr", "Vault address; defaults to VAULT_ADDR", func(value string) error {
		return os.Setenv("VAULT_ADDR", value)
	})
	fs.Func("vault-token", "Vault token; defaults to VAULT_TOKEN", func(value string) error {
		return os.Setenv("VAULT_TOKEN", value)
	})
	fs.Func("vault-namespace", "Vault namespace; defaults to VAULT_NAMESPACE", func(value string) error {
		return os.Setenv("VAULT_NAMESPACE", value)
	})
	fs.Func("vault-mount", "Vault KV v2 mount; defaults to VAULT_MOUNT or secret", func(value string) error {
		return os.Setenv("VAULT_MOUNT", value)
	})
	fs.Func("vault-prefix", "Vault path prefix; defaults to VAULT_PREFIX", func(value string) error {
		return os.Setenv("VAULT_PREFIX", value)
	})
}

type vaultUserpassFlags struct {
	username *string
	password *string
	mfa      *string
	method   *string
	otp      *string
	mount    *string
}

func addVaultUserpassFlags(fs *flag.FlagSet) vaultUserpassFlags {
	return vaultUserpassFlags{
		username: fs.String("vault-username", os.Getenv("VAULT_USERNAME"), "Vault userpass username; defaults to VAULT_USERNAME"),
		password: fs.String("vault-password", os.Getenv("VAULT_PASSWORD"), "Vault userpass password; defaults to VAULT_PASSWORD"),
		mfa:      fs.String("vault-mfa", os.Getenv("VAULT_MFA"), "Full Vault MFA header value, e.g. method_id:123456; defaults to VAULT_MFA"),
		method:   fs.String("vault-mfa-method", os.Getenv("VAULT_MFA_METHOD"), "Vault MFA method ID/name used with -vault-otp"),
		otp:      fs.String("vault-otp", os.Getenv("VAULT_OTP"), "Vault login MFA OTP used with -vault-mfa-method"),
		mount:    fs.String("vault-auth-mount", envDefault("VAULT_AUTH_MOUNT", "userpass"), "Vault userpass auth mount"),
	}
}

func (f vaultUserpassFlags) config() secrets.VaultUserpassConfig {
	return secrets.VaultUserpassConfig{
		Address:   os.Getenv("VAULT_ADDR"),
		Username:  *f.username,
		Password:  *f.password,
		MFA:       *f.mfa,
		MFAMethod: *f.method,
		OTP:       *f.otp,
		Namespace: os.Getenv("VAULT_NAMESPACE"),
		Mount:     *f.mount,
	}
}

func parseEnvFields(values []string) (secrets.EnvFields, error) {
	fields := make(secrets.EnvFields)
	for _, value := range values {
		key, envName, ok := strings.Cut(value, "=")
		if !ok {
			return nil, fmt.Errorf("-env-field must be field=ENV_NAME")
		}
		key = strings.TrimSpace(key)
		envName = strings.TrimSpace(envName)
		if key == "" || envName == "" {
			return nil, fmt.Errorf("-env-field must be field=ENV_NAME")
		}
		fields[key] = envName
	}
	return fields, nil
}

func hyperliquidEnvFields() secrets.EnvFields {
	return secrets.EnvFields{
		"address":       "HYPERLIQUID_ADDRESS",
		"private_key":   "HYPERLIQUID_PRIVATE_KEY",
		"vault_address": "HYPERLIQUID_VAULT_ADDRESS",
	}
}

func redactMap(fields map[string]string, reveal bool) map[string]string {
	result := make(map[string]string, len(fields))
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := fields[key]
		if !reveal && sensitiveField(key) {
			value = redact(value)
		}
		result[key] = value
	}
	return result
}

func sensitiveField(field string) bool {
	normalized := strings.ToLower(field)
	return strings.Contains(normalized, "private") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "password")
}

func redact(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	if len(value) <= 10 {
		return "***"
	}
	return value[:6] + "..." + value[len(value)-4:]
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func envDefault(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
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
