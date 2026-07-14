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
	app.settingsPath = filepath.Join(blocker, "settings.json")
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

func TestSaveSettingsReplaceFailurePreservesOldFileAndCleansTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	old := []byte(`{"locale":"zh"}`)
	if err := os.WriteFile(path, old, 0o644); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("replace failed")
	err := saveSettingsBytes(path, []byte(`{"locale":"en"}`), func(string, string) error {
		return wantErr
	}, func(string) error { return nil })
	if !errors.Is(err, wantErr) {
		t.Fatalf("save error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(old) {
		t.Fatalf("old file changed: %q, err=%v", got, err)
	}
	temps, err := filepath.Glob(filepath.Join(dir, ".settings.json.tmp-*"))
	if err != nil || len(temps) != 0 {
		t.Fatalf("temp files = %v, err=%v", temps, err)
	}
}

func TestSaveSettingsClassifiesDirectorySyncFailureAsCommitted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	wantErr := errors.New("directory sync failed")
	err := saveSettingsBytes(path, []byte(`{"locale":"en"}`), os.Rename, func(string) error { return wantErr })
	if !errors.Is(err, wantErr) || !settingsSaveCommitted(err) {
		t.Fatalf("save error = %v, committed=%v", err, settingsSaveCommitted(err))
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil || string(got) != `{"locale":"en"}` {
		t.Fatalf("committed file = %q, err=%v", got, readErr)
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

func TestApplyMergePatch_AcceptsShowNodeEnter(t *testing.T) {
	s := defaultSettings()
	patch := json.RawMessage(`{"ui":{"logger":{"showNodeEnter":true}}}`)
	if err := ApplyMergePatch(s, patch); err != nil {
		t.Fatalf("showNodeEnter patch must be accepted, got: %v", err)
	}
	if !s.UI.Logger.ShowNodeEnter {
		t.Fatalf("ShowNodeEnter should be true after patch")
	}
	// deep-merge unrelated logger fields preserved
	if !s.UI.Logger.ShowTime {
		t.Fatalf("ShowTime should remain true (deep merge)")
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
	loaded := LoadSettings(path)
	if loaded.UI.Logger.WriteFile != false {
		t.Errorf("WriteFile round-trip failed")
	}
	if loaded.UI.Logger.FileDir != "custom-logs" {
		t.Errorf("FileDir round-trip failed: got %q", loaded.UI.Logger.FileDir)
	}
}

func TestLoadSettings_MissingFileReturnsDefault(t *testing.T) {
	s := LoadSettings(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if s == nil {
		t.Fatal("nil settings")
	}
	if s.UI.Logger.FileDir != "logs" {
		t.Errorf("expected default FileDir=logs, got %q", s.UI.Logger.FileDir)
	}
	if !s.UI.Logger.Enabled || !s.UI.Logger.LiveView || s.UI.Logger.Level != "info" {
		t.Fatalf("unexpected logger defaults: %+v", s.UI.Logger)
	}
}

func TestLoadSettings_CorruptedFileReturnsDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	_ = os.WriteFile(path, []byte("{not valid json"), 0644)
	s := LoadSettings(path)
	if s.UI.Logger.FileDir != "logs" {
		t.Errorf("expected default on corrupt; got %q", s.UI.Logger.FileDir)
	}
}

func TestLoadSettings_EmptyFileDirFallsToDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	// 写一个没有 fileDir 字段的 settings（空字符串等同于未设置）
	_ = os.WriteFile(path, []byte(`{"ui":{"logger":{"panelOpen":true}},"locale":"zh","capture":{"method":"auto"},"ui":{"window":{"width":1100,"height":720}}}`), 0644)
	s := LoadSettings(path)
	if s.UI.Logger.FileDir != "logs" {
		t.Errorf("expected FileDir=logs fallback, got %q", s.UI.Logger.FileDir)
	}
}

func TestUISettings_LauncherRoundTrip(t *testing.T) {
	s := &Settings{
		UI: UISettings{
			LauncherToggleHotkey: "Ctrl+Shift+L",
			LauncherItems: []LauncherBlock{
				{ID: "b1", Type: "label", Label: "战斗"},
				{ID: "b2", Type: "container", ContainerID: "c1", Icon: "i-tabler-fish"},
				{ID: "b3", Type: "vsep"},
				{ID: "b4", Type: "container", ContainerID: "c2"},
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
	if b := out.UI.LauncherItems[1]; b.Type != "container" || b.ContainerID != "c1" || b.Icon != "i-tabler-fish" {
		t.Errorf("block 1 (container): got %+v", b)
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

func TestLoadSettings_UnmarshalIntoBase(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "s.json")

	// 1) 部分字段缺失 → 缺的保默认, 有的生效
	_ = os.WriteFile(p, []byte(`{"locale":"en"}`), 0o644)
	s := LoadSettings(p)
	if s.Locale != "en" {
		t.Fatalf("locale: want en, got %q", s.Locale)
	}
	if s.UI.Logger.FileDir != "logs" || s.Capture.Method != "auto" {
		t.Fatalf("missing fields should keep defaults, got FileDir=%q Method=%q", s.UI.Logger.FileDir, s.Capture.Method)
	}
	if s.UI.RecordingStopHotkey != "F12" {
		t.Fatalf("missing hotkey should keep default F12, got %q", s.UI.RecordingStopHotkey)
	}

	// 2) Window 显式 0 → 兜底
	_ = os.WriteFile(p, []byte(`{"ui":{"window":{"width":0,"height":0}}}`), 0o644)
	s = LoadSettings(p)
	if s.UI.Window.Width != 1100 || s.UI.Window.Height != 720 {
		t.Fatalf("window 0 should fall back, got %dx%d", s.UI.Window.Width, s.UI.Window.Height)
	}

	// 3) 非法 RecordingMouseMode → relative
	_ = os.WriteFile(p, []byte(`{"ui":{"recordingMouseMode":"bogus"}}`), 0o644)
	s = LoadSettings(p)
	if s.UI.RecordingMouseMode != "relative" {
		t.Fatalf("bogus mouse mode should fall back to relative, got %q", s.UI.RecordingMouseMode)
	}

	// 4) 损坏 JSON → 全新默认
	_ = os.WriteFile(p, []byte(`{bad`), 0o644)
	s = LoadSettings(p)
	if s.Locale != "zh" || s.UI.Window.Width != 1100 {
		t.Fatalf("corrupt should return fresh defaults, got locale=%q w=%d", s.Locale, s.UI.Window.Width)
	}
}
