package codex

import (
	"context"
	"errors"
)

// LoginAPIKey authenticates the local app-server session with an API key.
func (c *Client) LoginAPIKey(ctx context.Context, apiKey string) error {
	if ctx == nil {
		return errors.New("codex: nil context")
	}
	return c.loginStart(ctx, map[string]any{
		"type":   "apiKey",
		"apiKey": apiKey,
	}, nil)
}

// LoginChatGPT starts a browser-based ChatGPT login flow.
func (c *Client) LoginChatGPT(ctx context.Context) (*ChatGPTLoginHandle, error) {
	if ctx == nil {
		return nil, errors.New("codex: nil context")
	}
	var response struct {
		LoginID string `json:"loginId"`
		AuthURL string `json:"authUrl"`
	}
	if err := c.loginStart(ctx, map[string]any{"type": "chatgpt"}, &response); err != nil {
		return nil, err
	}
	return &ChatGPTLoginHandle{
		client:  c.transport,
		LoginID: response.LoginID,
		AuthURL: response.AuthURL,
	}, nil
}

// LoginDeviceCode starts a device-code ChatGPT login flow.
func (c *Client) LoginDeviceCode(ctx context.Context) (*DeviceCodeLoginHandle, error) {
	if ctx == nil {
		return nil, errors.New("codex: nil context")
	}
	var response struct {
		LoginID         string `json:"loginId"`
		VerificationURL string `json:"verificationUrl"`
		UserCode        string `json:"userCode"`
	}
	if err := c.loginStart(ctx, map[string]any{"type": "chatgptDeviceCode"}, &response); err != nil {
		return nil, err
	}
	return &DeviceCodeLoginHandle{
		client:          c.transport,
		LoginID:         response.LoginID,
		VerificationURL: response.VerificationURL,
		UserCode:        response.UserCode,
	}, nil
}

// Account reads the current account state.
func (c *Client) Account(ctx context.Context, refreshToken bool) (*AccountState, error) {
	if ctx == nil {
		return nil, errors.New("codex: nil context")
	}
	var response struct {
		Account            map[string]any `json:"account"`
		RequiresOpenAIAuth bool           `json:"requiresOpenaiAuth"`
	}
	if err := c.transport.request(ctx, "account/read", map[string]any{
		"refreshToken": refreshToken,
	}, &response); err != nil {
		return nil, err
	}
	return &AccountState{
		Account:            decodeAccount(response.Account),
		RequiresOpenAIAuth: response.RequiresOpenAIAuth,
	}, nil
}

// Logout clears the current account session.
func (c *Client) Logout(ctx context.Context) error {
	if ctx == nil {
		return errors.New("codex: nil context")
	}
	return c.transport.request(ctx, "account/logout", nil, nil)
}

// WaitContext blocks until the browser login attempt completes.
func (h *ChatGPTLoginHandle) WaitContext(ctx context.Context) (*LoginResult, error) {
	if h == nil || h.client == nil {
		return nil, ErrTransportClosed
	}
	if ctx == nil {
		return nil, errors.New("codex: nil context")
	}
	return (&Client{transport: h.client}).waitForLogin(ctx, h.LoginID)
}

// CancelContext cancels the browser login attempt.
func (h *ChatGPTLoginHandle) CancelContext(ctx context.Context) error {
	if h == nil || h.client == nil {
		return ErrTransportClosed
	}
	if ctx == nil {
		return errors.New("codex: nil context")
	}
	return h.client.request(ctx, "account/login/cancel", map[string]any{
		"loginId": h.LoginID,
	}, nil)
}

// WaitContext blocks until the device-code login attempt completes.
func (h *DeviceCodeLoginHandle) WaitContext(ctx context.Context) (*LoginResult, error) {
	if h == nil || h.client == nil {
		return nil, ErrTransportClosed
	}
	if ctx == nil {
		return nil, errors.New("codex: nil context")
	}
	return (&Client{transport: h.client}).waitForLogin(ctx, h.LoginID)
}

// CancelContext cancels the device-code login attempt.
func (h *DeviceCodeLoginHandle) CancelContext(ctx context.Context) error {
	if h == nil || h.client == nil {
		return ErrTransportClosed
	}
	if ctx == nil {
		return errors.New("codex: nil context")
	}
	return h.client.request(ctx, "account/login/cancel", map[string]any{
		"loginId": h.LoginID,
	}, nil)
}
