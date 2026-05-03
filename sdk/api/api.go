// Package api mirrors hyperliquid-python-sdk/hyperliquid/api.py.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"hyperliquid-bot/sdk/constants"
)

// API corresponds to Python:
// hyperliquid.api.API
type API struct {
	BaseURL    string
	HTTPClient *http.Client
}

// New corresponds to Python:
// hyperliquid.api.API.__init__
func New(baseURL string, timeout time.Duration) *API {
	if baseURL == "" {
		baseURL = constants.MainnetAPIURL
	}
	client := http.DefaultClient
	if timeout > 0 {
		client = &http.Client{Timeout: timeout}
	}
	return &API{BaseURL: baseURL, HTTPClient: client}
}

// Post corresponds to Python:
// hyperliquid.api.API.post
func (a *API) Post(ctx context.Context, urlPath string, payload any, out any) error {
	if payload == nil {
		payload = map[string]any{}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.BaseURL+urlPath, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := handleException(resp.StatusCode, respBody, resp.Header); err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("could not parse JSON: %s", string(respBody))
	}
	return nil
}

func handleException(statusCode int, body []byte, header http.Header) error {
	if statusCode < 400 {
		return nil
	}
	if statusCode < 500 {
		var errResp struct {
			Code any    `json:"code"`
			Msg  string `json:"msg"`
			Data any    `json:"data"`
		}
		if err := json.Unmarshal(body, &errResp); err != nil {
			return ClientError{StatusCode: statusCode, Message: string(body), Header: header}
		}
		return ClientError{
			StatusCode: statusCode,
			ErrorCode:  errResp.Code,
			Message:    errResp.Msg,
			Header:     header,
			ErrorData:  errResp.Data,
		}
	}
	return ServerError{StatusCode: statusCode, Message: string(body)}
}

// ClientError corresponds to Python:
// hyperliquid.utils.error.ClientError
type ClientError struct {
	StatusCode int
	ErrorCode  any
	Message    string
	Header     http.Header
	ErrorData  any
}

func (e ClientError) Error() string {
	return fmt.Sprintf("(%d, %v, %s, %v)", e.StatusCode, e.ErrorCode, e.Message, e.ErrorData)
}

// ServerError corresponds to Python:
// hyperliquid.utils.error.ServerError
type ServerError struct {
	StatusCode int
	Message    string
}

func (e ServerError) Error() string {
	return fmt.Sprintf("(%d, %s)", e.StatusCode, e.Message)
}
