package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestExtractEnum(t *testing.T) {
	got := extractEnum(map[string]any{
		"enum": []any{"thread/start"},
	})
	if len(got) != 1 || got[0] != "thread/start" {
		t.Fatalf("extractEnum() = %#v", got)
	}
}

func TestWalkSchemaCollectsMethodTitles(t *testing.T) {
	schema := map[string]any{
		"definitions": map[string]any{
			"ThreadStart": map[string]any{
				"title": "Thread/startRequestMethod",
				"enum":  []any{"thread/start"},
			},
			"TurnStarted": map[string]any{
				"title": "Turn/startedNotificationMethod",
				"enum":  []any{"turn/started"},
			},
		},
	}
	var titles []string
	walkSchema(schema, func(title string, enum []string) {
		titles = append(titles, title+"="+enum[0])
	})
	if len(titles) != 2 {
		t.Fatalf("walkSchema() titles = %#v", titles)
	}
}

func TestLoadSchemaBundleAndManifestCounts(t *testing.T) {
	bundle, err := loadSchemaBundle(filepath.Join(repoRoot(t), "schema"))
	if err != nil {
		t.Fatalf("loadSchemaBundle() error = %v", err)
	}
	if got := len(bundle.Definitions); got != 557 {
		t.Fatalf("len(bundle.Definitions) = %d, want 557", got)
	}
	lock, err := readLock(filepath.Join(repoRoot(t), "reference.lock.json"))
	if err != nil {
		t.Fatalf("readLock() error = %v", err)
	}
	manifestValue, err := buildManifestFromBundle(bundle, lock)
	if err != nil {
		t.Fatalf("buildManifestFromBundle() error = %v", err)
	}
	if manifestValue.Counts.AggregateDefinitions != 557 || manifestValue.Counts.Requests != 95 || manifestValue.Counts.Notifications != 70 || manifestValue.Counts.ServerRequests != 10 {
		t.Fatalf("unexpected manifest counts: %#v", manifestValue.Counts)
	}
	if len(manifestValue.MissingDefinitions) != 0 || len(manifestValue.MissingMethods) != 0 || len(manifestValue.UnsupportedConstructs) != 0 || len(manifestValue.DuplicateSymbols) != 0 {
		t.Fatalf("unexpected manifest empties: %#v", manifestValue)
	}
}

func TestGenerateProtocolDeterministic(t *testing.T) {
	lock, err := readLock(filepath.Join(repoRoot(t), "reference.lock.json"))
	if err != nil {
		t.Fatalf("readLock() error = %v", err)
	}
	first, err := generateProtocol(filepath.Join(repoRoot(t), "schema"), lock)
	if err != nil {
		t.Fatalf("generateProtocol(first) error = %v", err)
	}
	second, err := generateProtocol(filepath.Join(repoRoot(t), "schema"), lock)
	if err != nil {
		t.Fatalf("generateProtocol(second) error = %v", err)
	}
	if !bytes.Equal(mustJSON(t, first.Manifest), mustJSON(t, second.Manifest)) {
		t.Fatal("manifest generation is not deterministic")
	}
	for path, data := range first.Files {
		other, ok := second.Files[path]
		if !ok || !bytes.Equal(data, other) {
			t.Fatalf("generated file %s is not deterministic", path)
		}
	}
}

func TestDeriveResponseTypeOverrides(t *testing.T) {
	bundle, err := loadSchemaBundle(filepath.Join(repoRoot(t), "schema"))
	if err != nil {
		t.Fatalf("loadSchemaBundle() error = %v", err)
	}
	cases := []registryEntry{
		{Method: "account/read", WrapperType: "AccountReadRequestEnvelope", PayloadType: "GetAccountParams"},
		{Method: "account/rateLimits/read", WrapperType: "AccountRateLimitsReadRequestEnvelope"},
		{Method: "account/usage/read", WrapperType: "AccountUsageReadRequestEnvelope"},
		{Method: "account/workspaceMessages/read", WrapperType: "AccountWorkspaceMessagesReadRequestEnvelope"},
		{Method: "config/batchWrite", WrapperType: "ConfigBatchWriteRequestEnvelope", PayloadType: "ConfigBatchWriteParams"},
		{Method: "config/value/write", WrapperType: "ConfigValueWriteRequestEnvelope", PayloadType: "ConfigValueWriteParams"},
	}
	want := []string{
		"GetAccountResponse",
		"GetAccountRateLimitsResponse",
		"GetAccountTokenUsageResponse",
		"GetWorkspaceMessagesResponse",
		"ConfigWriteResponse",
		"ConfigWriteResponse",
	}
	for i, tc := range cases {
		if got := deriveResponseType(bundle, tc, false); got != want[i] {
			t.Fatalf("deriveResponseType(%s) = %q, want %q", tc.Method, got, want[i])
		}
	}
}

func TestVerifyMatchesCheckedInArtifacts(t *testing.T) {
	root := repoRoot(t)
	lock, err := readLock(filepath.Join(root, "reference.lock.json"))
	if err != nil {
		t.Fatalf("readLock() error = %v", err)
	}
	if err := verify(root, lock); err != nil {
		t.Fatalf("verify() error = %v", err)
	}
}

func TestGeneratedTypesCoverDefinitionSets(t *testing.T) {
	bundle, err := loadSchemaBundle(filepath.Join(repoRoot(t), "schema"))
	if err != nil {
		t.Fatalf("loadSchemaBundle() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "protocol", "generated_types.go"))
	if err != nil {
		t.Fatalf("ReadFile(generated_types.go) error = %v", err)
	}
	re := regexp.MustCompile(`(?m)^type (\w+) `)
	decls := map[string]bool{}
	for _, match := range re.FindAllStringSubmatch(string(data), -1) {
		decls[match[1]] = true
	}
	for name := range bundle.Definitions {
		if !decls[goTypeName(name)] {
			t.Fatalf("missing aggregate type %s", name)
		}
	}
	for name := range bundle.Supplemental {
		if !decls[goTypeName(name)] {
			t.Fatalf("missing supplemental type %s", name)
		}
	}
}

func TestClientRequestRegistryEmptyResponsesAreExplicitlyAllowed(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "protocol", "generated_registries.go"))
	if err != nil {
		t.Fatalf("ReadFile(generated_registries.go) error = %v", err)
	}
	lines := strings.Split(string(data), "\n")
	inside := false
	var empties []string
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "var ClientRequestRegistry"):
			inside = true
		case strings.HasPrefix(line, "var ServerNotificationRegistry"):
			inside = false
		case inside && strings.Contains(line, `ResponseType: ""`):
			empties = append(empties, strings.TrimSpace(line))
		}
	}
	if len(empties) != 1 || !strings.Contains(empties[0], `"config/mcpServer/reload"`) {
		t.Fatalf("unexpected empty response entries: %#v", empties)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(root, "reference.lock.json")); err == nil {
			return root
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("unable to locate repo root from test working directory")
		}
		root = parent
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return data
}
