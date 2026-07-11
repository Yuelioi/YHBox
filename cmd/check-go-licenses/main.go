// Command check-go-licenses validates licenses of modules reachable from the
// repository's Go packages and writes a deterministic CSV report to stdout.
package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type moduleInfo struct {
	Path    string
	Version string
	Dir     string
	Main    bool
	Replace *moduleInfo
}

type packageInfo struct {
	Standard bool
	Module   *moduleInfo
}

type moduleLicense struct {
	module  *moduleInfo
	license string
	file    string
}

var allowedLicenses = map[string]bool{
	"Apache-2.0":   true,
	"BSD-2-Clause": true,
	"BSD-3-Clause": true,
	"ISC":          true,
	"MIT":          true,
}

func main() {
	modules, err := reachableModules()
	if err != nil {
		fatal(err)
	}
	report := make([]moduleLicense, 0, len(modules))
	var violations []error
	for _, module := range modules {
		license, file, err := detectModuleLicense(module.Dir)
		if err != nil {
			violations = append(violations, fmt.Errorf("%s@%s: %w", module.Path, module.Version, err))
			continue
		}
		if !allowedLicenses[license] {
			violations = append(violations, fmt.Errorf("%s@%s: disallowed license %s", module.Path, module.Version, license))
			continue
		}
		report = append(report, moduleLicense{module: module, license: license, file: file})
	}
	if len(violations) != 0 {
		fatal(errors.Join(violations...))
	}
	sort.Slice(report, func(i, j int) bool { return report[i].module.Path < report[j].module.Path })
	writer := csv.NewWriter(os.Stdout)
	_ = writer.Write([]string{"module", "version", "license", "license_file"})
	for _, item := range report {
		_ = writer.Write([]string{item.module.Path, item.module.Version, item.license, filepath.Base(item.file)})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		fatal(err)
	}
}

func reachableModules() (map[string]*moduleInfo, error) {
	cmd := exec.Command("go", "list", "-deps", "-json", "./...")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	modules := map[string]*moduleInfo{}
	decoder := json.NewDecoder(stdout)
	for {
		var pkg packageInfo
		if err := decoder.Decode(&pkg); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}
		if pkg.Standard || pkg.Module == nil || pkg.Module.Main {
			continue
		}
		module := pkg.Module
		if module.Replace != nil {
			module = module.Replace
		}
		if module.Dir == "" {
			return nil, fmt.Errorf("module %s has no source directory", module.Path)
		}
		modules[module.Path] = module
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return modules, nil
}

func detectModuleLicense(dir string) (string, string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", "", err
	}
	var candidates []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToUpper(entry.Name())
		if strings.HasPrefix(name, "LICENSE") || strings.HasPrefix(name, "LICENCE") || strings.HasPrefix(name, "COPYING") || name == "UNLICENSE" {
			candidates = append(candidates, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Strings(candidates)
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			return "", "", err
		}
		if license := classifyLicense(string(data)); license != "" {
			return license, candidate, nil
		}
	}
	return "", "", errors.New("no recognized license file")
}

func classifyLicense(text string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(text), " "))
	switch {
	case strings.Contains(normalized, "apache license") && strings.Contains(normalized, "version 2.0"):
		return "Apache-2.0"
	case strings.Contains(normalized, "permission is hereby granted, free of charge"):
		return "MIT"
	case strings.Contains(normalized, "permission to use, copy, modify") && strings.Contains(normalized, "with or without fee"):
		return "ISC"
	case strings.Contains(normalized, "redistribution and use in source and binary forms") && strings.Contains(normalized, "neither the name"):
		return "BSD-3-Clause"
	case strings.Contains(normalized, "redistribution and use in source and binary forms"):
		return "BSD-2-Clause"
	default:
		return ""
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
