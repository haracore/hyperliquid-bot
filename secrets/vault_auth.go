package secrets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type VaultToken struct {
	ClientToken   string   `json:"client_token"`
	Accessor      string   `json:"accessor,omitempty"`
	Policies      []string `json:"policies,omitempty"`
	LeaseDuration int      `json:"lease_duration,omitempty"`
	Renewable     bool     `json:"renewable,omitempty"`
}

type VaultUserpassConfig struct {
	Address    string
	Username   string
	Password   string
	MFA        string
	MFAMethod  string
	OTP        string
	Namespace  string
	Mount      string
	HTTPClient *http.Client
}

type VaultUserpassAuthenticator struct {
	address    string
	username   string
	password   string
	mfa        string
	mfaMethod  string
	otp        string
	namespace  string
	mount      string
	httpClient *http.Client
}

func NewVaultUserpassAuthenticator(config VaultUserpassConfig) *VaultUserpassAuthenticator {
	client := config.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	mount := cleanPath(config.Mount)
	if mount == "." || mount == "" {
		mount = "userpass"
	}
	return &VaultUserpassAuthenticator{
		address:    strings.TrimRight(strings.TrimSpace(config.Address), "/"),
		username:   strings.TrimSpace(config.Username),
		password:   config.Password,
		mfa:        strings.TrimSpace(config.MFA),
		mfaMethod:  strings.TrimSpace(config.MFAMethod),
		otp:        strings.TrimSpace(config.OTP),
		namespace:  strings.TrimSpace(config.Namespace),
		mount:      mount,
		httpClient: client,
	}
}

func (a *VaultUserpassAuthenticator) Login(ctx context.Context) (VaultToken, error) {
	if a.address == "" {
		return VaultToken{}, fmt.Errorf("vault address is required")
	}
	if a.username == "" {
		return VaultToken{}, fmt.Errorf("vault username is required")
	}
	if a.password == "" {
		return VaultToken{}, fmt.Errorf("vault password is required")
	}
	requestURL, err := a.loginURL()
	if err != nil {
		return VaultToken{}, err
	}
	body, err := json.Marshal(map[string]string{"password": a.password})
	if err != nil {
		return VaultToken{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return VaultToken{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	if a.namespace != "" {
		request.Header.Set("X-Vault-Namespace", a.namespace)
	}
	if mfa := a.mfaHeader(); mfa != "" {
		request.Header.Set("X-Vault-MFA", mfa)
	}

	response, err := a.httpClient.Do(request)
	if err != nil {
		return VaultToken{}, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return VaultToken{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return VaultToken{}, fmt.Errorf("vault userpass login: status %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var payload vaultLoginResponse
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return VaultToken{}, err
	}
	if payload.Auth.ClientToken == "" {
		return VaultToken{}, fmt.Errorf("vault userpass login did not return client token")
	}
	return payload.Auth, nil
}

func (a *VaultUserpassAuthenticator) mfaHeader() string {
	if a.mfa != "" {
		return a.mfa
	}
	if a.otp == "" {
		return ""
	}
	if a.mfaMethod != "" {
		return a.mfaMethod + ":" + a.otp
	}
	return a.otp
}

func (a *VaultUserpassAuthenticator) loginURL() (string, error) {
	base, err := url.Parse(a.address)
	if err != nil {
		return "", err
	}
	base.Path = path.Join(base.Path, "v1", "auth", a.mount, "login", a.username)
	return base.String(), nil
}

type vaultLoginResponse struct {
	Auth VaultToken `json:"auth"`
}

type VaultUserpassProvider struct {
	authenticator *VaultUserpassAuthenticator
	config        VaultConfig
}

func NewVaultUserpassProvider(authConfig VaultUserpassConfig, providerConfig VaultConfig) *VaultUserpassProvider {
	return &VaultUserpassProvider{
		authenticator: NewVaultUserpassAuthenticator(authConfig),
		config:        providerConfig,
	}
}

func (p *VaultUserpassProvider) Get(ctx context.Context, ref Ref) (Bundle, error) {
	token, err := p.authenticator.Login(ctx)
	if err != nil {
		return Bundle{}, err
	}
	config := p.config
	config.Address = defaultString(config.Address, p.authenticator.address)
	config.Namespace = defaultString(config.Namespace, p.authenticator.namespace)
	config.Token = token.ClientToken
	return NewVaultProvider(config).Get(ctx, ref)
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
