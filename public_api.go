package codex

import (
	"context"
	"encoding/json"
)

// Metadata describes the initialized app-server session.
type Metadata struct {
	ProtocolVersion string      `json:"protocolVersion"`
	UserAgent       string      `json:"userAgent"`
	ServerInfo      *ServerInfo `json:"serverInfo,omitempty"`
}

// ServerInfo describes the connected app-server runtime.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ModelListResponse is the public model-list payload.
type ModelListResponse struct {
	Data []map[string]any `json:"data"`
}

// AccountResponse is the public account/read payload.
type AccountResponse struct {
	Account            json.RawMessage `json:"account,omitempty"`
	RequiresOpenAIAuth bool            `json:"requiresOpenaiAuth"`
}

// AccountLoginCompleted is the routed login completion notification payload.
type AccountLoginCompleted struct {
	LoginID string          `json:"loginId"`
	Account json.RawMessage `json:"account,omitempty"`
}

// ChatGPTLoginHandle represents one live browser-based login attempt.
type ChatGPTLoginHandle struct {
	client  *sdkClient
	LoginID string `json:"loginId"`
	AuthURL string `json:"authUrl"`
}

// Wait blocks until the login attempt completes.
func (h *ChatGPTLoginHandle) Wait() (*AccountLoginCompleted, error) {
	return waitCompatLogin(context.Background(), h.client, h.LoginID)
}

// Cancel cancels the login attempt.
func (h *ChatGPTLoginHandle) Cancel() error {
	return h.client.request(context.Background(), "account/login/cancel", map[string]any{
		"loginId": h.LoginID,
	}, nil)
}

// DeviceCodeLoginHandle represents one live device-code login attempt.
type DeviceCodeLoginHandle struct {
	client          *sdkClient
	LoginID         string `json:"loginId"`
	VerificationURL string `json:"verificationUrl"`
	UserCode        string `json:"userCode"`
}

// Wait blocks until the device-code login completes.
func (h *DeviceCodeLoginHandle) Wait() (*AccountLoginCompleted, error) {
	return waitCompatLogin(context.Background(), h.client, h.LoginID)
}

// Cancel cancels the device-code login attempt.
func (h *DeviceCodeLoginHandle) Cancel() error {
	return h.client.request(context.Background(), "account/login/cancel", map[string]any{
		"loginId": h.LoginID,
	}, nil)
}

// ThreadReadResponse is the public thread/read payload subset.
type ThreadReadResponse struct {
	Thread struct {
		ID    string          `json:"id"`
		Name  string          `json:"name,omitempty"`
		Turns json.RawMessage `json:"turns,omitempty"`
	} `json:"thread"`
}

// Metadata returns the initialized app-server metadata when available.
func (c *Codex) Metadata() *Metadata {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.metadataSnapshot()
}

// Models returns the current app-server model list.
func (c *Codex) Models(includeHidden bool) (*ModelListResponse, error) {
	var response ModelListResponse
	if err := c.client.request(context.Background(), "model/list", map[string]any{
		"includeHidden": includeHidden,
	}, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// LoginAPIKey authenticates the local app-server session with an API key.
func (c *Codex) LoginAPIKey(apiKey string) error {
	return c.client.request(context.Background(), "account/login/start", map[string]any{
		"type":   "apiKey",
		"apiKey": apiKey,
	}, nil)
}

// StartChatGPTLogin starts a browser-based ChatGPT login flow.
func (c *Codex) StartChatGPTLogin() (*ChatGPTLoginHandle, error) {
	var response struct {
		Type    string `json:"type"`
		LoginID string `json:"loginId"`
		AuthURL string `json:"authUrl"`
	}
	if err := c.client.request(context.Background(), "account/login/start", map[string]any{
		"type": "chatgpt",
	}, &response); err != nil {
		return nil, err
	}
	if err := c.client.registerLogin(response.LoginID); err != nil {
		return nil, err
	}
	return &ChatGPTLoginHandle{
		client:  c.client,
		LoginID: response.LoginID,
		AuthURL: response.AuthURL,
	}, nil
}

// StartChatGPTDeviceCodeLogin starts a device-code ChatGPT login flow.
func (c *Codex) StartChatGPTDeviceCodeLogin() (*DeviceCodeLoginHandle, error) {
	var response struct {
		Type            string `json:"type"`
		LoginID         string `json:"loginId"`
		VerificationURL string `json:"verificationUrl"`
		UserCode        string `json:"userCode"`
	}
	if err := c.client.request(context.Background(), "account/login/start", map[string]any{
		"type": "chatgptDeviceCode",
	}, &response); err != nil {
		return nil, err
	}
	if err := c.client.registerLogin(response.LoginID); err != nil {
		return nil, err
	}
	return &DeviceCodeLoginHandle{
		client:          c.client,
		LoginID:         response.LoginID,
		VerificationURL: response.VerificationURL,
		UserCode:        response.UserCode,
	}, nil
}

// Account reads the current account state.
func (c *Codex) Account(refreshToken bool) (*AccountResponse, error) {
	var response AccountResponse
	if err := c.client.request(context.Background(), "account/read", map[string]any{
		"refreshToken": refreshToken,
	}, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Logout clears the current account session.
func (c *Codex) Logout() error {
	return c.client.request(context.Background(), "account/logout", nil, nil)
}

// Read loads the thread's persisted state from the app-server.
func (t *Thread) Read(includeTurns bool) (*ThreadReadResponse, error) {
	if t.exec != nil {
		return nil, ErrTransportClosed
	}
	if err := t.ensurePrepared(context.Background()); err != nil {
		return nil, err
	}
	threadID := t.ID()
	var response ThreadReadResponse
	if err := t.codex.client.request(context.Background(), "thread/read", map[string]any{
		"threadId":     threadID,
		"includeTurns": includeTurns,
	}, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// SetName updates the persisted thread name.
func (t *Thread) SetName(name string) error {
	if t.exec != nil {
		return ErrTransportClosed
	}
	if err := t.ensurePrepared(context.Background()); err != nil {
		return err
	}
	threadID := t.ID()
	return t.codex.client.request(context.Background(), "thread/name/set", map[string]any{
		"threadId": threadID,
		"name":     name,
	}, nil)
}

// Compact requests app-server compaction for the current thread.
func (t *Thread) Compact() error {
	if t.exec != nil {
		return ErrTransportClosed
	}
	if err := t.ensurePrepared(context.Background()); err != nil {
		return err
	}
	threadID := t.ID()
	return t.codex.client.request(context.Background(), "thread/compact/start", map[string]any{
		"threadId": threadID,
	}, nil)
}
