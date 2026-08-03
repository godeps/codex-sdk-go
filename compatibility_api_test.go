package codex

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompatibilityPublicAPISnapshotMatchesV01Golden(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("go", "doc", "-all", ".")
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go doc -all .: %v\n%s", err, out)
	}

	got := strings.ReplaceAll(string(out), "\r\n", "\n")
	want := readCompatFile(t, "public-api.txt")
	if got == want {
		return
	}
	for _, line := range strings.Split(want, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "func ") && !strings.HasPrefix(line, "type ") {
			continue
		}
		if !strings.Contains(got, line) {
			t.Fatalf("public API is missing legacy signature %q\n--- got ---\n%s", line, got)
		}
	}
}

func TestCompatibilityCompileFixturesBuildAgainstCurrentPublicAPI(t *testing.T) {
	t.Parallel()

	root := filepath.Join(repoRoot(t), "testdata", "compat", "v0.1", "compile")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read compile fixtures: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fixtureRoot := filepath.Join(root, name)
			source, err := os.ReadFile(filepath.Join(fixtureRoot, "main.go"))
			if err != nil {
				t.Fatalf("read fixture main.go: %v", err)
			}

			temp := t.TempDir()
			goMod := "module compatfixture\n\ngo 1.25.5\n\nrequire github.com/godeps/codex-sdk-go v0.0.0\n\nreplace github.com/godeps/codex-sdk-go => " + repoRoot(t) + "\n"
			if err := os.WriteFile(filepath.Join(temp, "go.mod"), []byte(goMod), 0o644); err != nil {
				t.Fatalf("write go.mod: %v", err)
			}
			if err := os.WriteFile(filepath.Join(temp, "main.go"), source, 0o644); err != nil {
				t.Fatalf("write main.go: %v", err)
			}

			cmd := exec.Command("go", "test", "-mod=mod", ".")
			cmd.Dir = temp
			cmd.Env = append(os.Environ(), "GOWORK=off")
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go test fixture %s: %v\n%s", name, err, out)
			}
		})
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return root
}

func readCompatFile(t *testing.T, rel string) string {
	t.Helper()

	path := filepath.Join(repoRoot(t), "testdata", "compat", "v0.1", rel)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}
