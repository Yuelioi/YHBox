package gui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettings_LoadMissingFile_Defaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nope.json")

	s := LoadSettings(path)
	if !s.Fish.AutoSell {
		t.Errorf("Fish.AutoSell default want true, got false")
	}
	if s.Cook.IntervalMs != 1500 {
		t.Errorf("Cook.IntervalMs default want 1500, got %d", s.Cook.IntervalMs)
	}
	if !s.UI.Logger.Show {
		t.Errorf("UI.Logger.Show default want true, got false")
	}
	if !s.UI.Logger.AutoScroll {
		t.Errorf("UI.Logger.AutoScroll default want true, got false")
	}
	if s.UI.Window.Width != 500 {
		t.Errorf("UI.Window.Width default want 500, got %d", s.UI.Window.Width)
	}
	if s.UI.Window.Height != 650 {
		t.Errorf("UI.Window.Height default want 650, got %d", s.UI.Window.Height)
	}
}

func TestSettings_SaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	want := &Settings{
		Fish: FishSettings{AutoSell: false},
		Cook: CookSettings{IntervalMs: 2000},
		UI: UISettings{
			Window: WindowSettings{Width: 800, Height: 900},
			Logger: LoggerSettings{Show: false, AutoScroll: false},
		},
	}
	if err := SaveSettings(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got := LoadSettings(path)
	if got.Fish.AutoSell != false {
		t.Errorf("AutoSell roundtrip: %v", got.Fish.AutoSell)
	}
	if got.Cook.IntervalMs != 2000 {
		t.Errorf("IntervalMs roundtrip: %d", got.Cook.IntervalMs)
	}
	if got.UI.Logger.Show != false {
		t.Errorf("Logger.Show roundtrip: %v", got.UI.Logger.Show)
	}
	if got.UI.Logger.AutoScroll != false {
		t.Errorf("Logger.AutoScroll roundtrip: %v", got.UI.Logger.AutoScroll)
	}
	if got.UI.Window.Width != 800 {
		t.Errorf("Window.Width roundtrip: %d", got.UI.Window.Width)
	}
	if got.UI.Window.Height != 900 {
		t.Errorf("Window.Height roundtrip: %d", got.UI.Window.Height)
	}
}

func TestSettings_LoadCorrupt_Defaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	_ = os.WriteFile(path, []byte("{not valid json"), 0644)

	s := LoadSettings(path)
	if !s.Fish.AutoSell {
		t.Errorf("corrupt file should fall back to defaults; AutoSell=%v", s.Fish.AutoSell)
	}
}

func TestSettings_LoadPartial_MissingFieldsDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	_ = os.WriteFile(path, []byte(`{"fish":{"auto_sell":false}}`), 0644)

	s := LoadSettings(path)
	if s.Fish.AutoSell {
		t.Errorf("explicit auto_sell=false should load as false")
	}
	if s.Cook.IntervalMs != 1500 {
		t.Errorf("missing cook should default; got %d", s.Cook.IntervalMs)
	}
	if !s.UI.Logger.Show {
		t.Errorf("missing logger.show should default to true; got %v", s.UI.Logger.Show)
	}
	if !s.UI.Logger.AutoScroll {
		t.Errorf("missing logger.auto_scroll should default to true; got %v", s.UI.Logger.AutoScroll)
	}
	if s.UI.Window.Width != 500 {
		t.Errorf("missing window.width should default to 500; got %d", s.UI.Window.Width)
	}
}
