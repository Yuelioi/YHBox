package adbexec

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	EnvADBPath       = "YOTTA_ADB_PATH"
	LegacyEnvADBPath = "YHFISH_ADB_PATH"
)

type ResolveOptions struct {
	EnvPath       string
	ExecutableDir string
	WorkingDir    string
	GOOS          string
	Stat          func(string) (os.FileInfo, error)
}

func CommandContext(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, ExecutablePath(), args...)
	hideCommandWindow(cmd)
	return cmd
}

func ExecutablePath() string {
	envPath := cleanExplicitPath(os.Getenv(EnvADBPath))
	if envPath == "" {
		envPath = cleanExplicitPath(os.Getenv(LegacyEnvADBPath))
	}
	exeDir := ""
	if exePath, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exePath)
	}
	wd, _ := os.Getwd()
	return Resolve(ResolveOptions{
		EnvPath:       envPath,
		ExecutableDir: exeDir,
		WorkingDir:    wd,
		GOOS:          runtime.GOOS,
		Stat:          os.Stat,
	})
}

func Resolve(opt ResolveOptions) string {
	if envPath := cleanExplicitPath(opt.EnvPath); envPath != "" {
		return envPath
	}
	stat := opt.Stat
	if stat == nil {
		stat = os.Stat
	}
	adbName := adbBinaryName(opt.GOOS)
	for _, candidate := range bundledCandidates(opt.ExecutableDir, opt.WorkingDir, adbName) {
		if info, err := stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return "adb"
}

func bundledCandidates(executableDir, workingDir, adbName string) []string {
	roots := []string{}
	for _, root := range []string{executableDir, workingDir} {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		roots = append(roots, root)
	}
	seen := map[string]bool{}
	candidates := []string{}
	for _, root := range roots {
		for _, rel := range []string{
			filepath.Join("platform-tools", adbName),
			filepath.Join("adb", adbName),
			filepath.Join("tools", "adb", adbName),
			filepath.Join("bin", "platform-tools", adbName),
			filepath.Join("bin", "adb", adbName),
		} {
			candidate := filepath.Clean(filepath.Join(root, rel))
			key := strings.ToLower(candidate)
			if seen[key] {
				continue
			}
			seen[key] = true
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

func adbBinaryName(goos string) string {
	if goos == "" {
		goos = runtime.GOOS
	}
	if goos == "windows" {
		return "adb.exe"
	}
	return "adb"
}

func cleanExplicitPath(path string) string {
	return strings.Trim(strings.TrimSpace(path), `"'`)
}
