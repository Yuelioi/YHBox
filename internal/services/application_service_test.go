package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
)

func TestConfiguredApplicationIsImmediatelyUsableAndEditable(t *testing.T) {
	app := newTestApp(t, filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	path := filepath.Join(t.TempDir(), "AfterFX.exe")
	if err := os.WriteFile(path, []byte("after-effects-v1"), 0o700); err != nil {
		t.Fatal(err)
	}
	configured := InstalledApplicationSettings{Slot: "after-effects", Label: "After Effects", Executable: path, Arguments: []string{"-noui"}}
	if _, _, err := app.MutateSettings(func(settings *Settings) error {
		settings.Applications.Profiles = []InstalledApplicationSettings{configured}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	settingsService := NewSettingsService(app, nil)
	if err := os.WriteFile(path, []byte("after-effects-v2"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := settingsService.Update(`{"applications":{"profiles":[{"slot":"after-effects","label":"After Effects","executable":"` + escapeJSONPath(t, path) + `","arguments":["-noui","-fixed"]}]}}`); err != nil {
		t.Fatal(err)
	}
	drafts := app.Settings().Applications.InstallationDrafts()
	if len(drafts) != 1 || drafts[0].Slot != configured.Slot ||
		len(drafts[0].Profile.Arguments) != 2 {
		t.Fatalf("configured application drafts = %#v", drafts)
	}
}

func TestSettingsRejectInstallationSlotCollisionWithApplication(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tool.exe")
	if err := os.WriteFile(path, []byte("tool"), 0o700); err != nil {
		t.Fatal(err)
	}
	settings := defaultSettings()
	settings.Network.HTTPOrigins = []HTTPOriginSettings{{Slot: "shared", Label: "API", Origin: "https://example.com", ResponseByteLimit: 4096, TimeoutMilliseconds: 5000}}
	settings.Applications.Profiles = []InstalledApplicationSettings{{Slot: "shared", Label: "Tool", Executable: path, Arguments: []string{}}}
	if err := settings.Validate(); err == nil {
		t.Fatal("accepted one logical slot for network and application targets")
	}
}

func escapeJSONPath(t *testing.T, value string) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded[1 : len(encoded)-1])
}
