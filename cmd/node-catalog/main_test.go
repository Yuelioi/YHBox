package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderExportsExactNodeContract31Artifacts(t *testing.T) {
	tests := []struct {
		command string
		format  string
	}{
		{command: "catalog", format: "yotta.catalog"},
		{command: "authoring", format: "yotta.node-authoring-projection"},
	}
	for _, test := range tests {
		t.Run(test.command, func(t *testing.T) {
			raw, err := render(test.command)
			if err != nil {
				t.Fatal(err)
			}
			var document struct {
				Format  string `json:"format"`
				Version string `json:"version"`
			}
			if err := json.Unmarshal(raw, &document); err != nil {
				t.Fatalf("decode %s artifact: %v", test.command, err)
			}
			if document.Format != test.format || document.Version != "3.1" {
				t.Fatalf("%s artifact = %q / %q", test.command, document.Format, document.Version)
			}
		})
	}
}

func TestRenderDocumentationComesFromAuthoringProjection(t *testing.T) {
	raw, err := render("docs")
	if err != nil {
		t.Fatal(err)
	}
	document := string(raw)
	for _, want := range []string{
		"# Yotta 3.1 built-in nodes",
		"Generated from the strict Node Authoring Projection",
		"https://schemas.yotta.dev/nodes/text/concat",
		"| input | `a` |",
		"| input | `b` |",
		"| output | `result` |",
		"Exec and Error ports: none.",
	} {
		if !strings.Contains(document, want) {
			t.Fatalf("documentation missing %q", want)
		}
	}
}

func TestRenderRejectsLegacyCommands(t *testing.T) {
	for _, command := range []string{"export", "pins", "validate", ""} {
		if _, err := render(command); err == nil {
			t.Fatalf("legacy command %q was accepted", command)
		}
	}
}
