package services

import (
	"os"
	"testing"

	"github.com/yottaapp/yotta/internal/ai"
)

func TestConfiguredAIProfileNativeSmoke(t *testing.T) {
	settingsPath := os.Getenv("YOTTA_AI_PROFILE_SMOKE_SETTINGS")
	slot := os.Getenv("YOTTA_AI_PROFILE_SMOKE_SLOT")
	if settingsPath == "" || slot == "" {
		t.Skip("set YOTTA_AI_PROFILE_SMOKE_SETTINGS and YOTTA_AI_PROFILE_SMOKE_SLOT")
	}
	_, settings, err := OpenSettingsStore(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	configured := findAIProfile(settings, slot)
	if configured == nil {
		t.Fatalf("AI profile %q not found", slot)
	}
	service := newAIService(nil, nil, func(profile ai.ModelProfile) (ai.Provider, error) {
		return ai.NewNativeProvider(profile, ai.HTTPOptions{})
	})
	result := service.TestProfile(TestProfileRequest{Profile: *configured})
	if !result.Ok {
		t.Fatalf("native profile smoke failed: class=%s problem=%#v status=%d", result.FailureClass, result.Problem, result.HTTPStatus)
	}
}
