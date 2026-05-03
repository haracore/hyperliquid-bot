package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"hyperliquid-bot/execution/credentials"
)

type Config struct {
	Server      ServerConfig
	Hyperliquid HyperliquidConfig
	Secrets     SecretsConfig
	Overrides   OverridesConfig
}

type ServerConfig struct {
	Listen string
}

type HyperliquidConfig struct {
	BaseURL string
	Testnet bool
	Timeout time.Duration
}

type SecretsConfig struct {
	Provider      string
	Account       string
	Prefix        string
	Vault         VaultConfig
	VaultUserpass VaultUserpassConfig
}

type VaultConfig struct {
	Addr      string
	Token     string
	TokenEnv  string
	Namespace string
	Mount     string
	Prefix    string
}

type VaultUserpassConfig struct {
	Addr         string
	Username     string
	UsernameEnv  string
	Password     string
	PasswordEnv  string
	MFA          string
	MFAEnv       string
	MFAMethod    string
	MFAMethodEnv string
	OTP          string
	OTPEnv       string
	Namespace    string
	AuthMount    string
	Mount        string
	Prefix       string
}

type OverridesConfig struct {
	Address      string
	PrivateKey   string
	VaultAddress string
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			Listen: ":8080",
		},
		Hyperliquid: HyperliquidConfig{
			BaseURL: os.Getenv("HYPERLIQUID_BASE_URL"),
			Timeout: 20 * time.Second,
		},
		Secrets: SecretsConfig{
			Provider: credentials.ProviderEnv,
			Account:  envDefault("HYPERLIQUID_ACCOUNT", "main"),
			Prefix:   envDefault("HYPERLIQUID_SECRET_PREFIX", "accounts"),
			Vault: VaultConfig{
				Addr:      os.Getenv("VAULT_ADDR"),
				TokenEnv:  "VAULT_TOKEN",
				Namespace: os.Getenv("VAULT_NAMESPACE"),
				Mount:     envDefault("VAULT_MOUNT", "secret"),
				Prefix:    os.Getenv("VAULT_PREFIX"),
			},
			VaultUserpass: VaultUserpassConfig{
				Addr:         os.Getenv("VAULT_ADDR"),
				UsernameEnv:  "VAULT_USERNAME",
				PasswordEnv:  "VAULT_PASSWORD",
				MFAEnv:       "VAULT_MFA",
				MFAMethodEnv: "VAULT_MFA_METHOD",
				OTPEnv:       "VAULT_OTP",
				Namespace:    os.Getenv("VAULT_NAMESPACE"),
				AuthMount:    envDefault("VAULT_AUTH_MOUNT", "userpass"),
				Mount:        envDefault("VAULT_MOUNT", "secret"),
				Prefix:       os.Getenv("VAULT_PREFIX"),
			},
		},
		Overrides: OverridesConfig{
			Address:      os.Getenv("HYPERLIQUID_ADDRESS"),
			PrivateKey:   "",
			VaultAddress: "",
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if strings.TrimSpace(path) == "" {
		return cfg, nil
	}
	if err := parseFile(path, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) ProviderConfig() credentials.ProviderConfig {
	return credentials.ProviderConfig{
		Name:           c.Secrets.Provider,
		Account:        c.Secrets.Account,
		Prefix:         c.Secrets.Prefix,
		VaultAddress:   c.vaultAddress(),
		VaultToken:     valueOrEnv(c.Secrets.Vault.Token, c.Secrets.Vault.TokenEnv),
		VaultNamespace: c.vaultNamespace(),
		VaultMount:     c.vaultMount(),
		VaultPrefix:    c.vaultPrefix(),
		VaultUsername:  valueOrEnv(c.Secrets.VaultUserpass.Username, c.Secrets.VaultUserpass.UsernameEnv),
		VaultPassword:  valueOrEnv(c.Secrets.VaultUserpass.Password, c.Secrets.VaultUserpass.PasswordEnv),
		VaultMFA:       valueOrEnv(c.Secrets.VaultUserpass.MFA, c.Secrets.VaultUserpass.MFAEnv),
		VaultMFAMethod: valueOrEnv(c.Secrets.VaultUserpass.MFAMethod, c.Secrets.VaultUserpass.MFAMethodEnv),
		VaultOTP:       valueOrEnv(c.Secrets.VaultUserpass.OTP, c.Secrets.VaultUserpass.OTPEnv),
		VaultAuthMount: c.Secrets.VaultUserpass.AuthMount,
	}
}

func (c Config) vaultAddress() string {
	if c.Secrets.Provider == credentials.ProviderVaultUserpass {
		return c.Secrets.VaultUserpass.Addr
	}
	return c.Secrets.Vault.Addr
}

func (c Config) vaultNamespace() string {
	if c.Secrets.Provider == credentials.ProviderVaultUserpass {
		return c.Secrets.VaultUserpass.Namespace
	}
	return c.Secrets.Vault.Namespace
}

func (c Config) vaultMount() string {
	if c.Secrets.Provider == credentials.ProviderVaultUserpass {
		return c.Secrets.VaultUserpass.Mount
	}
	return c.Secrets.Vault.Mount
}

func (c Config) vaultPrefix() string {
	if c.Secrets.Provider == credentials.ProviderVaultUserpass {
		return c.Secrets.VaultUserpass.Prefix
	}
	return c.Secrets.Vault.Prefix
}

func parseFile(path string, cfg *Config) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	section := ""
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := stripComment(scanner.Text())
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s:%d: expected key = value", path, lineNumber)
		}
		if err := setValue(cfg, section, strings.TrimSpace(key), strings.TrimSpace(value)); err != nil {
			return fmt.Errorf("%s:%d: %w", path, lineNumber, err)
		}
	}
	return scanner.Err()
}

func setValue(cfg *Config, section string, key string, raw string) error {
	value, err := parseScalar(raw)
	if err != nil {
		return err
	}
	switch section {
	case "server":
		switch key {
		case "listen":
			cfg.Server.Listen = value
		default:
			return unknown(section, key)
		}
	case "hyperliquid":
		switch key {
		case "base_url":
			cfg.Hyperliquid.BaseURL = value
		case "testnet":
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("invalid bool %q", value)
			}
			cfg.Hyperliquid.Testnet = parsed
		case "timeout":
			parsed, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			cfg.Hyperliquid.Timeout = parsed
		default:
			return unknown(section, key)
		}
	case "secrets":
		switch key {
		case "provider":
			cfg.Secrets.Provider = value
		case "account":
			cfg.Secrets.Account = value
		case "prefix":
			cfg.Secrets.Prefix = value
		default:
			return unknown(section, key)
		}
	case "secrets.vault":
		switch key {
		case "addr":
			cfg.Secrets.Vault.Addr = value
		case "token":
			cfg.Secrets.Vault.Token = value
		case "token_env":
			cfg.Secrets.Vault.TokenEnv = value
		case "namespace":
			cfg.Secrets.Vault.Namespace = value
		case "mount":
			cfg.Secrets.Vault.Mount = value
		case "prefix":
			cfg.Secrets.Vault.Prefix = value
		default:
			return unknown(section, key)
		}
	case "secrets.vault_userpass":
		switch key {
		case "addr":
			cfg.Secrets.VaultUserpass.Addr = value
		case "username":
			cfg.Secrets.VaultUserpass.Username = value
		case "username_env":
			cfg.Secrets.VaultUserpass.UsernameEnv = value
		case "password":
			cfg.Secrets.VaultUserpass.Password = value
		case "password_env":
			cfg.Secrets.VaultUserpass.PasswordEnv = value
		case "mfa":
			cfg.Secrets.VaultUserpass.MFA = value
		case "mfa_env":
			cfg.Secrets.VaultUserpass.MFAEnv = value
		case "mfa_method":
			cfg.Secrets.VaultUserpass.MFAMethod = value
		case "mfa_method_env":
			cfg.Secrets.VaultUserpass.MFAMethodEnv = value
		case "otp":
			cfg.Secrets.VaultUserpass.OTP = value
		case "otp_env":
			cfg.Secrets.VaultUserpass.OTPEnv = value
		case "namespace":
			cfg.Secrets.VaultUserpass.Namespace = value
		case "auth_mount":
			cfg.Secrets.VaultUserpass.AuthMount = value
		case "mount":
			cfg.Secrets.VaultUserpass.Mount = value
		case "prefix":
			cfg.Secrets.VaultUserpass.Prefix = value
		default:
			return unknown(section, key)
		}
	case "overrides":
		switch key {
		case "address":
			cfg.Overrides.Address = value
		case "private_key":
			cfg.Overrides.PrivateKey = value
		case "vault_address":
			cfg.Overrides.VaultAddress = value
		default:
			return unknown(section, key)
		}
	default:
		return fmt.Errorf("unknown section %q", section)
	}
	return nil
}

func parseScalar(raw string) (string, error) {
	if strings.HasPrefix(raw, "\"") {
		value, err := strconv.Unquote(raw)
		if err != nil {
			return "", err
		}
		return value, nil
	}
	switch raw {
	case "true", "false":
		return raw, nil
	default:
		return strings.TrimSpace(raw), nil
	}
}

func stripComment(line string) string {
	inString := false
	escaped := false
	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && inString {
			escaped = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if r == '#' && !inString {
			return line[:i]
		}
	}
	return line
}

func unknown(section string, key string) error {
	return fmt.Errorf("unknown key %q in [%s]", key, section)
}

func valueOrEnv(value string, envName string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	if strings.TrimSpace(envName) == "" {
		return ""
	}
	return os.Getenv(envName)
}

func envDefault(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}
