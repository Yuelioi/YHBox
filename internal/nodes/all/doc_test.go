package all

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

const nodeImportPrefix = "github.com/yottaapp/yotta/internal/nodes/"

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

func TestTargetAndWindowCategoriesStaySeparated(t *testing.T) {
	targetSelection := map[string]struct{}{
		"Win32WindowTarget": {},
		"AndroidTarget":     {},
	}
	disallowedTargets := map[string]struct{}{
		"BrowserTarget": {},
	}
	windowOperations := map[string]struct{}{
		"WaitWindow":            {},
		"WaitWindowGone":        {},
		"BringWindowForeground": {},
		"GetWindow":             {},
		"WindowState":           {},
		"MoveResizeWindow":      {},
		"CloseWindow":           {},
	}
	nonWin32Targets := map[string]struct{}{
		"AndroidTarget": {},
	}

	for _, rn := range node.All() {
		spec := rn.Spec
		if _, ok := disallowedTargets[spec.Kind]; ok {
			t.Errorf("%s must not be registered as a built-in node", spec.Kind)
		}

		if spec.NeedsForeground && !spec.NeedsWindow && !spec.NeedsTarget {
			t.Errorf("%s has NeedsForeground without NeedsWindow/NeedsTarget; foreground is a Win32 sendinput hint on target-aware actions", spec.Kind)
		}
		if spec.NeedsTarget && len(spec.TargetCapabilities) == 0 {
			t.Errorf("%s has NeedsTarget without TargetCapabilities", spec.Kind)
		}

		if _, ok := targetSelection[spec.Kind]; ok {
			if spec.Category != "Target" {
				t.Errorf("%s category = %q, want Target", spec.Kind, spec.Category)
			}
			if spec.NeedsWindow || spec.NeedsTarget || spec.NeedsForeground {
				t.Errorf("%s is a target selection node and must not require existing Win32 window services", spec.Kind)
			}
		}

		if spec.Category == "Target" {
			if _, ok := targetSelection[spec.Kind]; !ok {
				t.Errorf("%s is in Target category but is not an approved target selection node", spec.Kind)
			}
		}

		if spec.Category == "Window" {
			if _, ok := windowOperations[spec.Kind]; !ok {
				t.Errorf("%s is in Window category but is not an approved Win32 HWND operation node", spec.Kind)
			}
		}

		if _, ok := nonWin32Targets[spec.Kind]; ok {
			if hasInput(spec.Inputs, "Window", "Window") || hasOutputData(spec.Outputs, "Window", "Window") {
				t.Errorf("%s must not expose Window pins; Android targets are not Win32 HWNDs", spec.Kind)
			}
		}
	}
}

func hasInput(inputs []node.InputSpec, name, typ string) bool {
	for _, in := range inputs {
		if in.Name == name && in.Type == typ {
			return true
		}
	}
	return false
}

func hasOutputData(outputs []node.OutputSpec, name, typ string) bool {
	for _, out := range outputs {
		for _, data := range out.Data {
			if data.Name == name && data.Type == typ {
				return true
			}
		}
	}
	return false
}
