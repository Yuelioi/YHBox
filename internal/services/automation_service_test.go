package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/yottaapp/yotta/internal/appcontrol"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
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
		TargetKind: automationinstalled.TargetKindDesktopWindow, AdapterKind: automationinstalled.AdapterKindWin32,
		WindowTitle: "Editor", WindowClass: "EditorWindow", InputBackend: "postmessage", CaptureBackend: "gdi", ResolveTimeoutMilliseconds: 500,
	}
	if _, _, err := app.MutateSettings(func(settings *Settings) error {
		settings.Applications.Profiles = []InstalledApplicationSettings{application}
		settings.Automation.Targets = []InstalledAutomationTargetSettings{target}
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
	if err != nil || consent == "" || app.Settings().Automation.Targets[0].WorkflowConsent.String() != consent {
		t.Fatalf("GrantWorkflowConsent = %q, %v", consent, err)
	}
	settingsService := NewSettingsService(app, nil)
	if err := settingsService.Update(`{"automation":{"targets":[{"slot":"editor-input","label":"Editor input","targetKind":"desktop-window","adapterKind":"win32","applicationSlot":"editor","windowTitle":"Editor","windowClass":"EditorWindowV2","inputBackend":"postmessage","captureBackend":"gdi","mouseCounts360":0,"resolveTimeoutMilliseconds":500,"workflowConsent":"` + consent + `"}]}}`); err != nil {
		t.Fatal(err)
	}
	if app.Settings().Automation.Targets[0].WorkflowConsent != "" {
		t.Fatal("semantic automation target edit retained prior consent")
	}
	if _, err := service.GrantWorkflowConsent("missing"); err == nil {
		t.Fatal("granted consent to missing automation target")
	}
}

func TestAutomationTargetTypesExposeSemanticKindAndNativeAdapter(t *testing.T) {
	types := NewAutomationService(nil).ListTargetTypes()
	if len(types) != 3 || types[0].TargetKind != automationinstalled.TargetKindDesktopWindow ||
		types[0].AdapterKind != automationinstalled.AdapterKindWin32 || len(types[0].Operations) == 0 ||
		len(types[0].ApplicationIdentityKinds) != 1 || types[0].ApplicationIdentityKinds[0] != automationinstalled.IdentityKindWindowsExecutable {
		t.Fatalf("target types = %#v", types)
	}
	if types[1].TargetKind != automationinstalled.TargetKindAndroidDevice || types[1].AdapterKind != automationinstalled.AdapterKindAndroidADB ||
		len(types[1].Operations) == 0 || types[1].ApplicationIdentityKinds[0] != automationinstalled.IdentityKindADBDevice {
		t.Fatalf("Android target type = %#v", types[1])
	}
	if types[2].TargetKind != automationinstalled.TargetKindBrowserCDP || types[2].AdapterKind != automationinstalled.AdapterKindBrowserCDP ||
		len(types[2].Operations) == 0 || types[2].ApplicationIdentityKinds[0] != automationinstalled.IdentityKindBrowserPage {
		t.Fatalf("browser target type = %#v", types[2])
	}
}

func TestBrowserAutomationTargetInstallsWithoutDesktopApplication(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	target := InstalledAutomationTargetSettings{
		Slot: "browser", Label: "Browser page", TargetKind: automationinstalled.TargetKindBrowserCDP, AdapterKind: automationinstalled.AdapterKindBrowserCDP,
		BrowserEndpoint: "http://127.0.0.1:9222", BrowserTargetID: "page-1",
		BrowserWebSocketURL: "ws://127.0.0.1:9222/devtools/page/page-1", BrowserTitle: "Fixture", BrowserURL: "https://example.test/",
		ResolveTimeoutMilliseconds: 1000,
	}
	if _, _, err := app.MutateSettings(func(settings *Settings) error {
		settings.Automation.Targets = []InstalledAutomationTargetSettings{target}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	drafts, err := app.Settings().Automation.InstallationDrafts(app.Settings().Applications)
	if err != nil || len(drafts) != 1 || drafts[0].Profile.BrowserTargetID != target.BrowserTargetID || drafts[0].Profile.Application.Executable != "" {
		t.Fatalf("browser drafts = %#v, %v", drafts, err)
	}
	consent, err := NewAutomationService(app).GrantWorkflowConsent(target.Slot)
	if err != nil || consent == "" {
		t.Fatalf("GrantWorkflowConsent = %q, %v", consent, err)
	}
}

func TestAndroidAutomationTargetInstallsWithoutDesktopApplication(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	target := InstalledAutomationTargetSettings{
		Slot: "android", Label: "Android emulator", TargetKind: automationinstalled.TargetKindAndroidDevice, AdapterKind: automationinstalled.AdapterKindAndroidADB,
		ADBSerial: "emulator-5554", ADBProduct: "sdk_gphone64_x86_64", ADBModel: "sdk_gphone64_x86_64", ADBDevice: "emu64xa",
		AndroidPackage: "dev.yotta.fixture", ResolveTimeoutMilliseconds: 1000,
	}
	if _, _, err := app.MutateSettings(func(settings *Settings) error {
		settings.Automation.Targets = []InstalledAutomationTargetSettings{target}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	drafts, err := app.Settings().Automation.InstallationDrafts(app.Settings().Applications)
	if err != nil || len(drafts) != 1 || drafts[0].Profile.AndroidPackage != target.AndroidPackage || drafts[0].Profile.Application.Executable != "" {
		t.Fatalf("Android drafts = %#v, %v", drafts, err)
	}
	consent, err := NewAutomationService(app).GrantWorkflowConsent(target.Slot)
	if err != nil || consent == "" {
		t.Fatalf("GrantWorkflowConsent = %q, %v", consent, err)
	}
}

func TestSettingsRejectAutomationTargetWithUnknownApplicationOrSharedSlot(t *testing.T) {
	settings := defaultSettings()
	settings.Automation.Targets = []InstalledAutomationTargetSettings{{
		Slot: "input", Label: "Input", ApplicationSlot: "missing", WindowTitle: "Editor",
		TargetKind: automationinstalled.TargetKindDesktopWindow, AdapterKind: automationinstalled.AdapterKindWin32,
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
	settings.Automation.Targets[0].ApplicationSlot = "editor"
	settings.Automation.Targets[0].Slot = "editor"
	if err := settings.Validate(); err == nil {
		t.Fatal("accepted one logical slot for application and automation targets")
	}
}
