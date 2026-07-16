package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func TestParseOptionsAcceptsEveryCommandAndResolvesPaths(t *testing.T) {
	t.Parallel()
	for _, command := range []string{"validate", "compile", "inspect", "run"} {
		command := command
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			settings := filepath.Join(root, "settings.json")
			opt, err := parseOptions([]string{
				"--data-root", root,
				"--settings", settings,
				"--principal", "test-principal",
				"--timeout", "30s",
				command, "argument",
			})
			if err != nil {
				t.Fatalf("parse options: %v", err)
			}
			if opt.command != command || opt.argument != "argument" || opt.principal != "test-principal" || opt.timeout != 30*time.Second {
				t.Fatalf("unexpected options: %#v", opt)
			}
			if opt.dataRoot != root || opt.settings != settings || !filepath.IsAbs(opt.executable) {
				t.Fatalf("paths were not preserved/resolved: %#v", opt)
			}
		})
	}
}

func TestParseOptionsRejectsInvalidInvocation(t *testing.T) {
	t.Parallel()
	for _, arguments := range [][]string{
		{},
		{"validate"},
		{"unknown", "workflow"},
		{"--timeout", "0s", "inspect", "workflow"},
		{"--timeout", "not-a-duration", "inspect", "workflow"},
	} {
		if _, err := parseOptions(arguments); err == nil {
			t.Fatalf("expected %q to be rejected", strings.Join(arguments, " "))
		}
	}
}

func TestBuildRuntimeStartsWithAnEmptyInstallationSet(t *testing.T) {
	root := t.TempDir()
	runtime, err := buildRuntime(options{
		dataRoot:   root,
		settings:   filepath.Join(root, "settings.json"),
		executable: filepath.Join(root, "Yotta.CLI.exe"),
	})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := runtime.Start(ctx); err != nil {
		t.Fatalf("start runtime: %v", err)
	}
	if err := runtime.Close(ctx); err != nil {
		t.Fatalf("close runtime: %v", err)
	}
}

func TestCompileResultViewAlwaysEmitsDiagnostics(t *testing.T) {
	view := compileResultView(compiler.CompileResult{})
	raw, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"diagnostics":null}` {
		t.Fatalf("unexpected compile view: %s", raw)
	}
}
