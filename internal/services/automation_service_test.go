package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/yottaapp/yotta/internal/appcontrol"
)

func TestAutomationWorkflowConsentIsExplicitAndTargetEditsRevokeIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Editor.exe")
	if err := os.WriteFile(path, []byte("editor-v1"), 0o700); err != nil {
		t.Fatal(err)
	}
	inspection, err := appcontrol.InspectExecutable(path)
	if err != nil {
		t.Fatal(err)
	}
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	application := InstalledApplicationSettings{
		Slot: "editor", Label: "Editor", Executable: inspection.Executable, ExecutableDigest: inspection.Digest,
		Arguments: []string{"--fixed-launch-argument"},
	}
	target := InstalledAutomationTargetSettings{
		Slot: "editor-input", Label: "Editor input", ApplicationSlot: application.Slot,
		WindowTitle: "Editor", WindowClass: "EditorWindow", InputBackend: "postmessage", CaptureBackend: "gdi", ResolveTimeoutMilliseconds: 500,
	}
	if _, _, err := app.MutateSettings(func(settings *Settings) error {
		settings.Applications.Profiles = []InstalledApplicationSettings{application}
		settings.Automation.Win32Targets = []InstalledAutomationTargetSettings{target}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	drafts, err := app.Settings().Automation.InstallationDrafts(app.Settings().Applications)
	if err != nil || len(drafts) != 1 || len(drafts[0].Profile.Application.Arguments) != 0 {
		t.Fatalf("automation drafts = %#v, %v", drafts, err)
	}
	service := NewAutomationService(app)
	consent, err := service.GrantWorkflowConsent(target.Slot)
	if err != nil || consent == "" || app.Settings().Automation.Win32Targets[0].WorkflowConsent.String() != consent {
		t.Fatalf("GrantWorkflowConsent = %q, %v", consent, err)
	}
	settingsService := NewSettingsService(app, nil)
	if err := settingsService.Update(`{"automation":{"win32Targets":[{"slot":"editor-input","label":"Editor input","applicationSlot":"editor","windowTitle":"Editor","windowClass":"EditorWindowV2","inputBackend":"postmessage","captureBackend":"gdi","mouseCounts360":0,"resolveTimeoutMilliseconds":500,"workflowConsent":"` + consent + `"}]}}`); err != nil {
		t.Fatal(err)
	}
	if app.Settings().Automation.Win32Targets[0].WorkflowConsent != "" {
		t.Fatal("semantic automation target edit retained prior consent")
	}
	if _, err := service.GrantWorkflowConsent("missing"); err == nil {
		t.Fatal("granted consent to missing automation target")
	}
}

func TestSettingsRejectAutomationTargetWithUnknownApplicationOrSharedSlot(t *testing.T) {
	settings := defaultSettings()
	settings.Automation.Win32Targets = []InstalledAutomationTargetSettings{{
		Slot: "input", Label: "Input", ApplicationSlot: "missing", WindowTitle: "Editor",
		InputBackend: "sendinput", CaptureBackend: "gdi", ResolveTimeoutMilliseconds: 500,
	}}
	if err := settings.Validate(); err == nil {
		t.Fatal("accepted automation target with unknown installed application")
	}

	path := filepath.Join(t.TempDir(), "Editor.exe")
	if err := os.WriteFile(path, []byte("editor"), 0o700); err != nil {
		t.Fatal(err)
	}
	inspection, err := appcontrol.InspectExecutable(path)
	if err != nil {
		t.Fatal(err)
	}
	settings.Applications.Profiles = []InstalledApplicationSettings{{
		Slot: "editor", Label: "Editor", Executable: inspection.Executable, ExecutableDigest: inspection.Digest, Arguments: []string{},
	}}
	settings.Automation.Win32Targets[0].ApplicationSlot = "editor"
	settings.Automation.Win32Targets[0].Slot = "editor"
	if err := settings.Validate(); err == nil {
		t.Fatal("accepted one logical slot for application and automation targets")
	}
}
