package all

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
)

const nodeImportPrefix = "yotta/internal/nodes/"

func TestAllImportsEveryBuiltInNodePackage(t *testing.T) {
	expected, err := builtInNodePackageDirs("..")
	if err != nil {
		t.Fatal(err)
	}

	actual, err := blankNodeImports("doc.go")
	if err != nil {
		t.Fatal(err)
	}

	if diff := missingStrings(expected, actual); len(diff) > 0 {
		t.Fatalf("internal/nodes/all is missing imports: %v", diff)
	}
	if diff := missingStrings(actual, expected); len(diff) > 0 {
		t.Fatalf("internal/nodes/all imports non-node packages: %v", diff)
	}
}

func builtInNodePackageDirs(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var pkgs []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "all" {
			continue
		}
		hasGo, err := dirHasNonTestGo(filepath.Join(root, entry.Name()))
		if err != nil {
			return nil, err
		}
		if hasGo {
			pkgs = append(pkgs, nodeImportPrefix+entry.Name())
		}
	}
	slices.Sort(pkgs)
	return pkgs, nil
}

func dirHasNonTestGo(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.Type().IsRegular() && filepath.Ext(name) == ".go" && !isTestGo(name) {
			return true, nil
		}
	}
	return false, nil
}

func isTestGo(name string) bool {
	return len(name) >= len("_test.go") && name[len(name)-len("_test.go"):] == "_test.go"
}

func blankNodeImports(filename string) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	var imports []string
	for _, spec := range file.Imports {
		if spec.Name == nil || spec.Name.Name != "_" {
			continue
		}
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, err
		}
		if len(path) >= len(nodeImportPrefix) && path[:len(nodeImportPrefix)] == nodeImportPrefix {
			imports = append(imports, path)
		}
	}
	slices.Sort(imports)
	return imports, nil
}

func missingStrings(want, got []string) []string {
	gotSet := make(map[string]struct{}, len(got))
	for _, v := range got {
		gotSet[v] = struct{}{}
	}
	var missing []string
	for _, v := range want {
		if _, ok := gotSet[v]; !ok {
			missing = append(missing, v)
		}
	}
	return missing
}
