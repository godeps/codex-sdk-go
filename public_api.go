package codex

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

// ChatGPTLoginHandle represents one live browser-based login attempt.
type ChatGPTLoginHandle struct {
	client  *sdkClient
	LoginID string `json:"loginId"`
	AuthURL string `json:"authUrl"`
}

// DeviceCodeLoginHandle represents one live device-code login attempt.
type DeviceCodeLoginHandle struct {
	client          *sdkClient
	LoginID         string `json:"loginId"`
	VerificationURL string `json:"verificationUrl"`
	UserCode        string `json:"userCode"`
}
