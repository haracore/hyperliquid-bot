package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	execution "hyperliquid-bot/execution/client"
	"hyperliquid-bot/execution/credentials"
)

const (
	mainnetAPIURL = "https://api.hyperliquid.xyz"
	testnetAPIURL = "https://api.hyperliquid-testnet.xyz"
)

type Config struct {
	Credentials          credentials.ProviderConfig
	AddressOverride      string
	PrivateKeyOverride   string
	VaultAddressOverride string
	BaseURL              string
	Testnet              bool
	Timeout              time.Duration
}

type App struct {
	config Config
	tmpl   *template.Template
}

func New(config Config) *App {
	if config.Timeout == 0 {
		config.Timeout = 20 * time.Second
	}
	if strings.TrimSpace(config.Credentials.Name) == "" {
		config.Credentials.Name = credentials.ProviderEnv
	}
	if strings.TrimSpace(config.Credentials.Account) == "" {
		config.Credentials.Account = "main"
	}
	if strings.TrimSpace(config.Credentials.Prefix) == "" {
		config.Credentials.Prefix = "accounts"
	}
	config.BaseURL = resolveBaseURL(config.BaseURL, config.Testnet)
	return &App{
		config: config,
		tmpl:   template.Must(template.New("ui").Parse(pageTemplate)),
	}
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.home)
	mux.HandleFunc("/balances", a.balances)
	mux.HandleFunc("/positions", a.positions)
	mux.HandleFunc("/perp/orders", a.perpOrders)
	mux.HandleFunc("/spot/orders", a.spotOrders)
	mux.HandleFunc("/static/app.css", a.styles)
	return mux
}

type pageData struct {
	Title          string
	Active         string
	DefaultAddress string
	BaseURL        string
	Testnet        bool
	SecretProvider string
	Account        string
	Error          string
	ResultJSON     string
	Balances       *balancesView
	Positions      []execution.Position
	OrderContext   orderPageContext
}

type balancesView struct {
	PerpSummary   []kv
	SpotBalances  []spotBalance
	PerpPositions []execution.Position
}

type kv struct {
	Key   string
	Value string
}

type spotBalance struct {
	Coin  string
	Total string
	Hold  string
}

type orderPageContext struct {
	Kind string
	Dex  string
}

func (a *App) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/balances", http.StatusFound)
}

func (a *App) balances(w http.ResponseWriter, r *http.Request) {
	data := a.basePage("Balances", "balances")
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			data.Error = err.Error()
			a.render(w, data)
			return
		}
		address := strings.TrimSpace(r.FormValue("address"))
		if address == "" {
			account, err := a.resolveAccountFields(r.Context(), "", "", "")
			if err != nil {
				data.Error = err.Error()
				a.render(w, data)
				return
			}
			address = account.Address
		}
		if address == "" {
			data.Error = "address is required"
			a.render(w, data)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), a.config.Timeout)
		defer cancel()
		client := a.newReadClient("")
		result, err := client.Balances(ctx, address)
		if err != nil {
			data.Error = err.Error()
			a.render(w, data)
			return
		}
		data.DefaultAddress = address
		data.Balances = buildBalancesView(result)
		data.ResultJSON = prettyJSON(result)
	}
	a.render(w, data)
}

func (a *App) positions(w http.ResponseWriter, r *http.Request) {
	data := a.basePage("Perp Positions", "positions")
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			data.Error = err.Error()
			a.render(w, data)
			return
		}
		address := strings.TrimSpace(r.FormValue("address"))
		dex := strings.TrimSpace(r.FormValue("dex"))
		showAll := r.FormValue("all") == "on"
		if address == "" {
			account, err := a.resolveAccountFields(r.Context(), "", "", "")
			if err != nil {
				data.Error = err.Error()
				a.render(w, data)
				return
			}
			address = account.Address
		}
		if address == "" {
			data.Error = "address is required"
			a.render(w, data)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), a.config.Timeout)
		defer cancel()
		client := a.newReadClient(dex)
		positions, err := client.PerpPositions(ctx, address, showAll)
		if err != nil {
			data.Error = err.Error()
			a.render(w, data)
			return
		}
		data.DefaultAddress = address
		data.OrderContext.Dex = dex
		data.Positions = positions
		data.ResultJSON = prettyJSON(positions)
	}
	a.render(w, data)
}

func (a *App) perpOrders(w http.ResponseWriter, r *http.Request) {
	a.orders(w, r, "perp")
}

func (a *App) spotOrders(w http.ResponseWriter, r *http.Request) {
	a.orders(w, r, "spot")
}

func (a *App) orders(w http.ResponseWriter, r *http.Request, kind string) {
	title := "Spot Orders"
	if kind == "perp" {
		title = "Perp Orders"
	}
	data := a.basePage(title, kind+"-orders")
	data.OrderContext.Kind = kind
	if r.Method == http.MethodPost {
		result, err := a.handleOrderAction(r, kind)
		if err != nil {
			data.Error = err.Error()
		} else {
			data.ResultJSON = prettyJSON(result)
		}
		data.DefaultAddress = strings.TrimSpace(r.FormValue("address"))
		data.OrderContext.Dex = strings.TrimSpace(r.FormValue("dex"))
	}
	a.render(w, data)
}

func (a *App) handleOrderAction(r *http.Request, kind string) (any, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	action := r.FormValue("action")
	dex := strings.TrimSpace(r.FormValue("dex"))
	ctx, cancel := context.WithTimeout(r.Context(), a.config.Timeout)
	defer cancel()

	switch action {
	case "open-orders":
		address := strings.TrimSpace(r.FormValue("address"))
		if address == "" {
			account, err := a.resolveAccountFields(ctx, "", "", "")
			if err != nil {
				return nil, err
			}
			address = account.Address
		}
		if address == "" {
			return nil, fmt.Errorf("address is required")
		}
		client := a.newReadClient(dex)
		if kind == "perp" {
			return client.PerpOpenOrders(ctx, address)
		}
		return client.SpotOpenOrders(ctx, address, r.FormValue("frontend") == "on")
	case "place":
		if err := a.requireWrite(r); err != nil {
			return nil, err
		}
		account, err := a.resolveAccount(ctx, "", "", "")
		if err != nil {
			return nil, err
		}
		client := a.newWriteClient(dex, account)
		request, err := parseOrderRequest(r, kind == "perp")
		if err != nil {
			return nil, err
		}
		if kind == "perp" {
			return client.PlacePerpOrder(ctx, request)
		}
		return client.PlaceSpotOrder(ctx, request)
	case "cancel-oid":
		if err := a.requireWrite(r); err != nil {
			return nil, err
		}
		account, err := a.resolveAccount(ctx, "", "", "")
		if err != nil {
			return nil, err
		}
		client := a.newWriteClient(dex, account)
		request, err := parseCancelOrderRequest(r)
		if err != nil {
			return nil, err
		}
		if kind == "perp" {
			return client.CancelPerpOrder(ctx, request)
		}
		return client.CancelSpotOrder(ctx, request)
	case "cancel-cloid":
		if err := a.requireWrite(r); err != nil {
			return nil, err
		}
		account, err := a.resolveAccount(ctx, "", "", "")
		if err != nil {
			return nil, err
		}
		client := a.newWriteClient(dex, account)
		request, err := parseCancelByCloidRequest(r)
		if err != nil {
			return nil, err
		}
		if kind == "perp" {
			return client.CancelPerpByCloid(ctx, request)
		}
		return client.CancelSpotByCloid(ctx, request)
	case "modify":
		if err := a.requireWrite(r); err != nil {
			return nil, err
		}
		account, err := a.resolveAccount(ctx, "", "", "")
		if err != nil {
			return nil, err
		}
		client := a.newWriteClient(dex, account)
		request, err := parseModifyOrderRequest(r, kind == "perp")
		if err != nil {
			return nil, err
		}
		if kind == "perp" {
			return client.ModifyPerpOrder(ctx, request)
		}
		return client.ModifySpotOrder(ctx, request)
	default:
		return nil, fmt.Errorf("unknown action %q", action)
	}
}

func (a *App) requireWrite(r *http.Request) error {
	if r.FormValue("confirm") != "on" {
		return fmt.Errorf("confirmation checkbox is required")
	}
	return nil
}

func (a *App) newReadClient(dex string) *execution.Client {
	return execution.New(execution.Config{
		BaseURL: a.config.BaseURL,
		Timeout: a.config.Timeout,
		Dex:     dex,
	})
}

func (a *App) newWriteClient(dex string, account credentials.Account) *execution.Client {
	var vault *string
	if strings.TrimSpace(account.VaultAddress) != "" {
		vault = &account.VaultAddress
	}
	return execution.New(execution.Config{
		BaseURL:      a.config.BaseURL,
		Timeout:      a.config.Timeout,
		PrivateKey:   account.PrivateKey,
		Dex:          dex,
		VaultAddress: vault,
	})
}

func (a *App) resolveAccount(ctx context.Context, privateKeyOverride string, addressOverride string, vaultOverride string) (credentials.Account, error) {
	resolveCtx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	account, err := credentials.ResolveAccount(resolveCtx, a.config.Credentials)
	if err != nil {
		return credentials.Account{}, fmt.Errorf("resolve execution secrets: %w", err)
	}
	account = credentials.ApplyOverrides(account, a.config.PrivateKeyOverride, a.config.AddressOverride, a.config.VaultAddressOverride)
	account = credentials.ApplyOverrides(account, privateKeyOverride, addressOverride, vaultOverride)
	if strings.TrimSpace(account.PrivateKey) == "" {
		return credentials.Account{}, fmt.Errorf("private key is not configured")
	}
	return account, nil
}

func (a *App) resolveAccountFields(ctx context.Context, privateKeyOverride string, addressOverride string, vaultOverride string) (credentials.Account, error) {
	resolveCtx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	account, err := credentials.ResolveAccountFields(resolveCtx, a.config.Credentials)
	if err != nil {
		return credentials.Account{}, fmt.Errorf("resolve execution secrets: %w", err)
	}
	account = credentials.ApplyOverrides(account, a.config.PrivateKeyOverride, a.config.AddressOverride, a.config.VaultAddressOverride)
	account = credentials.ApplyOverrides(account, privateKeyOverride, addressOverride, vaultOverride)
	return account, nil
}

func (a *App) basePage(title string, active string) pageData {
	return pageData{
		Title:          title,
		Active:         active,
		DefaultAddress: a.config.AddressOverride,
		BaseURL:        a.config.BaseURL,
		Testnet:        a.config.Testnet,
		SecretProvider: a.config.Credentials.Name,
		Account:        a.config.Credentials.Account,
	}
}

func (a *App) render(w http.ResponseWriter, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *App) styles(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write([]byte(stylesheet))
}

func parseOrderRequest(r *http.Request, allowReduceOnly bool) (execution.OrderRequest, error) {
	coin := strings.TrimSpace(r.FormValue("coin"))
	if coin == "" {
		return execution.OrderRequest{}, fmt.Errorf("coin is required")
	}
	isBuy, err := parseSide(r.FormValue("side"))
	if err != nil {
		return execution.OrderRequest{}, err
	}
	size, err := parsePositiveFloat(r.FormValue("size"), "size")
	if err != nil {
		return execution.OrderRequest{}, err
	}
	price, err := parsePositiveFloat(r.FormValue("price"), "price")
	if err != nil {
		return execution.OrderRequest{}, err
	}
	tif := strings.TrimSpace(r.FormValue("tif"))
	if !validTIF(tif) {
		return execution.OrderRequest{}, fmt.Errorf("tif must be one of Gtc, Ioc, Alo")
	}
	var cloid *execution.Cloid
	if raw := strings.TrimSpace(r.FormValue("cloid")); raw != "" {
		parsed, err := execution.NewCloid(raw)
		if err != nil {
			return execution.OrderRequest{}, fmt.Errorf("cloid: %w", err)
		}
		cloid = &parsed
	}
	return execution.OrderRequest{
		Coin:       coin,
		IsBuy:      isBuy,
		Size:       size,
		Price:      price,
		TIF:        tif,
		ReduceOnly: allowReduceOnly && r.FormValue("reduceOnly") == "on",
		Cloid:      cloid,
	}, nil
}

func parseCancelOrderRequest(r *http.Request) (execution.CancelOrderRequest, error) {
	coin := strings.TrimSpace(r.FormValue("coin"))
	if coin == "" {
		return execution.CancelOrderRequest{}, fmt.Errorf("coin is required")
	}
	oid, err := parsePositiveInt(r.FormValue("oid"), "oid")
	if err != nil {
		return execution.CancelOrderRequest{}, err
	}
	return execution.CancelOrderRequest{Coin: coin, Oid: oid}, nil
}

func parseCancelByCloidRequest(r *http.Request) (execution.CancelByCloidRequest, error) {
	coin := strings.TrimSpace(r.FormValue("coin"))
	if coin == "" {
		return execution.CancelByCloidRequest{}, fmt.Errorf("coin is required")
	}
	cloid, err := execution.NewCloid(strings.TrimSpace(r.FormValue("cloid")))
	if err != nil {
		return execution.CancelByCloidRequest{}, fmt.Errorf("cloid: %w", err)
	}
	return execution.CancelByCloidRequest{Coin: coin, Cloid: cloid}, nil
}

func parseModifyOrderRequest(r *http.Request, allowReduceOnly bool) (execution.ModifyOrderRequest, error) {
	coin := strings.TrimSpace(r.FormValue("coin"))
	if coin == "" {
		return execution.ModifyOrderRequest{}, fmt.Errorf("coin is required")
	}
	oid, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("oid")))
	orderID, err := execution.ParseOrderID(oid, strings.TrimSpace(r.FormValue("oidCloid")))
	if err != nil {
		return execution.ModifyOrderRequest{}, fmt.Errorf("order cloid: %w", err)
	}
	if oid == 0 && strings.TrimSpace(r.FormValue("oidCloid")) == "" {
		return execution.ModifyOrderRequest{}, fmt.Errorf("oid or oid cloid is required")
	}
	isBuy, err := parseSide(r.FormValue("side"))
	if err != nil {
		return execution.ModifyOrderRequest{}, err
	}
	size, err := parsePositiveFloat(r.FormValue("size"), "size")
	if err != nil {
		return execution.ModifyOrderRequest{}, err
	}
	price, err := parsePositiveFloat(r.FormValue("price"), "price")
	if err != nil {
		return execution.ModifyOrderRequest{}, err
	}
	tif := strings.TrimSpace(r.FormValue("tif"))
	if !validTIF(tif) {
		return execution.ModifyOrderRequest{}, fmt.Errorf("tif must be one of Gtc, Ioc, Alo")
	}
	var cloid *execution.Cloid
	if raw := strings.TrimSpace(r.FormValue("newCloid")); raw != "" {
		parsed, err := execution.NewCloid(raw)
		if err != nil {
			return execution.ModifyOrderRequest{}, fmt.Errorf("new cloid: %w", err)
		}
		cloid = &parsed
	}
	return execution.ModifyOrderRequest{
		OrderID:    orderID,
		Coin:       coin,
		IsBuy:      isBuy,
		Size:       size,
		Price:      price,
		TIF:        tif,
		ReduceOnly: allowReduceOnly && r.FormValue("reduceOnly") == "on",
		Cloid:      cloid,
	}, nil
}

func buildBalancesView(result execution.BalancesResult) *balancesView {
	view := &balancesView{
		PerpSummary:   buildPerpSummary(result.PerpState),
		SpotBalances:  buildSpotBalances(result.SpotState),
		PerpPositions: execution.ExtractPerpPositions(result.PerpState, false),
	}
	return view
}

func buildPerpSummary(state map[string]any) []kv {
	rows := []kv{}
	appendValue := func(key string, value any) {
		if value != nil {
			rows = append(rows, kv{Key: key, Value: fmt.Sprint(value)})
		}
	}
	appendValue("withdrawable", state["withdrawable"])
	if summary, ok := state["marginSummary"].(map[string]any); ok {
		for _, key := range []string{"accountValue", "totalRawUsd", "totalMarginUsed", "totalNtlPos"} {
			appendValue(key, summary[key])
		}
	}
	return rows
}

func buildSpotBalances(state map[string]any) []spotBalance {
	rawBalances, _ := state["balances"].([]any)
	rows := make([]spotBalance, 0, len(rawBalances))
	for _, raw := range rawBalances {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		rows = append(rows, spotBalance{
			Coin:  fmt.Sprint(row["coin"]),
			Total: firstPresent(row, "total", "balance"),
			Hold:  firstPresent(row, "hold", "holdBalance"),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Coin < rows[j].Coin
	})
	return rows
}

func firstPresent(row map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := row[key]; ok && value != nil {
			return fmt.Sprint(value)
		}
	}
	return ""
}

func parseSide(side string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "buy", "b":
		return true, nil
	case "sell", "s":
		return false, nil
	default:
		return false, fmt.Errorf("side must be buy or sell")
	}
}

func validTIF(tif string) bool {
	switch tif {
	case "Gtc", "Ioc", "Alo":
		return true
	default:
		return false
	}
}

func parsePositiveFloat(raw string, name string) (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be greater than 0", name)
	}
	return value, nil
}

func parsePositiveInt(raw string, name string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be greater than 0", name)
	}
	return value, nil
}

func prettyJSON(value any) string {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Sprint(value)
	}
	return strings.TrimRight(buf.String(), "\n")
}

func resolveBaseURL(baseURL string, testnet bool) string {
	if testnet {
		return testnetAPIURL
	}
	trimmed := strings.TrimSpace(baseURL)
	if trimmed != "" {
		return trimmed
	}
	return mainnetAPIURL
}
