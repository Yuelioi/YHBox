// Command yotta-versions owns repository product-version projections and
// reports the independently evolving compatibility-version inventory.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/hostapi"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/pluginprotocol"
	runartifact "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/scriptengine"
	"github.com/yottaapp/yotta/internal/services/mcpserver"
	"github.com/yottaapp/yotta/internal/services/schedule"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/internal/workflowstore"
)

const productVersionVariable = "github.com/yottaapp/yotta/pkg/version.Version"
const retiredUnifiedVersionLiteral = "3.1"

var numericSemverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

type productVersion struct {
	Major int
	Minor int
	Patch int
}

func (v productVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v productVersion) WindowsManifest() string {
	return fmt.Sprintf("%d.%d.%d.0", v.Major, v.Minor, v.Patch)
}

func parseProductVersion(value string) (productVersion, error) {
	match := numericSemverPattern.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return productVersion{}, fmt.Errorf("product version must be numeric SemVer MAJOR.MINOR.PATCH, got %q", value)
	}
	parts := [3]int{}
	for index := range parts {
		parsed, err := strconv.Atoi(match[index+1])
		if err != nil {
			return productVersion{}, fmt.Errorf("parse product version %q: %w", value, err)
		}
		parts[index] = parsed
	}
	if parts[0] > 255 || parts[1] > 255 || parts[2] > 65535 {
		return productVersion{}, errors.New("product version exceeds Windows installer numeric limits")
	}
	return productVersion{Major: parts[0], Minor: parts[1], Patch: parts[2]}, nil
}

type projection struct {
	Path      string
	Transform func([]byte, productVersion) ([]byte, error)
}

var productProjections = []projection{
	{Path: "build/config.yml", Transform: projectWailsConfig},
	{Path: "build/windows/info.json", Transform: projectWindowsInfo},
	{Path: "build/windows/nsis/wails_tools.nsh", Transform: projectNSIS},
	{Path: "build/windows/wails.exe.manifest", Transform: projectWindowsManifest},
	{Path: "build/windows/wails.dev.manifest", Transform: projectWindowsManifest},
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("usage: yotta-versions <show|check|sync|bump|inventory|ldflag>")
	}
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	current, err := readProductVersion(root)
	if err != nil {
		return err
	}
	switch arguments[0] {
	case "show":
		if len(arguments) != 1 {
			return errors.New("usage: yotta-versions show")
		}
		fmt.Println(current)
		return nil
	case "ldflag":
		if len(arguments) != 1 {
			return errors.New("usage: yotta-versions ldflag")
		}
		fmt.Printf("-X %s=%s\n", productVersionVariable, current)
		return nil
	case "check":
		if len(arguments) != 1 {
			return errors.New("usage: yotta-versions check")
		}
		changed, err := syncProductProjections(root, current, false)
		if err != nil {
			return err
		}
		if len(changed) != 0 {
			return fmt.Errorf("product version projections are stale: %s; run task version:sync", strings.Join(changed, ", "))
		}
		if err := checkRetiredUnifiedVersionLiterals(root); err != nil {
			return err
		}
		fmt.Printf("product version projections OK: %s\n", current)
		return nil
	case "sync":
		if len(arguments) != 1 {
			return errors.New("usage: yotta-versions sync")
		}
		changed, err := syncProductProjections(root, current, true)
		if err != nil {
			return err
		}
		if len(changed) == 0 {
			fmt.Printf("product version projections already current: %s\n", current)
			return nil
		}
		fmt.Printf("updated product version projections to %s: %s\n", current, strings.Join(changed, ", "))
		return nil
	case "bump":
		return bumpProductVersion(root, current, arguments[1:])
	case "inventory":
		if len(arguments) != 1 {
			return errors.New("usage: yotta-versions inventory")
		}
		printInventory(current)
		return nil
	default:
		return fmt.Errorf("unknown yotta-versions command %q", arguments[0])
	}
}

func repositoryRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if fileExists(filepath.Join(current, "VERSION")) && fileExists(filepath.Join(current, "go.mod")) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", errors.New("repository root containing VERSION and go.mod was not found")
		}
		current = parent
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func readProductVersion(root string) (productVersion, error) {
	raw, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		return productVersion{}, fmt.Errorf("read VERSION: %w", err)
	}
	if bytes.ContainsAny(bytes.TrimSpace(raw), " \t\r\n") {
		return productVersion{}, errors.New("VERSION must contain exactly one numeric SemVer")
	}
	return parseProductVersion(string(raw))
}

func syncProductProjections(root string, version productVersion, write bool) ([]string, error) {
	changed := make([]string, 0, len(productProjections))
	type pendingWrite struct {
		path string
		raw  []byte
		mode os.FileMode
	}
	pending := make([]pendingWrite, 0, len(productProjections))
	for _, item := range productProjections {
		path := filepath.Join(root, filepath.FromSlash(item.Path))
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat product version projection %s: %w", item.Path, err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read product version projection %s: %w", item.Path, err)
		}
		projected, err := item.Transform(raw, version)
		if err != nil {
			return nil, fmt.Errorf("project product version into %s: %w", item.Path, err)
		}
		if bytes.Equal(raw, projected) {
			continue
		}
		changed = append(changed, item.Path)
		pending = append(pending, pendingWrite{path: path, raw: projected, mode: info.Mode().Perm()})
	}
	if !write {
		return changed, nil
	}
	for _, item := range pending {
		if err := os.WriteFile(item.path, item.raw, item.mode); err != nil {
			return nil, fmt.Errorf("write product version projection %s: %w", item.path, err)
		}
	}
	return changed, nil
}

func bumpProductVersion(root string, current productVersion, arguments []string) error {
	flags := flag.NewFlagSet("bump", flag.ContinueOnError)
	dryRun := flags.Bool("dry-run", false, "report without writing")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("usage: yotta-versions bump [--dry-run] <patch|minor|major|MAJOR.MINOR.PATCH>")
	}
	target := current
	switch value := flags.Arg(0); value {
	case "patch":
		target.Patch++
	case "minor":
		target.Minor++
		target.Patch = 0
	case "major":
		target.Major++
		target.Minor = 0
		target.Patch = 0
	default:
		parsed, err := parseProductVersion(value)
		if err != nil {
			return err
		}
		target = parsed
	}
	if _, err := parseProductVersion(target.String()); err != nil {
		return err
	}
	if target == current {
		return fmt.Errorf("product version is already %s", current)
	}
	if *dryRun {
		fmt.Printf("would bump product version %s -> %s and refresh %d projections\n", current, target, len(productProjections))
		return nil
	}
	versionPath := filepath.Join(root, "VERSION")
	if err := os.WriteFile(versionPath, []byte(target.String()+"\n"), 0o644); err != nil {
		return fmt.Errorf("write VERSION: %w", err)
	}
	changed, err := syncProductProjections(root, target, true)
	if err != nil {
		_ = os.WriteFile(versionPath, []byte(current.String()+"\n"), 0o644)
		return err
	}
	fmt.Printf("bumped product version %s -> %s: VERSION, %s\n", current, target, strings.Join(changed, ", "))
	return nil
}

func projectWailsConfig(raw []byte, version productVersion) ([]byte, error) {
	return replaceExactlyOne(raw, regexp.MustCompile(`(?m)^  version: "[^"\r\n]+"$`), []byte(`  version: "`+version.String()+`"`))
}

type windowsInfo struct {
	Fixed struct {
		FileVersion    string `json:"file_version"`
		ProductVersion string `json:"product_version"`
	} `json:"fixed"`
	Info map[string]map[string]string `json:"info"`
}

func projectWindowsInfo(raw []byte, version productVersion) ([]byte, error) {
	var document windowsInfo
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, err
	}
	if len(document.Info) == 0 {
		return nil, errors.New("Windows info has no language blocks")
	}
	document.Fixed.FileVersion = version.WindowsManifest()
	document.Fixed.ProductVersion = version.WindowsManifest()
	for _, values := range document.Info {
		values["FileVersion"] = version.String()
		values["ProductVersion"] = version.String()
	}
	projected, err := json.MarshalIndent(document, "", "\t")
	if err != nil {
		return nil, err
	}
	return append(projected, '\n'), nil
}

func projectNSIS(raw []byte, version productVersion) ([]byte, error) {
	pattern := regexp.MustCompile(`(?m)^    !define INFO_PRODUCTVERSION "[^"\r\n]+"$`)
	return replaceExactlyOne(raw, pattern, []byte(`    !define INFO_PRODUCTVERSION "`+version.String()+`"`))
}

func projectWindowsManifest(raw []byte, version productVersion) ([]byte, error) {
	pattern := regexp.MustCompile(`(<assemblyIdentity type="win32" name="com\.yottaapp\.yotta" version=")[^"]+(" processorArchitecture="\*"/>)`)
	match := pattern.FindSubmatchIndex(raw)
	if match == nil || pattern.FindAllIndex(raw, -1) == nil || len(pattern.FindAllIndex(raw, -1)) != 1 {
		return nil, errors.New("expected exactly one Yotta assemblyIdentity")
	}
	replacement := []byte(`${1}` + version.WindowsManifest() + `${2}`)
	return pattern.ReplaceAll(raw, replacement), nil
}

func replaceExactlyOne(raw []byte, pattern *regexp.Regexp, replacement []byte) ([]byte, error) {
	if len(pattern.FindAllIndex(raw, -1)) != 1 {
		return nil, fmt.Errorf("expected exactly one match for %s", pattern)
	}
	return pattern.ReplaceAll(raw, replacement), nil
}

func printInventory(product productVersion) {
	rows := [][2]string{
		{"product", product.String()},
		{schema.Format, schema.Version},
		{datatype.Format, datatype.Version},
		{nodecontract.Format, nodecontract.Version},
		{nodeauthoring.Format, nodeauthoring.Version},
		{nodecatalog.Format, nodecatalog.Version},
		{compiler.ProgramFormat, compiler.ProgramVersion},
		{capability.DefinitionFormat, capability.DefinitionVersion},
		{capability.PlanFormat, capability.PlanVersion},
		{capability.RunGrantFormat, capability.RunGrantVersion},
		{runartifact.RecordFormat, runartifact.RecordVersion},
		{"schedule-schema", string(schedule.CurrentSchemaVersion)},
		{"host-interface", hostapi.Current},
		{"script-worker-protocol", scriptengine.Protocol},
		{"plugin-protocol", pluginprotocol.Protocol},
		{mcpserver.CatalogSearchFormat, mcpserver.CatalogSearchVersion},
		{mcpserver.CatalogDescriptionFormat, mcpserver.CatalogDescriptionVersion},
		{"blob-store-layout", blob.LayoutVersion},
		{"workflow-source-store-layout", workflowstore.SourceLayoutVersion},
		{"program-store-layout", workflowstore.ProgramLayoutVersion},
	}
	for _, row := range rows {
		fmt.Printf("%-32s %s\n", row[0], row[1])
	}
}

func checkRetiredUnifiedVersionLiterals(root string) error {
	scanRoots := []string{"internal", "pkg", "cmd", "sdk", filepath.Join("frontend", "src")}
	allowed := map[string]struct{}{
		filepath.Clean(filepath.Join("internal", "appbootstrap", "workspace_root.go")): {},
		filepath.Clean(filepath.Join("cmd", "yotta-versions", "main.go")):              {},
	}
	var violations []string
	for _, relativeRoot := range scanRoots {
		absoluteRoot := filepath.Join(root, relativeRoot)
		err := filepath.WalkDir(absoluteRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if _, ok := allowed[filepath.Clean(relative)]; ok || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".go", ".ts", ".vue", ".js", ".mjs":
			default:
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if bytes.Contains(raw, []byte(retiredUnifiedVersionLiteral)) {
				violations = append(violations, filepath.ToSlash(relative))
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("scan retired unified version literals in %s: %w", relativeRoot, err)
		}
	}
	if len(violations) != 0 {
		return fmt.Errorf("retired unified version literal %q found in current source: %s", retiredUnifiedVersionLiteral, strings.Join(violations, ", "))
	}
	return nil
}
