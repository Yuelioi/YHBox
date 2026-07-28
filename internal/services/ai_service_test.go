package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/yottaapp/yotta/internal/ai"
)

func TestProfileUsesProviderNativeGenerationAndStoredCredential(t *testing.T) {
	store := newFakeSecretStore()
	secrets := NewAISecrets(store)
	if err := secrets.SetSlot("primary", "stored-secret"); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/responses" || request.Header.Get("Authorization") != "Bearer stored-secret" {
			t.Errorf("request = %s, authorization = %q", request.URL.Path, request.Header.Get("Authorization"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
			"id":"response-1","model":"gpt-resolved","status":"completed",
			"output":[{"type":"message","content":[{"type":"output_text","text":"OK"}]}],
			"usage":{"input_tokens":4,"output_tokens":1}
		}`))
	}))
	defer server.Close()
	service := newAIService(nil, secrets, func(profile ai.ModelProfile) (ai.Provider, error) {
		return ai.NewNativeProvider(profile, ai.HTTPOptions{})
	})
	configured := modelSettingsForTest("primary", "Primary")
	configured.Endpoint = server.URL + "/v1/responses"
	configured.AllowLocalHTTP = true
	result := service.TestProfile(TestProfileRequest{Profile: configured})
	if !result.Ok || result.Provider != ai.ProviderOpenAIResponses || result.ResolvedModel != "gpt-resolved" || result.Finish != ai.FinishCompleted {
		t.Fatalf("TestProfile = %#v", result)
	}
	status := service.SecretStatus([]string{"primary", "missing"})
	if !status["primary"] || status["missing"] {
		t.Fatalf("SecretStatus = %#v", status)
	}
}

func TestProfileFailureIncludesProviderHTTPStatus(t *testing.T) {
	status := http.StatusNotFound
	result := aiTestFailure(&ai.ProviderFailure{
		Stage: ai.FailureHTTP, Class: ai.FailureNotFound, HTTPStatus: &status, Retry: ai.RetryNever,
	})
	if result.FailureClass != ai.FailureNotFound || result.HTTPStatus != http.StatusNotFound {
		t.Fatalf("TestProfile failure = %#v", result)
	}
}

func TestProfileEditDowngradesStaleEvaluation(t *testing.T) {
	app := newTestApp(t, filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	configured := evaluatedModelSettingsForTest(t, "primary", "Primary")
	_, _, err := app.MutateSettings(func(settings *Settings) error {
		settings.AI.Profiles = []AIModelSettings{configured}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	changed := configured
	changed.Model = "changed-model"
	patch, err := json.Marshal(map[string]any{"ai": map[string]any{"profiles": []AIModelSettings{changed}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := NewSettingsService(app, NewAISecrets(newFakeSecretStore())).Update(string(patch)); err != nil {
		t.Fatal(err)
	}
	current := app.Settings().AI.Profiles[0]
	if current.Evaluation != ai.EvaluationUnverified || current.EvaluationSuite != "" || !current.EvaluationReport.Empty() {
		t.Fatalf("stale evaluation was not revoked: %#v", current)
	}
}

func TestApplyAndRevokeEvaluationUsesExactCurrentArtifacts(t *testing.T) {
	app := newTestApp(t, filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	configured := modelSettingsForTest("primary", "Primary")
	_, _, err := app.MutateSettings(func(settings *Settings) error {
		settings.AI.Profiles = []AIModelSettings{configured}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	evaluated := evaluatedModelSettingsForTest(t, "primary", "Primary")
	service := NewAIService(app, NewAISecrets(newFakeSecretStore()))
	if err := service.ApplyEvaluation("primary", evaluated.EvaluationReport); err != nil {
		t.Fatal(err)
	}
	current := app.Settings().AI.Profiles[0]
	if current.Evaluation != ai.EvaluationApproved || current.EvaluationSuite == "" || current.EvaluationReport.Empty() {
		t.Fatalf("evaluation was not applied: %#v", current)
	}
	if err := service.RevokeEvaluation("primary"); err != nil {
		t.Fatal(err)
	}
	current = app.Settings().AI.Profiles[0]
	if current.Evaluation != ai.EvaluationUnverified || current.EvaluationSuite != "" || !current.EvaluationReport.Empty() {
		t.Fatalf("evaluation was not revoked: %#v", current)
	}
}
