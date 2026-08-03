package runtimebin

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrRuntimeNotInstalled = errors.New("runtimebin: runtime not installed")

type ResolvedRuntime struct {
	BinaryPath string
	RootDir    string
	PathDirs   []string
	Source     string
	Target     TargetSpec
	Version    string
}

type ResolveOptions struct {
	ExplicitBinary string
	Env            map[string]string
	CacheRoot      string
	RuntimeVersion string
	AllowPATH      bool
	Target         *TargetSpec
}

func ResolveRuntime(opts ResolveOptions) (ResolvedRuntime, error) {
	target := targetOrCurrent(opts.Target)
	if opts.ExplicitBinary != "" {
		if _, err := os.Stat(opts.ExplicitBinary); err != nil {
			return ResolvedRuntime{}, fmt.Errorf("runtimebin: explicit runtime path %s: %w", opts.ExplicitBinary, err)
		}
		return ResolvedRuntime{
			BinaryPath: opts.ExplicitBinary,
			RootDir:    filepath.Dir(filepath.Dir(opts.ExplicitBinary)),
			PathDirs:   []string{filepath.Dir(opts.ExplicitBinary)},
			Source:     "explicit",
			Target:     target,
			Version:    opts.RuntimeVersion,
		}, nil
	}

	env := opts.Env
	if env == nil {
		env = environmentMap()
	}
	if envPath, ok := env["CODEX_RUNTIME_PATH"]; ok && envPath != "" {
		if _, err := os.Stat(envPath); err != nil {
			return ResolvedRuntime{}, fmt.Errorf("runtimebin: CODEX_RUNTIME_PATH %s: %w", envPath, err)
		}
		return ResolvedRuntime{
			BinaryPath: envPath,
			RootDir:    filepath.Dir(filepath.Dir(envPath)),
			PathDirs:   []string{filepath.Dir(envPath)},
			Source:     "env",
			Target:     target,
			Version:    opts.RuntimeVersion,
		}, nil
	}

	rootDir, err := managedInstallRoot(opts.CacheRoot, opts.RuntimeVersion, target)
	if err != nil {
		return ResolvedRuntime{}, err
	}
	binaryPath := filepath.Join(rootDir, "bin", target.Executable)
	if _, err := os.Stat(binaryPath); err == nil {
		return ResolvedRuntime{
			BinaryPath: binaryPath,
			RootDir:    rootDir,
			PathDirs:   TargetPathDirs(rootDir),
			Source:     "managed",
			Target:     target,
			Version:    opts.RuntimeVersion,
		}, nil
	}

	if opts.AllowPATH {
		path, err := lookPath(binaryBaseName(target.Executable), env, opts.Env == nil)
		if err != nil {
			return ResolvedRuntime{}, fmt.Errorf("%w: install %s for %s or set CODEX_RUNTIME_PATH", ErrRuntimeNotInstalled, opts.RuntimeVersion, target.Triple)
		}
		return ResolvedRuntime{
			BinaryPath: path,
			RootDir:    filepath.Dir(path),
			PathDirs:   []string{filepath.Dir(path)},
			Source:     "path",
			Target:     target,
			Version:    opts.RuntimeVersion,
		}, nil
	}

	return ResolvedRuntime{}, fmt.Errorf("%w: install %s for %s or set an explicit path", ErrRuntimeNotInstalled, opts.RuntimeVersion, target.Triple)
}

func lookPath(file string, env map[string]string, inherit bool) (string, error) {
	if inherit {
		return exec.LookPath(file)
	}
	candidates := []string{file}
	if isWindowsEnv() && filepath.Ext(file) == "" {
		extensions := env["PATHEXT"]
		if extensions == "" {
			extensions = ".COM;.EXE;.BAT;.CMD"
		}
		for _, extension := range strings.Split(extensions, ";") {
			if extension != "" {
				candidates = append(candidates, file+strings.ToLower(extension), file+strings.ToUpper(extension))
			}
		}
	}
	for _, dir := range filepath.SplitList(env[PathEnvKey(env)]) {
		if dir == "" {
			continue
		}
		for _, candidate := range candidates {
			path := filepath.Join(dir, candidate)
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				continue
			}
			if !isWindowsEnv() && info.Mode().Perm()&0o111 == 0 {
				continue
			}
			return path, nil
		}
	}
	return "", exec.ErrNotFound
}

func PrependPathDirs(env map[string]string, dirs []string) {
	if len(dirs) == 0 {
		return
	}
	key := PathEnvKey(env)
	if isWindowsEnv() {
		for existingKey := range env {
			if strings.EqualFold(existingKey, "PATH") && existingKey != key {
				delete(env, existingKey)
			}
		}
	}
	seen := map[string]struct{}{}
	values := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		values = append(values, dir)
	}
	for _, existing := range strings.Split(env[key], string(os.PathListSeparator)) {
		if existing == "" {
			continue
		}
		if _, ok := seen[existing]; ok {
			continue
		}
		values = append(values, existing)
	}
	env[key] = strings.Join(values, string(os.PathListSeparator))
}

func PathEnvKey(env map[string]string) string {
	if !isWindowsEnv() {
		return "PATH"
	}
	for _, key := range []string{"Path", "PATH"} {
		for existing := range env {
			if existing == key {
				return existing
			}
		}
	}
	for existing := range env {
		if strings.EqualFold(existing, "PATH") {
			return existing
		}
	}
	return "PATH"
}

func environmentMap() map[string]string {
	env := map[string]string{}
	for _, entry := range os.Environ() {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	return env
}

func targetOrCurrent(target *TargetSpec) TargetSpec {
	if target != nil {
		return *target
	}
	current, err := CurrentTarget()
	if err != nil {
		panic(err)
	}
	return current
}

func binaryBaseName(executable string) string {
	if strings.HasSuffix(executable, ".exe") {
		return strings.TrimSuffix(executable, ".exe")
	}
	return executable
}

func isWindowsEnv() bool {
	return strings.EqualFold(os.Getenv("OS"), "Windows_NT") || filepath.Separator == '\\'
}
