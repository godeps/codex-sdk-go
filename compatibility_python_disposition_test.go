package codex

import (
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type pythonDisposition struct {
	Entries         []pythonDispositionEntry           `json:"entries"`
	AllowedStatuses []string                           `json:"allowed_statuses"`
	Patterns        []pythonDispositionPattern         `json:"patterns"`
	Overrides       map[string]pythonDispositionLegacy `json:"overrides"`
}

type pythonDispositionEntry struct {
	PythonTest string   `json:"python_test"`
	Category   string   `json:"category"`
	Status     string   `json:"status"`
	Rationale  string   `json:"rationale"`
	Evidence   []string `json:"evidence"`
}

type pythonDispositionPattern struct {
	Pattern  string   `json:"pattern"`
	Status   string   `json:"status"`
	Reason   string   `json:"reason"`
	Evidence []string `json:"evidence"`
}

type pythonDispositionLegacy struct {
	Status   string   `json:"status"`
	Reason   string   `json:"reason"`
	Evidence []string `json:"evidence"`
}

func TestCompatibilityPythonDispositionCoversPinnedPythonTests(t *testing.T) {
	t.Parallel()

	dispositionPath := filepath.Join(repoRoot(t), "testdata", "python-test-disposition.json")
	data, err := os.ReadFile(dispositionPath)
	if err != nil {
		t.Fatalf("read python disposition: %v", err)
	}

	var disposition pythonDisposition
	if err := json.Unmarshal(data, &disposition); err != nil {
		t.Fatalf("unmarshal python disposition: %v", err)
	}
	allowedCategories := map[string]struct{}{
		"behavior":             {},
		"generation":           {},
		"packaging":            {},
		"documentation":        {},
		"python_language_only": {},
		"legacy":               {},
	}
	allowedStatuses := map[string]struct{}{
		"equivalent":                  {},
		"release_pipeline_equivalent": {},
		"language_not_applicable":     {},
		"covered":                     {},
		"partial":                     {},
		"planned":                     {},
		"not_applicable":              {},
	}

	pythonTestsRoot := pinnedPythonTestsRoot(t)
	cmd := exec.Command("python", "-c", `
from pathlib import Path
import json, re
import sys
root = Path(sys.argv[1])
items = []
for path in sorted(root.glob('test_*.py')):
    text = path.read_text()
    for name in re.findall(r'^def (test_[A-Za-z0-9_]+)\(', text, re.M):
        items.append(f"{path.name}::{name}")
print(json.dumps(items))
`, pythonTestsRoot)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("enumerate python tests: %v\n%s", err, out)
	}

	var pythonTests []string
	if err := json.Unmarshal(out, &pythonTests); err != nil {
		t.Fatalf("decode python test list: %v\n%s", err, out)
	}

	goTests := collectGoTests(t)
	entries := map[string]pythonDispositionEntry{}
	addEntry := func(entry pythonDispositionEntry) {
		if strings.TrimSpace(entry.PythonTest) == "" {
			t.Fatal("python disposition contains empty python_test")
		}
		if _, dup := entries[entry.PythonTest]; dup {
			t.Fatalf("duplicate python disposition for %s", entry.PythonTest)
		}
		if _, ok := allowedCategories[entry.Category]; !ok {
			t.Fatalf("python disposition for %s uses unknown category %q", entry.PythonTest, entry.Category)
		}
		if _, ok := allowedStatuses[entry.Status]; !ok {
			t.Fatalf("python disposition for %s uses unknown status %q", entry.PythonTest, entry.Status)
		}
		if strings.TrimSpace(entry.Rationale) == "" {
			t.Fatalf("python disposition for %s is missing rationale", entry.PythonTest)
		}
		if len(entry.Evidence) == 0 {
			t.Fatalf("python disposition for %s is missing evidence", entry.PythonTest)
		}
		if entry.Status == "language_not_applicable" && entry.Category != "python_language_only" {
			t.Fatalf("python disposition for %s must use python_language_only with language_not_applicable", entry.PythonTest)
		}
		if entry.Category == "python_language_only" && entry.Status != "language_not_applicable" {
			t.Fatalf("python disposition for %s must use language_not_applicable for python_language_only rows", entry.PythonTest)
		}
		if entry.Status == "release_pipeline_equivalent" && !hasWorkflowEvidence(entry.Evidence) {
			t.Fatalf("python disposition for %s must cite workflow evidence for release_pipeline_equivalent", entry.PythonTest)
		}
		if entry.Status == "equivalent" && !hasGoTestEvidence(entry.Evidence) {
			t.Fatalf("python disposition for %s must cite at least one go_test evidence for equivalent rows", entry.PythonTest)
		}
		for _, evidence := range entry.Evidence {
			validateEvidence(t, evidence, goTests)
		}
		entries[entry.PythonTest] = entry
	}

	for _, entry := range disposition.Entries {
		addEntry(entry)
	}

	if len(entries) == 0 && (len(disposition.Patterns) != 0 || len(disposition.Overrides) != 0) {
		for _, testID := range pythonTests {
			if override, ok := disposition.Overrides[testID]; ok {
				addEntry(pythonDispositionEntry{
					PythonTest: testID,
					Category:   "legacy",
					Status:     override.Status,
					Rationale:  override.Reason,
					Evidence:   override.Evidence,
				})
				continue
			}
			matched := false
			for _, pattern := range disposition.Patterns {
				ok, err := filepath.Match(pattern.Pattern, testID)
				if err != nil {
					t.Fatalf("invalid disposition pattern %q: %v", pattern.Pattern, err)
				}
				if !ok {
					continue
				}
				addEntry(pythonDispositionEntry{
					PythonTest: testID,
					Category:   "legacy",
					Status:     pattern.Status,
					Rationale:  pattern.Reason,
					Evidence:   pattern.Evidence,
				})
				matched = true
				break
			}
			if !matched {
				t.Fatalf("missing python disposition for %s", testID)
			}
		}
	}

	for _, testID := range pythonTests {
		if _, ok := entries[testID]; !ok {
			t.Fatalf("missing python disposition for %s", testID)
		}
	}
	for testID := range entries {
		if !containsString(pythonTests, testID) {
			t.Fatalf("stale python disposition entry for %s", testID)
		}
	}
}

func pinnedPythonTestsRoot(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	candidates := []string{}
	if reference := os.Getenv("CODEX_REFERENCE_REPO"); reference != "" {
		candidates = append(candidates, filepath.Join(reference, "sdk", "python", "tests"))
	}
	candidates = append(candidates,
		filepath.Clean(filepath.Join(root, "../../other/codex/sdk/python/tests")),
		filepath.Join(root, "_reference", "codex", "sdk", "python", "tests"),
	)
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	t.Fatal("unable to resolve pinned Python test directory")
	return ""
}

func collectGoTests(t *testing.T) map[string]map[string]struct{} {
	t.Helper()

	re := regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]+)\(`)
	goTests := map[string]map[string]struct{}{}
	if err := filepath.WalkDir(repoRoot(t), func(filePath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(filePath, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(repoRoot(t), filePath)
		if err != nil {
			return err
		}
		rel = path.Clean(filepath.ToSlash(rel))
		for _, match := range re.FindAllStringSubmatch(string(data), -1) {
			set := goTests[rel]
			if set == nil {
				set = map[string]struct{}{}
				goTests[rel] = set
			}
			set[match[1]] = struct{}{}
		}
		return nil
	}); err != nil {
		t.Fatalf("walk go tests: %v", err)
	}
	if len(goTests) == 0 {
		t.Fatal("no Go test files found")
	}
	return goTests
}

func validateEvidence(t *testing.T, evidence string, goTests map[string]map[string]struct{}) {
	t.Helper()

	switch {
	case strings.HasPrefix(evidence, "go_test:"):
		rest := strings.TrimPrefix(evidence, "go_test:")
		idx := strings.LastIndex(rest, ":")
		if idx <= 0 || idx == len(rest)-1 {
			t.Fatalf("invalid go_test evidence %q", evidence)
		}
		path := rest[:idx]
		testName := rest[idx+1:]
		testSet := goTests[path]
		if testSet == nil {
			t.Fatalf("go_test evidence %q references missing file %s", evidence, path)
		}
		if _, ok := testSet[testName]; !ok {
			t.Fatalf("go_test evidence %q references missing test %s", evidence, testName)
		}
	case strings.HasPrefix(evidence, "workflow:"):
		rest := strings.TrimPrefix(evidence, "workflow:")
		path := rest
		job := ""
		if idx := strings.LastIndex(rest, ":"); idx > 0 {
			candidatePath := rest[:idx]
			if strings.HasSuffix(candidatePath, ".yml") || strings.HasSuffix(candidatePath, ".yaml") {
				path = candidatePath
				job = rest[idx+1:]
			}
		}
		abs := filepath.Join(repoRoot(t), path)
		data, err := os.ReadFile(abs)
		if err != nil {
			t.Fatalf("workflow evidence %q references missing file: %v", evidence, err)
		}
		if job == "" {
			return
		}
		pattern := regexp.MustCompile(`(?m)^  ` + regexp.QuoteMeta(job) + `:`)
		if !pattern.Match(data) {
			t.Fatalf("workflow evidence %q references missing job %s", evidence, job)
		}
	case strings.Contains(evidence, "_test.go:Test"):
		idx := strings.LastIndex(evidence, ":")
		if idx <= 0 || idx == len(evidence)-1 {
			t.Fatalf("invalid legacy go_test evidence %q", evidence)
		}
		path := evidence[:idx]
		testName := evidence[idx+1:]
		testSet := goTests[path]
		if testSet == nil {
			t.Fatalf("legacy go_test evidence %q references missing file %s", evidence, path)
		}
		if _, ok := testSet[testName]; !ok {
			t.Fatalf("legacy go_test evidence %q references missing test %s", evidence, testName)
		}
	default:
		t.Fatalf("unsupported evidence format %q", evidence)
	}
}

func hasWorkflowEvidence(evidence []string) bool {
	for _, item := range evidence {
		if strings.HasPrefix(item, "workflow:") {
			return true
		}
	}
	return false
}

func hasGoTestEvidence(evidence []string) bool {
	for _, item := range evidence {
		if strings.HasPrefix(item, "go_test:") {
			return true
		}
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
