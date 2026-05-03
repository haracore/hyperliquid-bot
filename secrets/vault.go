package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
)

type VaultConfig struct {
	Address    string
	Token      string
	Namespace  string
	Mount      string
	Prefix     string
	HTTPClient *http.Client
}

type VaultProvider struct {
	address    string
	token      string
	namespace  string
	mount      string
	prefix     string
	httpClient *http.Client
}

func NewVaultProvider(config VaultConfig) *VaultProvider {
	client := config.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	mount := cleanPath(config.Mount)
	if mount == "." || mount == "" {
		mount = "secret"
	}
	return &VaultProvider{
		address:    strings.TrimRight(strings.TrimSpace(config.Address), "/"),
		token:      strings.TrimSpace(config.Token),
		namespace:  strings.TrimSpace(config.Namespace),
		mount:      mount,
		prefix:     cleanPath(config.Prefix),
		httpClient: client,
	}
}

func (p *VaultProvider) Get(ctx context.Context, ref Ref) (Bundle, error) {
	if p.address == "" {
		return Bundle{}, fmt.Errorf("vault address is required")
	}
	if p.token == "" {
		return Bundle{}, fmt.Errorf("vault token is required")
	}
	secretPath := joinPath(p.prefix, ref.Path)
	requestURL, err := p.secretURL(secretPath, ref.Version)
	if err != nil {
		return Bundle{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return Bundle{}, err
	}
	request.Header.Set("X-Vault-Token", p.token)
	if p.namespace != "" {
		request.Header.Set("X-Vault-Namespace", p.namespace)
	}

	response, err := p.httpClient.Do(request)
	if err != nil {
		return Bundle{}, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return Bundle{}, err
	}
	if response.StatusCode == http.StatusNotFound {
		return Bundle{}, fmt.Errorf("%w: %s", ErrNotFound, secretPath)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Bundle{}, fmt.Errorf("vault read %s: status %d: %s", secretPath, response.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload vaultKV2Response
	if err := json.Unmarshal(body, &payload); err != nil {
		return Bundle{}, err
	}
	return Bundle{
		Path:    secretPath,
		Version: payload.Data.Metadata.Version,
		Fields:  stringMap(payload.Data.Data),
	}, nil
}

func (p *VaultProvider) secretURL(secretPath string, version int) (string, error) {
	base, err := url.Parse(p.address)
	if err != nil {
		return "", err
	}
	base.Path = path.Join(base.Path, "v1", p.mount, "data", secretPath)
	if version > 0 {
		query := base.Query()
		query.Set("version", strconv.Itoa(version))
		base.RawQuery = query.Encode()
	}
	return base.String(), nil
}

type vaultKV2Response struct {
	Data struct {
		Data     map[string]any `json:"data"`
		Metadata struct {
			Version int `json:"version"`
		} `json:"metadata"`
	} `json:"data"`
}

func stringMap(values map[string]any) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		if value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			result[key] = typed
		default:
			result[key] = fmt.Sprint(typed)
		}
	}
	return result
}
