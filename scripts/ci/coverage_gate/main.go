package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type gateFlags []string

func (g *gateFlags) String() string {
	return strings.Join(*g, ",")
}

func (g *gateFlags) Set(value string) error {
	*g = append(*g, value)
	return nil
}

type packageCoverage struct {
	Package          string  `json:"package"`
	Profile          string  `json:"profile"`
	ThresholdPercent float64 `json:"thresholdPercent"`
	CoveredPercent   float64 `json:"coveredPercent"`
	Passed           bool    `json:"passed"`
}

type coverageReport struct {
	Packages []packageCoverage `json:"packages"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("coverage_gate", flag.ContinueOnError)
	var (
		output string
		gates  gateFlags
	)
	fs.StringVar(&output, "output", "", "output JSON report path")
	fs.Var(&gates, "gate", "coverage gate in package:threshold:profile format")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if output == "" {
		return errors.New("coverage_gate: --output is required")
	}
	if len(gates) == 0 {
		return errors.New("coverage_gate: at least one --gate is required")
	}

	report := coverageReport{Packages: make([]packageCoverage, 0, len(gates))}
	failed := false
	for _, raw := range gates {
		pkg, threshold, profile, err := parseGate(raw)
		if err != nil {
			return err
		}
		covered, err := coverageFromProfile(profile)
		if err != nil {
			return err
		}
		item := packageCoverage{
			Package:          pkg,
			Profile:          profile,
			ThresholdPercent: threshold,
			CoveredPercent:   covered,
			Passed:           covered >= threshold,
		}
		if !item.Passed {
			failed = true
		}
		report.Packages = append(report.Packages, item)
	}

	if err := writeJSON(output, report); err != nil {
		return err
	}
	if failed {
		return errors.New("coverage_gate: one or more coverage thresholds were not met")
	}
	return nil
}

func parseGate(raw string) (pkg string, threshold float64, profile string, err error) {
	parts := strings.SplitN(raw, ":", 3)
	if len(parts) != 3 {
		return "", 0, "", fmt.Errorf("coverage_gate: invalid gate %q", raw)
	}
	threshold, err = strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return "", 0, "", fmt.Errorf("coverage_gate: invalid threshold in %q: %w", raw, err)
	}
	if threshold < 0 || threshold > 100 {
		return "", 0, "", fmt.Errorf("coverage_gate: threshold must be between 0 and 100 in %q", raw)
	}
	if parts[0] == "" || parts[2] == "" {
		return "", 0, "", fmt.Errorf("coverage_gate: invalid gate %q", raw)
	}
	return parts[0], threshold, parts[2], nil
}

func coverageFromProfile(path string) (float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var totalStatements int64
	var coveredStatements int64
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if lineNo == 1 {
			if !strings.HasPrefix(line, "mode:") {
				return 0, fmt.Errorf("coverage_gate: invalid coverage profile header in %s", path)
			}
			continue
		}
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return 0, fmt.Errorf("coverage_gate: malformed profile line %d in %s", lineNo, path)
		}
		statements, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("coverage_gate: invalid statement count in %s line %d: %w", path, lineNo, err)
		}
		count, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("coverage_gate: invalid execution count in %s line %d: %w", path, lineNo, err)
		}
		totalStatements += statements
		if count > 0 {
			coveredStatements += statements
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	if totalStatements == 0 {
		return 0, fmt.Errorf("coverage_gate: no statements found in %s", path)
	}
	return float64(coveredStatements) * 100 / float64(totalStatements), nil
}

func writeJSON(path string, value any) error {
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(bytes, '\n'), 0o644)
}
