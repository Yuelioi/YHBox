package desktopapp

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var stableProblemCall = regexp.MustCompile(`(?:apperr\.New(?:Retryable)?|problem|projectError|toolError)\("([a-z][a-z0-9_.-]+)"`)

func TestServiceProblemIDsHaveBothTranslations(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	zh := mustReadContractFile(t, filepath.Join(root, "frontend", "src", "i18n", "locales", "zh", "app.ts"))
	en := mustReadContractFile(t, filepath.Join(root, "frontend", "src", "i18n", "locales", "en", "app.ts"))
	ids := map[string]string{}
	servicesRoot := filepath.Join(root, "internal", "services")
	if err := filepath.WalkDir(servicesRoot, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && entry.IsDir() && entry.Name() == "mcpserver" {
			return filepath.SkipDir
		}
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		source := mustReadContractFile(t, path)
		for _, match := range stableProblemCall.FindAllStringSubmatch(source, -1) {
			ids[match[1]] = path
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for id, source := range ids {
		needle := "'" + id + "'"
		if !strings.Contains(zh, needle) || !strings.Contains(en, needle) {
			t.Errorf("stable Problem %q from %s lacks zh/en error translation", id, source)
		}
	}
}

func TestFrontendHasNoLegacyRawErrorEventFallback(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "frontend", "src"))
	for _, relative := range []string{
		filepath.Join("stores", "recording.ts"),
		filepath.Join("views", "SettingsAutomation.vue"),
	} {
		source := mustReadContractFile(t, filepath.Join(root, relative))
		if strings.Contains(source, "payload.error") {
			t.Errorf("%s still accepts the legacy payload.error event bypass", relative)
		}
	}
}

func mustReadContractFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
