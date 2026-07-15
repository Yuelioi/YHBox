package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
)

func TestApplicationWorkflowConsentIsExplicitAndProfileEditsRevokeIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AfterFX.exe")
	if err := os.WriteFile(path, []byte("after-effects-v1"), 0o700); err != nil {
		t.Fatal(err)
	}
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	service := NewApplicationService(app)
	inspection, err := service.InspectExecutable(path)
	if err != nil {
		t.Fatal(err)
	}
	configured := InstalledApplicationSettings{Slot: "after-effects", Label: "After Effects", Executable: inspection.Executable, ExecutableDigest: inspection.Digest, Arguments: []string{"-noui"}}
	if _, _, err := app.MutateSettings(func(settings *Settings) error {
		settings.Applications.Profiles = []InstalledApplicationSettings{configured}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	consent, err := service.GrantWorkflowConsent(configured.Slot)
	if err != nil || consent == "" || app.Settings().Applications.Profiles[0].WorkflowConsent.String() != consent {
		t.Fatalf("GrantWorkflowConsent = %q, %v", consent, err)
	}
	settingsService := NewSettingsService(app, nil)
	if err := settingsService.Update(`{"applications":{"profiles":[{"slot":"after-effects","label":"After Effects","executable":"` + escapeJSONPath(t, inspection.Executable) + `","executableDigest":"` + inspection.Digest.String() + `","arguments":["-noui","-fixed"],"workflowConsent":"` + consent + `"}]}}`); err != nil {
		t.Fatal(err)
	}
	if app.Settings().Applications.Profiles[0].WorkflowConsent != "" {
		t.Fatal("semantic application profile edit retained prior consent")
	}
	if _, err := service.GrantWorkflowConsent("missing"); err == nil {
		t.Fatal("granted consent to missing application")
	}
}

func TestSettingsRejectInstallationSlotCollisionWithApplication(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tool.exe")
	if err := os.WriteFile(path, []byte("tool"), 0o700); err != nil {
		t.Fatal(err)
	}
	service := NewApplicationService(NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop()))
	inspection, err := service.InspectExecutable(path)
	if err != nil {
		t.Fatal(err)
	}
	settings := defaultSettings()
	settings.Network.HTTPOrigins = []HTTPOriginSettings{{Slot: "shared", Label: "API", Origin: "https://example.com", ResponseByteLimit: 4096, TimeoutMilliseconds: 5000}}
	settings.Applications.Profiles = []InstalledApplicationSettings{{Slot: "shared", Label: "Tool", Executable: inspection.Executable, ExecutableDigest: inspection.Digest, Arguments: []string{}}}
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
