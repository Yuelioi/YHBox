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
