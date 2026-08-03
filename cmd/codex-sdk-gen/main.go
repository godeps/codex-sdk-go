package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type lockFile struct {
	UpstreamRepo   string       `json:"upstreamRepo"`
	Commit         string       `json:"commit"`
	PythonRoot     string       `json:"pythonRoot"`
	RuntimePackage string       `json:"runtimePackage"`
	RuntimeVersion string       `json:"runtimeVersion"`
	SchemaPath     string       `json:"schemaPath"`
	SchemaFile     string       `json:"schemaFile"`
	SchemaSHA256   string       `json:"schemaSHA256"`
	Targets        []lockTarget `json:"targets"`
}

type lockTarget struct {
	GoOS   string `json:"goos"`
	GoArch string `json:"goarch"`
	Triple string `json:"triple"`
	Asset  string `json:"asset"`
}

type manifest struct {
	SchemaVersion           int              `json:"schemaVersion"`
	Source                  manifestSource   `json:"source"`
	Counts                  manifestCounts   `json:"counts"`
	Definitions             []string         `json:"definitions"`
	SupplementalDefinitions []string         `json:"supplementalDefinitions"`
	Requests                []manifestMethod `json:"requests"`
	Notifications           []manifestMethod `json:"notifications"`
	ServerRequests          []manifestMethod `json:"serverRequests"`
	MissingDefinitions      []string         `json:"missingDefinitions"`
	MissingMethods          []string         `json:"missingMethods"`
	UnsupportedConstructs   []string         `json:"unsupportedConstructs"`
	DuplicateSymbols        []string         `json:"duplicateSymbols"`
}

type manifestSource struct {
	UpstreamRepo   string `json:"upstreamRepo"`
	Commit         string `json:"commit"`
	PythonRoot     string `json:"pythonRoot"`
	RuntimePackage string `json:"runtimePackage"`
	RuntimeVersion string `json:"runtimeVersion"`
	SchemaFile     string `json:"schemaFile"`
	SchemaSHA256   string `json:"schemaSHA256"`
}

type manifestCounts struct {
	AggregateDefinitions    int `json:"aggregateDefinitions"`
	SupplementalDefinitions int `json:"supplementalDefinitions"`
	Requests                int `json:"requests"`
	Notifications           int `json:"notifications"`
	ServerRequests          int `json:"serverRequests"`
}

type manifestMethod struct {
	Method string `json:"method"`
	Title  string `json:"title"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cmd := "verify"
	if len(args) != 0 {
		cmd = args[0]
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	lock, err := readLock(filepath.Join(root, "reference.lock.json"))
	if err != nil {
		return err
	}

	switch cmd {
	case "refresh":
		return refresh(root, lock)
	case "verify":
		return verify(root, lock)
	default:
		return fmt.Errorf("unknown command %q (supported: verify, refresh)", cmd)
	}
}

func refresh(root string, lock lockFile) error {
	referenceRoot, err := resolveReferenceRoot(root, lock)
	if err != nil {
		return err
	}
	sourceSchemaDir := filepath.Join(referenceRoot, filepath.FromSlash(filepath.Dir(lock.SchemaPath)))
	if err := copySchemaDirectory(sourceSchemaDir, filepath.Join(root, "schema")); err != nil {
		return err
	}
	if err := verifySchemaSHA(filepath.Join(root, "schema", lock.SchemaFile), lock.SchemaSHA256); err != nil {
		return err
	}
	output, err := generateProtocol(filepath.Join(root, "schema"), lock)
	if err != nil {
		return err
	}
	for path, data := range output.Files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(root, path), data, 0o644); err != nil {
			return err
		}
	}
	manifestJSON, err := marshalPretty(output.Manifest)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "protocol", "manifest.json"), manifestJSON, 0o644)
}

func verify(root string, lock lockFile) error {
	if _, err := resolveReferenceRoot(root, lock); err != nil {
		return err
	}
	schemaPath := filepath.Join(root, "schema", lock.SchemaFile)
	if err := verifySchemaSHA(schemaPath, lock.SchemaSHA256); err != nil {
		return err
	}
	output, err := generateProtocol(filepath.Join(root, "schema"), lock)
	if err != nil {
		return err
	}
	expectedJSON, err := marshalPretty(output.Manifest)
	if err != nil {
		return err
	}
	if err := verifyFileBytes(filepath.Join(root, "protocol", "manifest.json"), expectedJSON); err != nil {
		return err
	}
	for path, data := range output.Files {
		if err := verifyFileBytes(filepath.Join(root, path), data); err != nil {
			return err
		}
	}
	return nil
}

func readLock(path string) (lockFile, error) {
	var lock lockFile
	data, err := os.ReadFile(path)
	if err != nil {
		return lock, err
	}
	if err := json.Unmarshal(data, &lock); err != nil {
		return lock, err
	}
	return lock, nil
}

func resolveReferenceRoot(root string, lock lockFile) (string, error) {
	candidates := []string{}
	if env := os.Getenv("CODEX_REFERENCE_REPO"); env != "" {
		candidates = append(candidates, env)
	}
	candidates = append(candidates,
		filepath.Clean(filepath.Join(root, "../../other/codex")),
		filepath.Join(root, "_reference", "codex"),
	)

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if stat, err := os.Stat(candidate); err != nil || !stat.IsDir() {
			continue
		}
		head, err := gitHead(candidate)
		if err != nil {
			continue
		}
		if head != lock.Commit {
			continue
		}
		if _, err := os.Stat(filepath.Join(candidate, filepath.FromSlash(lock.PythonRoot))); err != nil {
			continue
		}
		return candidate, nil
	}
	return "", errors.New("unable to resolve pinned codex reference checkout")
}

func gitHead(repo string) (string, error) {
	cmd := exec.Command("git", "-C", repo, "rev-parse", "--short=10", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func copySchemaDirectory(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if err := copyFile(filepath.Join(srcDir, entry.Name()), filepath.Join(dstDir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func verifyFileBytes(path string, want []byte) error {
	got, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("%s is stale; run `go run ./cmd/codex-sdk-gen refresh`", filepath.ToSlash(path))
	}
	return nil
}

func verifySchemaSHA(path, want string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != want {
		return fmt.Errorf("schema sha mismatch: got %s want %s", got, want)
	}
	return nil
}

func marshalPretty(v any) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
