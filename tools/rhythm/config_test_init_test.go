package rhythm

import (
	"os"
	"testing"

	"yhbox/pkg/locale"
)

// TestMain bootstraps the rhythm package config before any test runs.
// LoadConfig must succeed for tests that depend on Configs / HSVRanges.
func TestMain(m *testing.M) {
	if err := LoadConfig(locale.Zh); err != nil {
		println("rhythm test init failed: " + err.Error())
		os.Exit(1)
	}
	os.Exit(m.Run())
}
