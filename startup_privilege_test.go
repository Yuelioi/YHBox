package main

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/desktopapp"
)

func TestCompositionRootDoesNotElevateTheDesktopProcess(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("internal", "desktopapp", "desktop.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if selector.Sel.Name == "EnsureAdmin" || selector.Sel.Name == "RelaunchAsAdmin" {
			t.Errorf("desktop composition root must not call %s; elevated effects belong behind explicit capability providers", selector.Sel.Name)
		}
		return true
	})
}

func TestDesktopMainDelegatesEmbeddedResourcesAndReportsStartupFailure(t *testing.T) {
	var stderr bytes.Buffer
	exitCode := 0
	desktopMain(func(config desktopapp.Config) error {
		if len(config.TrayIcon) == 0 {
			t.Error("tray icon was not delegated")
		}
		if _, err := config.Assets.ReadFile("frontend/dist/index.html"); err != nil {
			t.Errorf("frontend assets were not delegated: %v", err)
		}
		return errors.New("boom")
	}, &stderr, func(code int) { exitCode = code })
	if exitCode != 1 || stderr.String() != "Yotta startup failed: boom\n" {
		t.Fatalf("unexpected startup failure: exit=%d stderr=%q", exitCode, stderr.String())
	}
}
