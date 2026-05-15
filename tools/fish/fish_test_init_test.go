package fish

import (
	"os"
	"testing"

	"yhbox/pkg/locale"
)

// TestMain bootstraps fish package config before any test runs.
func TestMain(m *testing.M) {
	if err := LoadConfig(locale.Zh); err != nil {
		println("fish test init failed: " + err.Error())
		os.Exit(1)
	}
	os.Exit(m.Run())
}
