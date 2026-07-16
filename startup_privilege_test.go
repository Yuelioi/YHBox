package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestCompositionRootDoesNotElevateTheDesktopProcess(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, 0)
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
