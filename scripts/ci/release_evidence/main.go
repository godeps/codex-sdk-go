package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/godeps/codex-sdk-go/internal/runtimebin"
)

type ciEvidence struct {
	WorkflowName             string   `json:"workflowName"`
	WorkflowRunURL           string   `json:"workflowRunUrl"`
	WorkflowRunID            string   `json:"workflowRunId"`
	Commit                   string   `json:"commit"`
	GoVersion                string   `json:"goVersion"`
	Commands                 []string `json:"commands"`
	StaticAnalyzer           string   `json:"staticAnalyzer"`
	RequiredRuntimeWorkflows []string `json:"requiredRuntimeWorkflows"`
}

type coverageEvidence struct {
	Packages []struct {
		Package          string  `json:"package"`
		ThresholdPercent float64 `json:"thresholdPercent"`
		CoveredPercent   float64 `json:"coveredPercent"`
		Passed           bool    `json:"passed"`
	} `json:"packages"`
}

type workflowJobsResponse struct {
	Jobs []workflowJob `json:"jobs"`
}

type workflowJob struct {
	Name string `json:"name"`
	URL  string `json:"html_url"`
}

type lockFile struct {
	Commit         string `json:"commit"`
	RuntimeVersion string `json:"runtimeVersion"`
	SchemaSHA256   string `json:"schemaSHA256"`
}

type stressSummary struct {
	DurationSeconds int      `json:"durationSeconds,omitempty"`
	Iterations      int      `json:"iterations,omitempty"`
	Commands        []string `json:"commands,omitempty"`
	FuzzTarget      string   `json:"fuzzTarget,omitempty"`
	FuzzTime        string   `json:"fuzzTime,omitempty"`
	Package         string   `json:"package,omitempty"`
}

type releaseEvidenceConfig struct {
	Output               string
	Tag                  string
	ReleaseWorkflowURL   string
	CIEvidence           string
	CoverageEvidence     string
	CILogsURL            string
	CIJobs               string
	RuntimeTestsRunID    string
	RuntimeTestsRunURL   string
	StressRunID          string
	StressRunURL         string
	StressEvidenceDir    string
	RuntimeNativeRunID   string
	RuntimeNativeRunURL  string
	RuntimeNativeJobs    string
	RuntimeBundleDir     string
	RuntimeManifest      string
	RuntimeManifestSig   string
	ProtocolManifest     string
	ReferenceLock        string
	KnownLimitationsJSON string
}

type releaseBundle struct {
	Commit             string                 `json:"commit"`
	Tag                string                 `json:"tag"`
	GoVersion          string                 `json:"goVersion"`
	RuntimeVersion     string                 `json:"runtimeVersion"`
	SchemaSHA256       string                 `json:"schemaSha256"`
	GeneratorSHA256    string                 `json:"generatorSha256"`
	CI                 ciSection              `json:"ci"`
	Coverage           coverageEvidence       `json:"coverage"`
	Stress             stressSection          `json:"stress"`
	RuntimeNative      runtimeNativeSection   `json:"runtimeNative"`
	RuntimeTests       workflowEvidence       `json:"runtimeTests"`
	Artifacts          []artifactEvidence     `json:"artifacts"`
	Provenance         []provenanceEvidence   `json:"provenance"`
	KnownLimitations   []string               `json:"knownLimitations"`
	CompletedMatrix    []targetMatrixEvidence `json:"completedMatrix"`
	ManifestSignature  manifestSignature      `json:"manifestSignature"`
	ReleaseWorkflowURL string                 `json:"releaseWorkflowUrl"`
}

type ciSection struct {
	Workflow workflowEvidence `json:"workflow"`
	Commands []string         `json:"commands"`
	Analyzer string           `json:"analyzer"`
}

type workflowEvidence struct {
	RunID  string `json:"runId"`
	RunURL string `json:"runUrl"`
}

type runtimeNativeSection struct {
	Workflow   workflowEvidence    `json:"workflow"`
	TargetJobs []targetJobEvidence `json:"targetJobs"`
}

type stressSection struct {
	Workflow  workflowEvidence `json:"workflow"`
	Summaries []stressSummary  `json:"summaries"`
}

type targetJobEvidence struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type artifactEvidence struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type provenanceEvidence struct {
	Target string `json:"target"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type targetMatrixEvidence struct {
	TargetTriple         string `json:"targetTriple"`
	RuntimeGOOS          string `json:"runtimeGoos"`
	RuntimeGOARCH        string `json:"runtimeGoarch"`
	ArchiveSHA256        string `json:"archiveSha256"`
	ManifestSHA256       string `json:"manifestSha256"`
	VersionOutput        string `json:"versionOutput"`
	InitializeSucceeded  bool   `json:"initializeSucceeded"`
	CloseSucceeded       bool   `json:"closeSucceeded"`
	ModelListSucceeded   bool   `json:"modelListSucceeded"`
	ThreadStartSucceeded bool   `json:"threadStartSucceeded"`
	TurnStartSucceeded   bool   `json:"turnStartSucceeded"`
	StreamSucceeded      bool   `json:"streamSucceeded"`
	JobURL               string `json:"jobUrl,omitempty"`
}

type manifestSignature struct {
	KeyID     string `json:"keyId"`
	Algorithm string `json:"algorithm"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("release_evidence", flag.ContinueOnError)
	var cfg releaseEvidenceConfig
	fs.StringVar(&cfg.Output, "output", "", "output bundle path")
	fs.StringVar(&cfg.Tag, "tag", "", "release tag")
	fs.StringVar(&cfg.ReleaseWorkflowURL, "release-workflow-url", "", "release workflow URL")
	fs.StringVar(&cfg.CIEvidence, "ci-evidence", "", "ci evidence json")
	fs.StringVar(&cfg.CoverageEvidence, "coverage-evidence", "", "coverage evidence json")
	fs.StringVar(&cfg.CIJobs, "ci-jobs", "", "ci jobs json")
	fs.StringVar(&cfg.RuntimeTestsRunID, "runtime-tests-run-id", "", "runtime-tests run id")
	fs.StringVar(&cfg.RuntimeTestsRunURL, "runtime-tests-run-url", "", "runtime-tests run URL")
	fs.StringVar(&cfg.StressRunID, "stress-run-id", "", "stress run id")
	fs.StringVar(&cfg.StressRunURL, "stress-run-url", "", "stress run URL")
	fs.StringVar(&cfg.StressEvidenceDir, "stress-evidence-dir", "", "stress evidence directory")
	fs.StringVar(&cfg.RuntimeNativeRunID, "runtime-native-run-id", "", "runtime-native run id")
	fs.StringVar(&cfg.RuntimeNativeRunURL, "runtime-native-run-url", "", "runtime-native run URL")
	fs.StringVar(&cfg.RuntimeNativeJobs, "runtime-native-jobs", "", "runtime-native jobs json")
	fs.StringVar(&cfg.RuntimeBundleDir, "runtime-bundle-dir", "", "runtime bundle directory")
	fs.StringVar(&cfg.RuntimeManifest, "runtime-manifest", "", "runtime manifest path")
	fs.StringVar(&cfg.RuntimeManifestSig, "runtime-manifest-signature", "", "runtime manifest signature path")
	fs.StringVar(&cfg.ProtocolManifest, "protocol-manifest", "", "protocol manifest path")
	fs.StringVar(&cfg.ReferenceLock, "reference-lock", "reference.lock.json", "reference lock path")
	fs.StringVar(&cfg.KnownLimitationsJSON, "known-limitations-json", "", "optional known limitations JSON array")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if cfg.Output == "" || cfg.Tag == "" || cfg.CIEvidence == "" || cfg.CoverageEvidence == "" || cfg.StressEvidenceDir == "" || cfg.RuntimeNativeJobs == "" || cfg.RuntimeBundleDir == "" || cfg.RuntimeManifest == "" || cfg.RuntimeManifestSig == "" || cfg.ProtocolManifest == "" {
		return errors.New("release_evidence: missing required flags")
	}

	var ci ciEvidence
	if err := readJSON(cfg.CIEvidence, &ci); err != nil {
		return err
	}
	var coverage coverageEvidence
	if err := readJSON(cfg.CoverageEvidence, &coverage); err != nil {
		return err
	}
	var nativeJobs workflowJobsResponse
	if err := readJSON(cfg.RuntimeNativeJobs, &nativeJobs); err != nil {
		return err
	}
	lock := lockFile{}
	if err := readJSON(cfg.ReferenceLock, &lock); err != nil {
		return err
	}

	manifest, manifestBytes, err := runtimebin.LoadManifest(cfg.RuntimeManifest)
	if err != nil {
		return err
	}
	envelope, err := runtimebin.LoadSignatureEnvelope(cfg.RuntimeManifestSig)
	if err != nil {
		return err
	}
	if err := runtimebin.VerifyManifestSignature(manifestBytes, envelope, runtimebin.DefaultTrustRoots()); err != nil {
		return err
	}

	if manifest.RuntimeVersion != lock.RuntimeVersion {
		return fmt.Errorf("release_evidence: runtime manifest version %s does not match lock %s", manifest.RuntimeVersion, lock.RuntimeVersion)
	}

	bundle, err := buildBundle(cfg, ci, coverage, nativeJobs, lock, manifest, envelope)
	if err != nil {
		return err
	}
	return writeJSON(cfg.Output, bundle)
}

func buildBundle(cfg releaseEvidenceConfig, ci ciEvidence, coverage coverageEvidence, nativeJobs workflowJobsResponse, lock lockFile, manifest runtimebin.Manifest, envelope runtimebin.SignatureEnvelope) (releaseBundle, error) {
	generatorHash, err := hashPaths([]string{
		filepath.Join(repoRoot(), "cmd", "codex-sdk-gen", "main.go"),
		filepath.Join(repoRoot(), "cmd", "codex-sdk-gen", "protocolgen.go"),
	})
	if err != nil {
		return releaseBundle{}, err
	}

	var limitations []string
	if cfg.KnownLimitationsJSON != "" {
		if err := readJSON(cfg.KnownLimitationsJSON, &limitations); err != nil {
			return releaseBundle{}, err
		}
	}

	manifestBytes, err := manifest.CanonicalBytes()
	if err != nil {
		return releaseBundle{}, err
	}
	manifestSHA := sha256Hex(manifestBytes)
	stressSummaries, err := loadStressSummaries(cfg.StressEvidenceDir)
	if err != nil {
		return releaseBundle{}, err
	}
	archiveEvidence, provenance, completedMatrix, err := collectRuntimeEvidence(cfg.RuntimeBundleDir, manifest, manifestSHA, nativeJobs)
	if err != nil {
		return releaseBundle{}, err
	}

	goVersion := strings.TrimSpace(ci.GoVersion)
	if goVersion == "" {
		goVersion = detectGoVersion()
	}

	return releaseBundle{
		Commit:          ci.Commit,
		Tag:             cfg.Tag,
		GoVersion:       goVersion,
		RuntimeVersion:  manifest.RuntimeVersion,
		SchemaSHA256:    lock.SchemaSHA256,
		GeneratorSHA256: generatorHash,
		CI: ciSection{
			Workflow: workflowEvidence{
				RunID:  ci.WorkflowRunID,
				RunURL: ci.WorkflowRunURL,
			},
			Commands: ci.Commands,
			Analyzer: ci.StaticAnalyzer,
		},
		Coverage: coverage,
		Stress: stressSection{
			Workflow: workflowEvidence{
				RunID:  cfg.StressRunID,
				RunURL: cfg.StressRunURL,
			},
			Summaries: stressSummaries,
		},
		RuntimeNative: runtimeNativeSection{
			Workflow: workflowEvidence{
				RunID:  cfg.RuntimeNativeRunID,
				RunURL: cfg.RuntimeNativeRunURL,
			},
			TargetJobs: collectTargetJobs(nativeJobs),
		},
		RuntimeTests: workflowEvidence{
			RunID:  cfg.RuntimeTestsRunID,
			RunURL: cfg.RuntimeTestsRunURL,
		},
		Artifacts:          archiveEvidence,
		Provenance:         provenance,
		KnownLimitations:   limitations,
		CompletedMatrix:    completedMatrix,
		ManifestSignature:  manifestSignature{KeyID: envelope.KeyID, Algorithm: envelope.Algorithm},
		ReleaseWorkflowURL: cfg.ReleaseWorkflowURL,
	}, nil
}

func collectRuntimeEvidence(root string, manifest runtimebin.Manifest, manifestSHA string, nativeJobs workflowJobsResponse) ([]artifactEvidence, []provenanceEvidence, []targetMatrixEvidence, error) {
	var artifacts []artifactEvidence
	var provenance []provenanceEvidence
	var matrix []targetMatrixEvidence
	jobURLs := mapJobURLs(nativeJobs)

	seenTargets := map[string]struct{}{}
	for _, target := range manifest.Targets {
		recordPath, err := findTargetFile(root, target.Triple, "record.json")
		if err != nil {
			return nil, nil, nil, err
		}
		evidencePath, err := findTargetEvidence(root, target.Triple)
		if err != nil {
			return nil, nil, nil, err
		}
		archivePath, err := findArchive(root, target.ArchiveName)
		if err != nil {
			return nil, nil, nil, err
		}

		var record runtimebin.ManifestTarget
		if err := readJSON(recordPath, &record); err != nil {
			return nil, nil, nil, err
		}
		var smoke runtimebin.NativeSmokeResult
		if err := readJSON(evidencePath, &smoke); err != nil {
			return nil, nil, nil, err
		}
		if record.Triple != target.Triple || smoke.TargetTriple != target.Triple {
			return nil, nil, nil, fmt.Errorf("release_evidence: mismatched target evidence for %s", target.Triple)
		}
		if _, ok := seenTargets[target.Triple]; ok {
			return nil, nil, nil, fmt.Errorf("release_evidence: duplicate target evidence for %s", target.Triple)
		}
		seenTargets[target.Triple] = struct{}{}

		sha, size, err := fileSHA256AndSize(archivePath)
		if err != nil {
			return nil, nil, nil, err
		}
		if sha != target.ArchiveSHA256 || size != target.ArchiveSize {
			return nil, nil, nil, fmt.Errorf("release_evidence: archive mismatch for %s", target.Triple)
		}
		artifacts = append(artifacts, artifactEvidence{
			Path:   filepath.ToSlash(archivePath),
			SHA256: sha,
			Size:   size,
		})

		for _, file := range target.Files {
			if file.Role != "provenance" {
				continue
			}
			provenance = append(provenance, provenanceEvidence{
				Target: target.Triple,
				Path:   file.Path,
				SHA256: file.SHA256,
			})
		}

		matrix = append(matrix, targetMatrixEvidence{
			TargetTriple:         smoke.TargetTriple,
			RuntimeGOOS:          smoke.RuntimeGOOS,
			RuntimeGOARCH:        smoke.RuntimeGOARCH,
			ArchiveSHA256:        smoke.ArchiveSHA256,
			ManifestSHA256:       manifestSHA,
			VersionOutput:        smoke.VersionOutput,
			InitializeSucceeded:  smoke.InitializeSucceeded,
			CloseSucceeded:       smoke.CloseSucceeded,
			ModelListSucceeded:   smoke.ModelListSucceeded,
			ThreadStartSucceeded: smoke.ThreadStartSucceeded,
			TurnStartSucceeded:   smoke.TurnStartSucceeded,
			StreamSucceeded:      smoke.StreamSucceeded,
			JobURL:               jobURLs[target.Triple],
		})
	}

	if len(seenTargets) != len(runtimebin.SupportedTargets()) {
		return nil, nil, nil, fmt.Errorf("release_evidence: found %d target evidence files, want %d", len(seenTargets), len(runtimebin.SupportedTargets()))
	}

	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	sort.Slice(provenance, func(i, j int) bool { return provenance[i].Target < provenance[j].Target })
	sort.Slice(matrix, func(i, j int) bool { return matrix[i].TargetTriple < matrix[j].TargetTriple })
	return artifacts, provenance, matrix, nil
}

func collectTargetJobs(jobs workflowJobsResponse) []targetJobEvidence {
	out := make([]targetJobEvidence, 0, len(jobs.Jobs))
	for _, job := range jobs.Jobs {
		if job.URL == "" {
			continue
		}
		out = append(out, targetJobEvidence(job))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func mapJobURLs(jobs workflowJobsResponse) map[string]string {
	out := map[string]string{}
	for _, job := range jobs.Jobs {
		for _, target := range runtimebin.SupportedTargets() {
			slug := target.GOOS + "_" + target.GOARCH
			if strings.Contains(job.Name, target.Triple) || strings.Contains(job.Name, slug) {
				out[target.Triple] = job.URL
			}
		}
	}
	return out
}

func findTargetFile(root string, targetTriple string, exactName string) (string, error) {
	var matches []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := filepath.Base(path)
		if name == exactName && strings.Contains(filepath.ToSlash(path), targetTriple) {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("release_evidence: expected exactly one %s for %s under %s, got %d", exactName, targetTriple, root, len(matches))
	}
	return matches[0], nil
}

func findTargetEvidence(root string, targetTriple string) (string, error) {
	var matches []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		if filepath.Base(filepath.Dir(path)) == "evidence" && strings.Contains(filepath.ToSlash(path), targetTriple) {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("release_evidence: expected exactly one native evidence JSON for %s under %s, got %d", targetTriple, root, len(matches))
	}
	return matches[0], nil
}

func findArchive(root string, archiveName string) (string, error) {
	var matches []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == archiveName {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("release_evidence: expected exactly one archive %s under %s, got %d", archiveName, root, len(matches))
	}
	return matches[0], nil
}

func loadStressSummaries(root string) ([]stressSummary, error) {
	paths, err := filepath.Glob(filepath.Join(root, "*.json"))
	if err != nil {
		return nil, err
	}
	if len(paths) != 8 {
		return nil, fmt.Errorf("release_evidence: expected 8 stress summaries under %s, got %d", root, len(paths))
	}
	summaries := make([]stressSummary, 0, len(paths))
	for _, path := range paths {
		var summary stressSummary
		if err := readJSON(path, &summary); err != nil {
			return nil, err
		}
		if filepath.Base(path) == "stress-summary.json" {
			if summary.DurationSeconds < 1800 || len(summary.Commands) == 0 {
				return nil, fmt.Errorf("release_evidence: invalid 30m stress summary at %s", path)
			}
		} else {
			if summary.FuzzTarget == "" || summary.FuzzTime != "10m" || summary.Package == "" {
				return nil, fmt.Errorf("release_evidence: invalid fuzz summary at %s", path)
			}
		}
		summaries = append(summaries, summary)
	}
	sort.Slice(summaries, func(i, j int) bool {
		left := summaries[i].FuzzTarget
		if left == "" {
			left = "0000-stress"
		}
		right := summaries[j].FuzzTarget
		if right == "" {
			right = "0000-stress"
		}
		return left < right
	})
	return summaries, nil
}

func hashPaths(paths []string) (string, error) {
	digest := sha256.New()
	for _, path := range paths {
		bytes, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		digest.Write([]byte(path))
		digest.Write([]byte{0})
		digest.Write(bytes)
		digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func fileSHA256AndSize(path string) (string, int64, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:]), int64(len(bytes)), nil
}

func detectGoVersion() string {
	out, err := exec.Command("go", "version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func sha256Hex(bytes []byte) string {
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:])
}

func readJSON(path string, value any) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, value)
}

func writeJSON(path string, value any) error {
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(bytes, '\n'), 0o644)
}
