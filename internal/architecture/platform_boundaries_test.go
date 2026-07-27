package architecture

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestPlatformNeutralPackagesDoNotImportWindowsAdapters(t *testing.T) {
	repoRoot := repositoryRoot(t)
	neutralRoots := []string{
		"internal/apperr",
		"internal/artifact",
		"internal/automation/controller",
		"internal/automation/target",
		"internal/automation/trace",
		"internal/capability",
		"internal/datatype",
		"internal/nodeadapter",
		"internal/nodeauthoring",
		"internal/nodecatalog",
		"internal/nodecontract",
		"internal/nodes",
		"internal/workflow/authoring",
		"internal/workflow/compiler",
		"internal/workflow/schema",
		"pkg/imageutil",
		"pkg/locale",
		"pkg/runctl",
		"pkg/vision",
	}
	banned := []string{
		"github.com/lxn/win",
		"golang.org/x/sys/windows",
		"github.com/yottaapp/yotta/pkg/capture",
		"github.com/yottaapp/yotta/pkg/input",
		"github.com/yottaapp/yotta/pkg/winutil",
	}
	assertNoBannedImports(t, repoRoot, neutralRoots, banned, nil)
}

func TestNodeAdapterImplementationsDoNotImportWorkflowCompiler(t *testing.T) {
	assertNoBannedImports(
		t,
		repositoryRoot(t),
		[]string{"internal/noderuntime", "internal/pluginhost"},
		[]string{"github.com/yottaapp/yotta/internal/workflow/compiler"},
		func(path string) bool {
			return strings.HasSuffix(path, "_test.go")
		},
	)
}

func TestPlatformIsolatedPackagesDoNotImportWin32Packages(t *testing.T) {
	assertNoBannedImports(
		t,
		repositoryRoot(t),
		[]string{
			"internal/automation/installed",
			"internal/hotkey",
			"internal/services/calibration",
			"internal/services/recording",
			"internal/services/tools",
		},
		[]string{
			"github.com/lxn/win",
			"golang.org/x/sys/windows",
			"github.com/yottaapp/yotta/pkg/capture",
			"github.com/yottaapp/yotta/pkg/input",
			"github.com/yottaapp/yotta/pkg/winutil",
		},
		func(path string) bool {
			name := filepath.Base(path)
			return strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, "_windows.go")
		},
	)
}

func TestBackendServicesDoNotImportWails(t *testing.T) {
	assertNoBannedImports(
		t,
		repositoryRoot(t),
		[]string{"internal/services"},
		[]string{"github.com/wailsapp/wails"},
		nil,
	)
}

func TestPresentationCompositionDelegatesStorageBackedAssemblyToLocalRuntime(t *testing.T) {
	repoRoot := repositoryRoot(t)
	files := []string{
		"cmd/yotta/main.go",
		"internal/desktopapp/desktop.go",
	}
	banned := []string{
		"github.com/yottaapp/yotta/internal/ai",
		"github.com/yottaapp/yotta/internal/appbootstrap",
		"github.com/yottaapp/yotta/internal/appcontrol",
		"github.com/yottaapp/yotta/internal/automation/installed",
		"github.com/yottaapp/yotta/internal/blob",
		"github.com/yottaapp/yotta/internal/httpegress",
		"github.com/yottaapp/yotta/internal/nodepackage",
		"github.com/yottaapp/yotta/internal/scriptengine",
		"github.com/yottaapp/yotta/internal/wasmrunner",
	}
	for _, relative := range files {
		path := filepath.Join(repoRoot, filepath.FromSlash(relative))
		assertFileHasNoBannedImports(t, repoRoot, path, banned)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		if !strings.Contains(string(raw), `"github.com/yottaapp/yotta/internal/localruntime"`) {
			t.Errorf("%s does not delegate storage-backed assembly to localruntime", relative)
		}
	}
}

func TestWorkflowCompilerDoesNotImportLegacyRuntimeOrStores(t *testing.T) {
	assertNoBannedTransitiveImports(
		t,
		repositoryRoot(t),
		[]string{"internal/workflow/compiler"},
		[]string{"github.com/yottaapp/yotta/internal/node", "github.com/yottaapp/yotta/internal/nodes"},
	)
}

func assertNoBannedTransitiveImports(t *testing.T, repoRoot string, roots, banned []string) {
	t.Helper()
	const module = "github.com/yottaapp/yotta/"
	queue := append([]string(nil), roots...)
	visited := map[string]bool{}
	for len(queue) > 0 {
		relative := queue[0]
		queue = queue[1:]
		if visited[relative] {
			continue
		}
		visited[relative] = true
		directory := filepath.Join(repoRoot, filepath.FromSlash(relative))
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatalf("scan dependency %s: %v", relative, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			path := filepath.Join(directory, entry.Name())
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			for _, spec := range file.Imports {
				importPath, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					t.Fatalf("parse import in %s: %v", path, err)
				}
				for _, prefix := range banned {
					if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
						t.Errorf("workflow compiler transitively imports %s via %s", importPath, filepath.ToSlash(path))
					}
				}
				if strings.HasPrefix(importPath, module) {
					queue = append(queue, strings.TrimPrefix(importPath, module))
				}
			}
		}
	}
}

func TestRootWiringDoesNotImportWin32Packages(t *testing.T) {
	repoRoot := repositoryRoot(t)
	entries, err := os.ReadDir(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_windows.go") {
			continue
		}
		assertFileHasNoBannedImports(t, repoRoot, filepath.Join(repoRoot, name), []string{
			"github.com/lxn/win",
			"golang.org/x/sys/windows",
		})
	}
}

func TestAutomationConsumersUseSemanticDescriptors(t *testing.T) {
	repoRoot := repositoryRoot(t)
	files := []string{
		"internal/appbootstrap/bootstrap.go",
		"internal/appbootstrap/policy.go",
		"internal/nodes/automation_input.go",
		"internal/nodes/automation_window.go",
		"internal/nodes/automation_capture.go",
		"internal/nodes/automation_playback.go",
		"frontend/src/app/editor/WorkflowInspector.vue",
		"frontend/src/views/AssetsView.vue",
		"frontend/src/views/WorkflowEditorView.vue",
	}
	for _, relative := range files {
		raw, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		for _, leaked := range []string{"win32Targets", "win32-window", "automationinstalled.TargetKind"} {
			if strings.Contains(string(raw), leaked) {
				t.Errorf("platform-specific automation identity %q leaked into %s", leaked, relative)
			}
		}
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}

func assertNoBannedImports(t *testing.T, repoRoot string, roots, banned []string, skip func(string) bool) {
	t.Helper()
	for _, relativeRoot := range roots {
		root := filepath.Join(repoRoot, filepath.FromSlash(relativeRoot))
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			if skip != nil && skip(path) {
				return nil
			}
			assertFileHasNoBannedImports(t, repoRoot, path, banned)
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", relativeRoot, err)
		}
	}
}

func assertFileHasNoBannedImports(t *testing.T, repoRoot, path string, banned []string) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Errorf("parse %s: %v", path, err)
		return
	}
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Errorf("parse import in %s: %v", path, err)
			continue
		}
		for _, prefix := range banned {
			if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
				rel, _ := filepath.Rel(repoRoot, path)
				t.Errorf("platform-neutral file %s imports %s", filepath.ToSlash(rel), importPath)
			}
		}
	}
}
