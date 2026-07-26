package services

import "testing"

func TestSettingsRejectInstallationSlotCollisionAcrossCapabilityFamilies(t *testing.T) {
	settings := defaultSettings()
	settings.AI.Profiles = []AIModelSettings{modelSettingsForTest("shared", "Model")}
	settings.Network.HTTPOrigins = []HTTPOriginSettings{{Slot: "shared", Label: "API", Origin: "https://example.com", ResponseByteLimit: 4096, TimeoutMilliseconds: 5000}}
	if err := settings.Validate(); err == nil {
		t.Fatal("accepted one logical slot for AI and HTTP targets")
	}
}
