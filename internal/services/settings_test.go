package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsValidate_OK(t *testing.T) {
	s := defaultSettings()
	if err := s.Validate(); err != nil {
		t.Fatalf("default settings should validate: %v", err)
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
			LauncherGroups: []LauncherGroup{
				{ID: "g1", Name: "战斗", Items: []LauncherItem{
					{ContainerID: "c1", Icon: "i-tabler-fish"},
					{ContainerID: "c2"},
				}},
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
	if len(out.UI.LauncherGroups) != 1 || len(out.UI.LauncherGroups[0].Items) != 2 {
		t.Fatalf("LauncherGroups: got %+v", out.UI.LauncherGroups)
	}
	g := out.UI.LauncherGroups[0]
	if g.Name != "战斗" || g.Items[0].ContainerID != "c1" || g.Items[0].Icon != "i-tabler-fish" {
		t.Errorf("group: got %+v", g)
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
	if len(s.UI.LauncherGroups) != 0 {
		t.Errorf("LauncherGroups zero: want len 0, got %d", len(s.UI.LauncherGroups))
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
