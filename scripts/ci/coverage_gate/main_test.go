package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGateRejectsInvalidFormat(t *testing.T) {
	t.Parallel()

	if _, _, _, err := parseGate("root:85"); err == nil {
		t.Fatal("expected invalid gate format to fail")
	}
}

func TestCoverageFromProfileCalculatesStatementCoverage(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	profile := filepath.Join(root, "cover.out")
	content := "mode: atomic\npkg/file.go:1.1,2.2 2 1\npkg/file.go:3.1,4.2 3 0\n"
	if err := os.WriteFile(profile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	covered, err := coverageFromProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if covered != 40 {
		t.Fatalf("coverage = %v, want 40", covered)
	}
}

func TestRunWritesReportAndFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	profile := filepath.Join(root, "cover.out")
	if err := os.WriteFile(profile, []byte("mode: atomic\npkg/file.go:1.1,2.2 1 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "coverage.json")

	err := run([]string{
		"--output", output,
		"--gate", "./internal/runtimebin:90:" + profile,
	})
	if err == nil {
		t.Fatal("expected threshold failure")
	}
	if _, statErr := os.Stat(output); statErr != nil {
		t.Fatalf("coverage report missing: %v", statErr)
	}
}
