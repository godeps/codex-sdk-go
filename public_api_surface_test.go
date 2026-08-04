package codex

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestPublicAPISurfaceUsesContextClient(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("go", "doc", "-all", ".")
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go doc -all .: %v\n%s", err, out)
	}
	doc := string(out)
	for _, required := range []string{
		"func NewClient(ctx context.Context, opts ...Option) (*Client, error)",
		"func (t *Thread) RunContext(ctx context.Context, input Input, turnOptions TurnOptions) (*TurnResult, error)",
		"func (h *ChatGPTLoginHandle) WaitContext(ctx context.Context) (*LoginResult, error)",
		"func (h *GoalHandle) RunContext(ctx context.Context) (*TurnResult, error)",
	} {
		if !strings.Contains(doc, required) {
			t.Errorf("public API is missing %q", required)
		}
	}
}

func TestLegacyFacadeIsAbsentFromPublicAPI(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("go", "doc", "-all", ".")
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go doc -all .: %v\n%s", err, out)
	}
	doc := string(out)
	for _, removed := range []string{
		"type Codex struct",
		"func NewCodex(",
		"type CodexExec struct",
		"type CodexExecArgs struct",
		"type ExecStream struct",
		"func NewCodexExec(",
		"type Turn struct",
		"type StreamedTurn struct",
		"type ModelListResponse struct",
		"type AccountResponse struct",
		"type AccountLoginCompleted struct",
		"type ThreadReadResponse struct",
		"func NewThread(",
		"func (t *Thread) Run(",
		"func (t *Thread) RunStreamed(",
		"func (t *Thread) Read(",
		"func (t *Thread) SetName(",
		"func (t *Thread) Compact(",
		"func (h *TurnHandle) Run()",
		"func (h *ChatGPTLoginHandle) Wait()",
		"func (h *ChatGPTLoginHandle) Cancel()",
		"func (h *DeviceCodeLoginHandle) Wait()",
		"func (h *DeviceCodeLoginHandle) Cancel()",
	} {
		if strings.Contains(doc, removed) {
			t.Errorf("legacy facade export still present: %q", removed)
		}
	}
}
