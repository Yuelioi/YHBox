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

func TestSettingsValidate_RejectsBadCookInterval(t *testing.T) {
	s := defaultSettings()
	s.Cook.IntervalMs = 50
	if err := s.Validate(); err == nil {
		t.Fatal("expected error for IntervalMs=50")
	}
	s.Cook.IntervalMs = 99999
	if err := s.Validate(); err == nil {
		t.Fatal("expected error for IntervalMs=99999")
	}
}

func TestSettingsValidate_RejectsBadHotkeyMods(t *testing.T) {
	s := defaultSettings()
	s.UI.Battle.HotkeyMods = "Win"
	if err := s.Validate(); err == nil {
		t.Fatal("expected error for HotkeyMods=Win")
	}
}

func TestApplyMergePatch_DeepMergeUILogger(t *testing.T) {
	s := defaultSettings()
	patch := json.RawMessage(`{"ui":{"logger":{"show":false}}}`)
	if err := ApplyMergePatch(s, patch); err != nil {
		t.Fatalf("ApplyMergePatch: %v", err)
	}
	if s.UI.Logger.Show != false {
		t.Errorf("show should be false; got %v", s.UI.Logger.Show)
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
	original.Fish.AutoSell = false
	original.Cook.IntervalMs = 2500

	if err := SaveSettings(path, original); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	loaded := LoadSettings(path)
	if loaded.Fish.AutoSell != false {
		t.Errorf("AutoSell round-trip failed")
	}
	if loaded.Cook.IntervalMs != 2500 {
		t.Errorf("IntervalMs round-trip failed: got %d", loaded.Cook.IntervalMs)
	}
}

func TestLoadSettings_MissingFileReturnsDefault(t *testing.T) {
	s := LoadSettings(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if s == nil {
		t.Fatal("nil settings")
	}
	if s.Cook.IntervalMs != 1500 {
		t.Errorf("expected default IntervalMs=1500, got %d", s.Cook.IntervalMs)
	}
}

func TestLoadSettings_CorruptedFileReturnsDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	_ = os.WriteFile(path, []byte("{not valid json"), 0644)
	s := LoadSettings(path)
	if s.Cook.IntervalMs != 1500 {
		t.Errorf("expected default on corrupt; got %d", s.Cook.IntervalMs)
	}
}
