package services

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/yottaapp/yotta/internal/apperr"
)

func TestSettingsServiceProjectsInvalidPatch(t *testing.T) {
	app := newTestApp(t, filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	err := NewSettingsService(app, nil).Update(`{"ui":{"launcherSize":"impossibly-large"}}`)
	problem := apperr.From(err)
	if problem.ID != "settings.invalid" || problem.Category != apperr.CategoryValidation {
		t.Fatalf("problem = %#v", problem)
	}
}

func TestSettingsServiceProjectsPersistenceFailure(t *testing.T) {
	app := newTestApp(t, filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	app.settingsSaver = func(string, *Settings) error { return errors.New("disk unavailable") }
	err := NewSettingsService(app, nil).Update(`{"locale":"en"}`)
	problem := apperr.From(err)
	if problem.ID != "settings.update_failed" || !problem.Retryable {
		t.Fatalf("problem = %#v", problem)
	}
}
