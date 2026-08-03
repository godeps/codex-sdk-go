package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompatibilityOutputSchemaRejectsNonObjects(t *testing.T) {
	t.Parallel()

	for _, schema := range []any{"bad", 1, true, []string{"x"}} {
		_, err := createOutputSchemaFile(schema)
		if err == nil {
			t.Fatalf("expected output schema error for %T", schema)
		}
	}
}

func TestCompatibilityOutputSchemaWritesStableJSONAndCleansUp(t *testing.T) {
	t.Parallel()

	schema, err := createOutputSchemaFile(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"answer": map[string]any{"type": "string"},
		},
	})
	if err != nil {
		t.Fatalf("createOutputSchemaFile: %v", err)
	}

	data, err := os.ReadFile(schema.SchemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	want := readCompatFile(t, filepath.Join("transcripts", "structured-output-schema.json"))
	if strings.TrimSpace(string(data)) != strings.TrimSpace(want) {
		t.Fatalf("unexpected schema JSON:\n got: %s\nwant: %s", data, want)
	}

	dir := filepath.Dir(schema.SchemaPath)
	if err := schema.Cleanup(); err != nil {
		t.Fatalf("cleanup schema: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("expected schema dir removed, stat err=%v", err)
	}
}
