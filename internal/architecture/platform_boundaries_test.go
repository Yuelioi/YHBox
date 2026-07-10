package architecture

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestPlatformNeutralPackagesDoNotImportWindowsAdapters(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	neutralRoots := []string{
		"internal/apperr",
		"internal/automation/controller",
		"internal/automation/target",
		"internal/automation/trace",
		"internal/node",
		"internal/services/execution",
		"internal/services/expr",
		"internal/services/llm",
		"internal/services/script",
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

	for _, relativeRoot := range neutralRoots {
		root := filepath.Join(repoRoot, filepath.FromSlash(relativeRoot))
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, spec := range file.Imports {
				importPath, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					return err
				}
				for _, prefix := range banned {
					if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
						rel, _ := filepath.Rel(repoRoot, path)
						t.Errorf("platform-neutral file %s imports %s", filepath.ToSlash(rel), importPath)
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", relativeRoot, err)
		}
	}
}
