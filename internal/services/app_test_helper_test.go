package services

import (
	"testing"

	"github.com/rs/zerolog"
)

func newTestApp(t testing.TB, settingsPath string, sink *LogSink, rootLog zerolog.Logger) *App {
	t.Helper()
	app, err := OpenApp(settingsPath, "", sink, rootLog)
	if err != nil {
		t.Fatal(err)
	}
	return app
}
