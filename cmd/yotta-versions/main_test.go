package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseProductVersionAndWindowsProjection(t *testing.T) {
	version, err := parseProductVersion("3.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if version.String() != "3.1.0" || version.WindowsManifest() != "3.1.0.0" {
		t.Fatalf("unexpected projections: %s / %s", version, version.WindowsManifest())
	}
	for _, invalid := range []string{"3.1", "03.1.0", "3.1.0-beta.1", "256.0.0", "1.256.0", "1.0.65536"} {
		if _, err := parseProductVersion(invalid); err == nil {
			t.Fatalf("parseProductVersion(%q) succeeded", invalid)
		}
	}
}

func TestProjectionFormats(t *testing.T) {
	version, _ := parseProductVersion("3.2.4")
	config, err := projectWailsConfig([]byte("version: '3'\ninfo:\n  version: \"3.1.0\"\n"), version)
	if err != nil || !strings.Contains(string(config), `version: "3.2.4"`) {
		t.Fatalf("projectWailsConfig = %q, %v", config, err)
	}
	manifest, err := projectWindowsManifest([]byte(`<assemblyIdentity type="win32" name="com.yottaapp.yotta" version="3.1.0" processorArchitecture="*"/>`), version)
	if err != nil || !strings.Contains(string(manifest), `version="3.2.4.0"`) {
		t.Fatalf("projectWindowsManifest = %q, %v", manifest, err)
	}
	nsis, err := projectNSIS([]byte("    !define INFO_PRODUCTVERSION \"3.1.0\"\n"), version)
	if err != nil || !strings.Contains(string(nsis), `"3.2.4"`) {
		t.Fatalf("projectNSIS = %q, %v", nsis, err)
	}
	info, err := projectWindowsInfo([]byte(`{"fixed":{"file_version":"3.1.0"},"info":{"0409":{"ProductVersion":"3.1.0"}}}`), version)
	if err != nil {
		t.Fatal(err)
	}
	var projected windowsInfo
	if err := json.Unmarshal(info, &projected); err != nil {
		t.Fatal(err)
	}
	if projected.Fixed.FileVersion != "3.2.4.0" || projected.Fixed.ProductVersion != "3.2.4.0" ||
		projected.Info["0409"]["FileVersion"] != "3.2.4" || projected.Info["0409"]["ProductVersion"] != "3.2.4" {
		t.Fatalf("unexpected Windows info projection: %+v", projected)
	}
}

func TestProjectionRejectsAmbiguousOwners(t *testing.T) {
	version, _ := parseProductVersion("3.2.4")
	_, err := projectWailsConfig([]byte("  version: \"1.0.0\"\n  version: \"2.0.0\"\n"), version)
	if err == nil {
		t.Fatal("projectWailsConfig accepted multiple product version owners")
	}
}

func TestVersionInventoryTracksUserDataAndExcludesDerivedCaches(t *testing.T) {
	version, err := parseProductVersion("4.0.0")
	if err != nil {
		t.Fatal(err)
	}
	domains, err := currentVersionDomains(version)
	if err != nil {
		t.Fatal(err)
	}
	byName := make(map[string]bool, len(domains))
	for _, domain := range domains {
		byName[domain.Name] = true
		foundCurrent := false
		for _, readable := range domain.ReadableVersions {
			foundCurrent = foundCurrent || readable == domain.CurrentVersion
		}
		if !foundCurrent {
			t.Fatalf("domain %q does not read its writer version", domain.Name)
		}
	}
	for _, required := range []string{
		"yotta.workflow", "yotta.workflow-bundle", "snippet-schema", "yotta.run-summary",
		"yotta.node-package-registry", "automation-target-profile", "blob-store-layout",
	} {
		if !byName[required] {
			t.Fatalf("release compatibility inventory is missing %q", required)
		}
	}
	for _, derived := range []string{"yotta.program", "program-store-layout", "builtin-node-generator"} {
		if byName[derived] {
			t.Fatalf("derived domain %q must not become a durable compatibility promise", derived)
		}
	}
}

func TestCheckIgnoresArbitrarySourceText(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "VERSION", "4.0.0\n")
	writeTestFile(t, root, "go.mod", "module example.test/version-fixture\n")
	writeTestFile(t, root, "build/config.yml", "info:\n  version: \"0.0.0\"\n")
	writeTestFile(t, root, "build/windows/info.json", `{"fixed":{},"info":{"0409":{}}}`)
	writeTestFile(t, root, "build/windows/nsis/wails_tools.nsh", "    !define INFO_PRODUCTVERSION \"0.0.0\"\n")
	for _, name := range []string{"wails.exe.manifest", "wails.dev.manifest"} {
		writeTestFile(t, root, filepath.Join("build", "windows", name),
			`<assemblyIdentity type="win32" name="com.yottaapp.yotta" version="0.0.0.0" processorArchitecture="*"/>`)
	}
	for _, directory := range []string{"internal", "pkg", "cmd", "sdk"} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeTestFile(t, root, "frontend/src/timeline.ts", `export const occurredAt = "01:02:03.184"`)

	version, err := readProductVersion(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := syncProductProjections(root, version, true); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	if err := run([]string{"check"}); err != nil {
		t.Fatalf("check rejected unrelated source text: %v", err)
	}
}

func writeTestFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
