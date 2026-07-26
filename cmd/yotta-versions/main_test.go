package main

import (
	"encoding/json"
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
