package codex

import (
	"context"
	"testing"
)

func TestClientThreadOptionsWithNilConfig(t *testing.T) {
	server, captures := writeContextAPIServer(t)
	client, err := NewClient(context.Background(), WithCodexPath(server))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	noNetwork := false
	_, err = client.StartThread(context.Background(), ThreadOptions{
		ModelReasoningEffort: ReasoningHigh, SkipGitRepoCheck: true,
		NetworkAccessEnabled: &noNetwork, WebSearchMode: WebSearchDisabled,
		AdditionalDirectories: []string{"/tmp/extra"},
	})
	if err != nil {
		t.Fatal(err)
	}
	payload := captures.readJSON(t, "thread_start_1.json")
	config, ok := payload["config"].(map[string]any)
	if !ok {
		t.Fatalf("missing thread config: %#v", payload)
	}
	if config["web_search"] != "disabled" || config["model_reasoning_effort"] != "high" || config["skip_git_repo_check"] != true {
		t.Fatalf("options lost when Config is nil: %#v", config)
	}
	sandbox, ok := config["sandbox_workspace_write"].(map[string]any)
	if !ok || sandbox["network_access"] != false {
		t.Fatalf("network policy not passed: %#v", config)
	}
}
