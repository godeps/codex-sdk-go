package codex

import "sync"

// Codex is the main client for interacting with the Codex CLI.
// Use StartThread to begin a new thread or ResumeThread to continue an existing one.
type Codex struct {
	client  *sdkClient
	options CodexOptions

	managedOnce sync.Once
	managed     *Client
}

// NewCodex constructs a Codex client with optional settings.
func NewCodex(options CodexOptions) *Codex {
	return &Codex{
		client:  newSDKClient(options),
		options: options,
	}
}

// StartThread begins a new conversation with the agent.
func (c *Codex) StartThread(options ThreadOptions) *Thread {
	return newManagedThread(c, options, "")
}

// ResumeThread continues a conversation using an existing thread ID.
func (c *Codex) ResumeThread(id string, options ThreadOptions) *Thread {
	return newManagedThread(c, options, id)
}

// Close releases the shared long-lived app-server process if it has been started.
func (c *Codex) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.close()
}

func (c *Codex) sharedClient() *Client {
	if c == nil || c.client == nil {
		return nil
	}
	c.managedOnce.Do(func() {
		c.managed = &Client{
			transport:   c.client,
			threadLocks: make(map[string]*threadStartLock),
			goals:       make(map[string]*goalOperation),
		}
	})
	return c.managed
}
