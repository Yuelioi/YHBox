package services

import (
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
)

func TestHTTPWorkflowConsentIsExplicitAndProfileEditsRevokeIt(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	configured := HTTPOriginSettings{Slot: "http", Label: "Status API", Origin: "https://example.com", ResponseByteLimit: 4096, TimeoutMilliseconds: 5000}
	_, _, err := app.MutateSettings(func(settings *Settings) error {
		settings.Network.HTTPOrigins = []HTTPOriginSettings{configured}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	service := NewNetworkService(app)
	consent, err := service.GrantHTTPWorkflowConsent("http")
	if err != nil || consent == "" || app.Settings().Network.HTTPOrigins[0].WorkflowConsent.String() != consent {
		t.Fatalf("GrantHTTPWorkflowConsent = %q, %v", consent, err)
	}
	settingsService := NewSettingsService(app, nil)
	if err := settingsService.Update(`{"network":{"httpOrigins":[{"slot":"http","label":"Status API","origin":"https://example.com","allowPrivateNetwork":false,"responseByteLimit":8192,"timeoutMilliseconds":5000,"workflowConsent":"` + consent + `"}]}}`); err != nil {
		t.Fatal(err)
	}
	if app.Settings().Network.HTTPOrigins[0].WorkflowConsent != "" {
		t.Fatal("semantic HTTP profile edit retained prior workflow consent")
	}
	if _, err := service.GrantHTTPWorkflowConsent("missing"); err == nil {
		t.Fatal("granted consent to missing HTTP installation")
	}
}

func TestSettingsRejectInstallationSlotCollisionAcrossCapabilityFamilies(t *testing.T) {
	settings := defaultSettings()
	settings.AI.Profiles = []AIModelSettings{modelSettingsForTest("shared", "Model")}
	settings.Network.HTTPOrigins = []HTTPOriginSettings{{Slot: "shared", Label: "API", Origin: "https://example.com", ResponseByteLimit: 4096, TimeoutMilliseconds: 5000}}
	if err := settings.Validate(); err == nil {
		t.Fatal("accepted one logical slot for AI and HTTP targets")
	}
}
