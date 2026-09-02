package services

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
)

func TestAutomationSettingsPersistOnlyVersionedDiscriminatedProfile(t *testing.T) {
	typeOfSettings := reflect.TypeOf(InstalledAutomationTargetSettings{})
	for _, forbidden := range []string{"ApplicationSlot", "WindowTitle", "ADBSerial", "BrowserEndpoint", "InputBackend", "CaptureBackend"} {
		if _, exists := typeOfSettings.FieldByName(forbidden); exists {
			t.Fatalf("central automation settings schema still owns adapter field %s", forbidden)
		}
	}
	for _, required := range []string{"TargetKind", "AdapterKind", "ProfileVersion", "Profile"} {
		if _, exists := typeOfSettings.FieldByName(required); !exists {
			t.Fatalf("automation settings envelope is missing %s", required)
		}
	}
	source, err := os.ReadFile("settings.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"DesktopAutomationTargetSettings", "AndroidAutomationTargetSettings", "BrowserAutomationTargetSettings", "configured.isBrowser()", "configured.isAndroid()"} {
		if strings.Contains(string(source), forbidden) {
			t.Fatalf("central automation settings still decodes adapter payload through %s", forbidden)
		}
	}
}

func TestConfiguredAutomationTargetIsImmediatelyUsableAndEditable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Editor.exe")
	if err := os.WriteFile(path, []byte("editor-v1"), 0o700); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	application := InstalledApplicationSettings{Slot: "editor", Label: "Editor", Executable: path, Arguments: []string{"--fixed-launch-argument"}}
	target := InstalledAutomationTargetSettings{
		Slot: "editor-input", Label: "Editor input", TargetKind: automationinstalled.TargetKindDesktopWindow,
		AdapterKind: automationinstalled.AdapterKindWin32, ProfileVersion: automationinstalled.ProfileVersionV1,
		Profile: automationTargetProfile(DesktopAutomationTargetSettings{
			ApplicationSlot: application.Slot, WindowTitle: "^Editor\\s*$", WindowTitleMatch: "regex", WindowSelection: "unique",
			WindowClass: "EditorWindow", InputBackend: "postmessage", CaptureBackend: "gdi", ResolveTimeoutMilliseconds: 500,
		}),
	}
	if _, _, err := app.MutateSettings(func(settings *Settings) error {
		settings.Applications.Profiles = []InstalledApplicationSettings{application}
		settings.Automation.Targets = []InstalledAutomationTargetSettings{target}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	drafts, err := app.Settings().Automation.InstallationDrafts(app.Settings().Applications)
	if err != nil || len(drafts) != 1 {
		t.Fatalf("automation drafts = %#v, %v", drafts, err)
	}
	runtimeDrafts, err := app.Settings().Automation.InstallationDrafts(app.Settings().Applications, 4134)
	if err != nil || len(runtimeDrafts) != 1 || runtimeDrafts[0].RuntimeMouseCounts360 != 4134 {
		t.Fatalf("runtime calibration drafts = %#v, %v", runtimeDrafts, err)
	}
	sealed, err := automationinstalled.SealProfile(drafts[0].Profile)
	desktop, ok := automationinstalled.DesktopProfile(sealed)
	if err != nil || !ok || len(desktop.Application.Arguments) != 0 || desktop.WindowTitleMatch != "regex" {
		t.Fatalf("desktop profile = %#v, %v", desktop, err)
	}
	settingsService := NewSettingsService(app, nil)
	if err := settingsService.Update(`{"automation":{"targets":[{"slot":"editor-input","label":"Editor input","targetKind":"desktop-window","adapterKind":"win32","profileVersion":"1","profile":{"applicationSlot":"editor","windowTitle":"Editor","windowTitleMatch":"exact","windowSelection":"unique","windowClass":"EditorWindowV2","inputBackend":"postmessage","captureBackend":"gdi","mouseCounts360":0,"resolveTimeoutMilliseconds":500}}]}}`); err != nil {
		t.Fatal(err)
	}
	updatedDrafts, err := app.Settings().Automation.InstallationDrafts(app.Settings().Applications)
	if err != nil || len(updatedDrafts) != 1 {
		t.Fatalf("updated automation drafts = %#v, %v", updatedDrafts, err)
	}
	updated, err := automationinstalled.SealProfile(updatedDrafts[0].Profile)
	desktop, ok = automationinstalled.DesktopProfile(updated)
	if err != nil || !ok || desktop.WindowClass != "EditorWindowV2" {
		t.Fatalf("updated desktop profile = %#v, %v", desktop, err)
	}
}

func TestAutomationTargetTypesExposeSemanticKindAndNativeAdapter(t *testing.T) {
	types := NewAutomationService(nil).ListTargetTypes()
	if len(types) != 3 || types[0].TargetKind != automationinstalled.TargetKindDesktopWindow || types[0].AdapterKind != automationinstalled.AdapterKindWin32 || len(types[0].Operations) == 0 || types[0].ProfileVersion == "" || len(types[0].Resources) == 0 || len(types[0].Fields) == 0 {
		t.Fatalf("target types = %#v", types)
	}
	if types[1].TargetKind != automationinstalled.TargetKindAndroidDevice || types[1].AdapterKind != automationinstalled.AdapterKindAndroidADB || len(types[1].Operations) == 0 {
		t.Fatalf("Android target type = %#v", types[1])
	}
	if types[2].TargetKind != automationinstalled.TargetKindBrowserCDP || types[2].AdapterKind != automationinstalled.AdapterKindBrowserCDP || len(types[2].Operations) == 0 {
		t.Fatalf("browser target type = %#v", types[2])
	}
}

func TestBrowserAutomationTargetInstallsWithoutDesktopApplication(t *testing.T) {
	app := newTestApp(t, filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	target := InstalledAutomationTargetSettings{
		Slot: "browser", Label: "Browser page", TargetKind: automationinstalled.TargetKindBrowserCDP, AdapterKind: automationinstalled.AdapterKindBrowserCDP,
		ProfileVersion: automationinstalled.ProfileVersionV1,
		Profile:        automationTargetProfile(BrowserAutomationTargetSettings{BrowserEndpoint: "http://127.0.0.1:9222", BrowserTargetID: "page-1", BrowserWebSocketURL: "ws://127.0.0.1:9222/devtools/page/page-1", BrowserTitle: "Fixture", BrowserURL: "https://example.test/", ResolveTimeoutMilliseconds: 1000}),
	}
	if _, _, err := app.MutateSettings(func(settings *Settings) error {
		settings.Automation.Targets = []InstalledAutomationTargetSettings{target}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	drafts, err := app.Settings().Automation.InstallationDrafts(app.Settings().Applications)
	if err != nil || len(drafts) != 1 {
		t.Fatalf("browser drafts = %#v, %v", drafts, err)
	}
	sealed, err := automationinstalled.SealProfile(drafts[0].Profile)
	browser, ok := automationinstalled.BrowserProfile(sealed)
	if err != nil || !ok || browser.BrowserTargetID != "page-1" {
		t.Fatalf("browser profile = %#v, %v", browser, err)
	}
}

func TestAndroidAutomationTargetInstallsWithoutDesktopApplication(t *testing.T) {
	app := newTestApp(t, filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	target := InstalledAutomationTargetSettings{
		Slot: "android", Label: "Android emulator", TargetKind: automationinstalled.TargetKindAndroidDevice, AdapterKind: automationinstalled.AdapterKindAndroidADB,
		ProfileVersion: automationinstalled.ProfileVersionV1,
		Profile:        automationTargetProfile(AndroidAutomationTargetSettings{ADBSerial: "emulator-5554", ADBProduct: "sdk_gphone64_x86_64", ADBModel: "sdk_gphone64_x86_64", ADBDevice: "emu64xa", AndroidPackage: "dev.yotta.fixture", ResolveTimeoutMilliseconds: 1000}),
	}
	if _, _, err := app.MutateSettings(func(settings *Settings) error {
		settings.Automation.Targets = []InstalledAutomationTargetSettings{target}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	drafts, err := app.Settings().Automation.InstallationDrafts(app.Settings().Applications)
	if err != nil || len(drafts) != 1 {
		t.Fatalf("Android drafts = %#v, %v", drafts, err)
	}
	sealed, err := automationinstalled.SealProfile(drafts[0].Profile)
	android, ok := automationinstalled.AndroidProfile(sealed)
	if err != nil || !ok || android.AndroidPackage != "dev.yotta.fixture" {
		t.Fatalf("Android profile = %#v, %v", android, err)
	}
}

func TestSettingsRejectAutomationTargetWithUnknownApplicationOrSharedSlot(t *testing.T) {
	settings := defaultSettings()
	settings.Automation.Targets = []InstalledAutomationTargetSettings{{
		Slot: "input", Label: "Input", TargetKind: automationinstalled.TargetKindDesktopWindow, AdapterKind: automationinstalled.AdapterKindWin32,
		ProfileVersion: automationinstalled.ProfileVersionV1,
		Profile:        automationTargetProfile(DesktopAutomationTargetSettings{ApplicationSlot: "missing", WindowTitle: "Editor", WindowTitleMatch: "exact", WindowSelection: "unique", InputBackend: "sendinput", CaptureBackend: "gdi", ResolveTimeoutMilliseconds: 500}),
	}}
	if err := settings.Validate(); err == nil {
		t.Fatal("accepted automation target with unknown installed application")
	}
	path := filepath.Join(t.TempDir(), "Editor.exe")
	if err := os.WriteFile(path, []byte("editor"), 0o700); err != nil {
		t.Fatal(err)
	}
	settings.Applications.Profiles = []InstalledApplicationSettings{{Slot: "editor", Label: "Editor", Executable: path, Arguments: []string{}}}
	settings.Automation.Targets[0].Profile = automationTargetProfile(DesktopAutomationTargetSettings{ApplicationSlot: "editor", WindowTitle: "Editor", WindowTitleMatch: "exact", WindowSelection: "unique", InputBackend: "sendinput", CaptureBackend: "gdi", ResolveTimeoutMilliseconds: 500})
	settings.Automation.Targets[0].Slot = "editor"
	if err := settings.Validate(); err == nil {
		t.Fatal("accepted one logical slot for application and automation targets")
	}
}

func TestAutomationHealthFailureDoesNotExposeAdapterCause(t *testing.T) {
	health := automationHealthFailure(&automationinstalled.Failure{Code: automationinstalled.CodeCaptureFailed, Cause: errors.New(`PrintWindow denied C:\\Users\\private\\game.exe`)})
	encoded, err := json.Marshal(health)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if health.ID != automationinstalled.CodeCaptureFailed || strings.Contains(text, "PrintWindow") || strings.Contains(text, "private") || strings.Contains(text, "game.exe") {
		t.Fatalf("health leaked adapter cause: %s", text)
	}
}
