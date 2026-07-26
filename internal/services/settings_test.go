package services

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/yottaapp/yotta/internal/artifact"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
)

func TestAppSettingsReturnsDeepSnapshot(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	first := app.Settings()
	first.Locale = "en"
	first.UI.MouseProfiles = append(first.UI.MouseProfiles, MouseProfile{Label: "mutated", Counts360: 1})
	second := app.Settings()
	if second.Locale != "zh" || len(second.UI.MouseProfiles) != 0 {
		t.Fatalf("live settings escaped through snapshot: %+v", second)
	}
}

func TestAppMutateSettingsSerializesWritersWithoutLostUpdates(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	start := make(chan struct{})
	var wg sync.WaitGroup
	updates := []func(*Settings){
		func(s *Settings) { s.Locale = "en" },
		func(s *Settings) { s.UI.Logger.ShowTime = false },
	}
	for _, update := range updates {
		update := update
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, _, err := app.MutateSettings(func(s *Settings) error { update(s); return nil }); err != nil {
				t.Errorf("MutateSettings: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	got := app.Settings()
	if got.Locale != "en" || got.UI.Logger.ShowTime {
		t.Fatalf("concurrent updates lost: %+v", got)
	}
}

func TestAppMutateSettingsSerializesCommitSideEffects(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	app.settingsSaver = func(string, *Settings) error { return nil }
	firstEffectStarted := make(chan struct{})
	releaseFirstEffect := make(chan struct{})
	var (
		mu    sync.Mutex
		order []string
	)
	firstDone := make(chan error, 1)
	go func() {
		_, _, err := app.MutateSettings(func(s *Settings) error {
			s.Locale = "en"
			return nil
		}, func(*Settings, *Settings) {
			close(firstEffectStarted)
			<-releaseFirstEffect
			mu.Lock()
			order = append(order, "first")
			mu.Unlock()
		})
		firstDone <- err
	}()
	select {
	case <-firstEffectStarted:
	case <-time.After(time.Second):
		t.Fatal("first side effect did not start")
	}
	if app.settingsUpdateMu.TryLock() {
		app.settingsUpdateMu.Unlock()
		t.Fatal("settings writer lock was released before commit side effects")
	}
	secondDone := make(chan error, 1)
	go func() {
		_, _, err := app.MutateSettings(func(s *Settings) error {
			s.UI.Logger.ShowTime = false
			return nil
		}, func(*Settings, *Settings) {
			mu.Lock()
			order = append(order, "second")
			mu.Unlock()
		})
		secondDone <- err
	}()
	close(releaseFirstEffect)
	for name, result := range map[string]<-chan error{"first": firstDone, "second": secondDone} {
		select {
		case err := <-result:
			if err != nil {
				t.Fatalf("%s update error = %v", name, err)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s update did not finish", name)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Fatalf("side effect order = %v", order)
	}
}

func TestAppMutateSettingsSaveFailureKeepsPublishedSnapshot(t *testing.T) {
	dir := t.TempDir()
	app := NewApp(filepath.Join(dir, "settings.json"), nil, zerolog.Nop())
	blocker := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("block"), 0o644); err != nil {
		t.Fatal(err)
	}
	app.settingsSaver = func(string, *Settings) error {
		return os.ErrInvalid
	}
	before := app.Settings()
	if _, after, err := app.MutateSettings(func(s *Settings) error {
		s.Locale = "en"
		return nil
	}); err == nil || after != nil {
		t.Fatalf("MutateSettings = after=%+v err=%v, want precommit failure", after, err)
	}
	if got := app.Settings(); got.Locale != before.Locale {
		t.Fatalf("failed save published memory: before=%q after=%q", before.Locale, got.Locale)
	}
}

func TestAppMutateSettingsPublishesPostCommitSyncFailure(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	wantErr := errors.New("directory sync failed")
	app.settingsSaver = func(string, *Settings) error {
		return &settingsCommittedError{err: wantErr}
	}
	_, after, err := app.MutateSettings(func(s *Settings) error {
		s.Locale = "en"
		return nil
	})
	if !errors.Is(err, wantErr) || after == nil {
		t.Fatalf("MutateSettings = after=%+v err=%v", after, err)
	}
	if got := app.Settings(); got.Locale != "en" {
		t.Fatalf("committed update was not published: %+v", got)
	}
}

func TestAppMutateSettingsPreparesAndCommitsRuntimeActivation(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	prepared, committed, aborted := 0, 0, 0
	if err := app.AttachSettingsActivator(func(before, after *Settings) (*SettingsActivationPlan, error) {
		prepared++
		if before.Locale == after.Locale {
			return nil, nil
		}
		return &SettingsActivationPlan{
			Commit: func() error { committed++; return nil },
			Abort:  func() { aborted++ },
		}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.MutateSettings(func(settings *Settings) error { settings.Locale = "en"; return nil }); err != nil {
		t.Fatal(err)
	}
	if prepared != 1 || committed != 1 || aborted != 0 {
		t.Fatalf("activation lifecycle = prepared %d committed %d aborted %d", prepared, committed, aborted)
	}
}

func TestAppMutateSettingsAbortsPreparedRuntimeOnSaveFailure(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	app.settingsSaver = func(string, *Settings) error { return errors.New("disk unavailable") }
	committed, aborted := 0, 0
	if err := app.AttachSettingsActivator(func(*Settings, *Settings) (*SettingsActivationPlan, error) {
		return &SettingsActivationPlan{
			Commit: func() error { committed++; return nil },
			Abort:  func() { aborted++ },
		}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, after, err := app.MutateSettings(func(settings *Settings) error { settings.Locale = "en"; return nil }); err == nil || after != nil {
		t.Fatalf("save failure = after %+v, err %v", after, err)
	}
	if committed != 0 || aborted != 1 || app.Settings().Locale == "en" {
		t.Fatalf("activation lifecycle = committed %d aborted %d locale %q", committed, aborted, app.Settings().Locale)
	}
}

func TestUpdateWindowSizeLogsPersistenceFailure(t *testing.T) {
	sink := NewLogSink(nil)
	logger := zerolog.New(sink)
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), sink, logger)
	app.settingsSaver = func(string, *Settings) error { return errors.New("disk unavailable") }
	app.UpdateWindowSize(1200, 800)
	log := sink.Snapshot()
	if !strings.Contains(log, "persist window size") || !strings.Contains(log, "disk unavailable") {
		t.Fatalf("persistence failure was not logged: %s", log)
	}
}

func TestSettingsValidate_OK(t *testing.T) {
	s := defaultSettings()
	if err := s.Validate(); err != nil {
		t.Fatalf("default settings should validate: %v", err)
	}
}

func TestSettingsValidateRejectsUnknownLoggerLevel(t *testing.T) {
	s := defaultSettings()
	s.UI.Logger.Level = "verbose"
	if err := s.Validate(); err == nil {
		t.Fatal("unknown logger level validated")
	}
}

func TestSettingsServiceRemovingApplicationAlsoRemovesDependentTargets(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	_, _, err := app.MutateSettings(func(settings *Settings) error {
		settings.Applications.Profiles = []InstalledApplicationSettings{{
			Slot: "htgame", Label: "HTGame", Executable: `C:\Apps\HTGame.exe`,
			ExecutableDigest: artifact.Digest("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), Arguments: []string{},
		}}
		settings.Automation.Targets = []InstalledAutomationTargetSettings{{
			Slot: "window-target", Label: "异环",
			TargetKind: automationinstalled.TargetKindDesktopWindow, AdapterKind: automationinstalled.AdapterKindWin32,
			ProfileVersion: automationinstalled.ProfileVersionV1,
			Profile: automationTargetProfile(DesktopAutomationTargetSettings{
				ApplicationSlot: "htgame", WindowTitle: "异环", WindowTitleMatch: "exact", WindowSelection: "unique",
				WindowClass: "UnrealWindow", InputBackend: "sendinput", CaptureBackend: "gdi", ResolveTimeoutMilliseconds: 500,
			}),
		}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	service := NewSettingsService(app, nil)
	if err := service.Update(`{"applications":{"profiles":[]}}`); err != nil {
		t.Fatalf("remove application: %v", err)
	}
	got := app.Settings()
	if len(got.Applications.Profiles) != 0 || len(got.Automation.Targets) != 0 {
		t.Fatalf("dependent installation survived application removal: applications=%#v targets=%#v", got.Applications.Profiles, got.Automation.Targets)
	}
}

func TestApplyMergePatch_DeepMergeUILogger(t *testing.T) {
	s := defaultSettings()
	patch := json.RawMessage(`{"ui":{"logger":{"panelOpen":false}}}`)
	if err := ApplyMergePatch(s, patch); err != nil {
		t.Fatalf("ApplyMergePatch: %v", err)
	}
	if s.UI.Logger.PanelOpen != false {
		t.Errorf("panelOpen should be false; got %v", s.UI.Logger.PanelOpen)
	}
	if s.UI.Logger.AutoScroll != true {
		t.Errorf("autoScroll should still be true (deep merge); got %v", s.UI.Logger.AutoScroll)
	}
	if s.UI.Logger.ShowTime != true {
		t.Errorf("showTime should still be true; got %v", s.UI.Logger.ShowTime)
	}
}

func TestApplyMergePatch_RejectsUnknownField(t *testing.T) {
	s := defaultSettings()
	patch := json.RawMessage(`{"nonExistentField": 42}`)
	if err := ApplyMergePatch(s, patch); err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	original := defaultSettings()
	original.UI.Logger.WriteFile = false
	original.UI.Logger.FileDir = "custom-logs"

	if err := SaveSettings(path, original); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	loaded, err := LoadSettings(path)
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if loaded.UI.Logger.WriteFile != false {
		t.Errorf("WriteFile round-trip failed")
	}
	if loaded.UI.Logger.FileDir != "custom-logs" {
		t.Errorf("FileDir round-trip failed: got %q", loaded.UI.Logger.FileDir)
	}
}

func TestLoadSettings_MissingFileReturnsDefault(t *testing.T) {
	s, err := LoadSettings(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if s == nil {
		t.Fatal("nil settings")
	}
	if s.UI.Logger.FileDir != "" {
		t.Errorf("expected managed default log directory, got %q", s.UI.Logger.FileDir)
	}
	if !s.UI.Logger.Enabled || !s.UI.Logger.LiveView || s.UI.Logger.Level != "info" {
		t.Fatalf("unexpected logger defaults: %+v", s.UI.Logger)
	}
}

func TestLoadSettings_CorruptedFileRequiresRecovery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	_ = os.WriteFile(path, []byte("{not valid json"), 0644)
	if _, err := LoadSettings(path); !errors.Is(err, ErrSettingsRecoveryRequired) {
		t.Fatalf("LoadSettings error = %v, want recovery required", err)
	}
}

func TestLoadSettingsAcceptsLegacyWorkflowConsentWithoutDiscardingConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	settings := defaultSettings()
	settings.Locale = "en"
	settings.Network.HTTPOrigins = []HTTPOriginSettings{{
		Slot: "status-api", Label: "Status API", Origin: "https://example.test/",
		ResponseByteLimit: 8192, TimeoutMilliseconds: 1000,
	}}
	payload, err := artifact.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	var legacy map[string]any
	if err := json.Unmarshal(payload, &legacy); err != nil {
		t.Fatal(err)
	}
	network := legacy["network"].(map[string]any)
	origins := network["httpOrigins"].([]any)
	origins[0].(map[string]any)["workflowConsent"] = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	legacyPayload, err := artifact.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	checksum, err := artifact.Sum(settingsPayloadDomain, legacyPayload)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(settingsEnvelope{
		Format: SettingsFormat, Version: SettingsSchemaVersion, Generation: 1,
		Checksum: checksum, Payload: legacyPayload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	store, loaded, err := OpenSettingsStore(path)
	if err != nil {
		t.Fatalf("OpenSettingsStore: %v", err)
	}
	if loaded.Locale != "en" || len(loaded.Network.HTTPOrigins) != 1 || loaded.Network.HTTPOrigins[0].Slot != "status-api" {
		t.Fatalf("legacy configuration was discarded: %#v", loaded)
	}
	if err := store.Save(loaded); err != nil {
		t.Fatal(err)
	}
	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(current), "workflowConsent") {
		t.Fatal("retired workflowConsent field was written into the current settings generation")
	}
}

func TestLoadSettings_EmptyFileDirUsesManagedDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	settings := defaultSettings()
	if err := SaveSettings(path, settings); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.UI.Logger.FileDir != "" {
		t.Errorf("expected managed FileDir, got %q", s.UI.Logger.FileDir)
	}
}

func TestUISettings_LauncherRoundTrip(t *testing.T) {
	s := &Settings{
		UI: UISettings{
			LauncherToggleHotkey: "Ctrl+Shift+L",
			LauncherItems: []LauncherBlock{
				{ID: "b1", Type: "label", Label: "战斗"},
				{ID: "b2", Type: "workflow", WorkflowID: "w1", Icon: "i-tabler-fish"},
				{ID: "b3", Type: "vsep"},
				{ID: "b4", Type: "workflow", WorkflowID: "w2"},
			},
		},
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var out Settings
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out.UI.LauncherToggleHotkey != "Ctrl+Shift+L" {
		t.Errorf("LauncherToggleHotkey: want %q, got %q", "Ctrl+Shift+L", out.UI.LauncherToggleHotkey)
	}
	if len(out.UI.LauncherItems) != 4 {
		t.Fatalf("LauncherItems: got %+v", out.UI.LauncherItems)
	}
	if out.UI.LauncherItems[0].Type != "label" || out.UI.LauncherItems[0].Label != "战斗" {
		t.Errorf("block 0 (label): got %+v", out.UI.LauncherItems[0])
	}
	if b := out.UI.LauncherItems[1]; b.Type != "workflow" || b.WorkflowID != "w1" || b.Icon != "i-tabler-fish" {
		t.Errorf("block 1 (workflow): got %+v", b)
	}
}

func TestUISettings_LauncherZeroValue(t *testing.T) {
	var s Settings
	if err := json.Unmarshal([]byte(`{"ui":{}}`), &s); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if s.UI.LauncherToggleHotkey != "" {
		t.Errorf("LauncherToggleHotkey zero: want empty, got %q", s.UI.LauncherToggleHotkey)
	}
	if len(s.UI.LauncherItems) != 0 {
		t.Errorf("LauncherItems zero: want len 0, got %d", len(s.UI.LauncherItems))
	}
}
