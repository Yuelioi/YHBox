package services

import (
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
		return ai.NewNativeProvider(profile, ai.HTTPOptions{Endpoint: server.URL + "/v1/responses"})
	})
	result := service.TestProfile(TestProfileRequest{Profile: modelSettingsForTest("primary", "Primary")})
	if !result.Ok || result.Provider != ai.ProviderOpenAIResponses || result.ResolvedModel != "gpt-resolved" || result.Finish != ai.FinishCompleted {
		t.Fatalf("TestProfile = %#v", result)
	}
	status := service.SecretStatus([]string{"primary", "missing"})
	if !status["primary"] || status["missing"] {
		t.Fatalf("SecretStatus = %#v", status)
	}
}

func TestWorkflowConsentIsExplicitAndProfileEditsRevokeIt(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	store := newFakeSecretStore()
	secrets := NewAISecrets(store)
	_, _, err := app.MutateSettings(func(settings *Settings) error {
		settings.AI.Profiles = []AIModelSettings{modelSettingsForTest("primary", "Primary")}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	service := NewAIService(app, secrets)
	consent, err := service.GrantWorkflowUse("primary")
	if err != nil || consent == "" || app.Settings().AI.Profiles[0].WorkflowConsent.String() != consent {
		t.Fatalf("GrantWorkflowUse = %q, %v", consent, err)
	}
	settingsService := NewSettingsService(app, secrets)
	if err := settingsService.Update(`{"ai":{"profiles":[{"slot":"primary","label":"Primary","provider":"openai-responses","model":"gpt-changed","maxOutputTokens":4096,"capabilities":{"structuredOutput":true,"toolCalling":false,"parallelTools":false,"background":false,"zeroRetention":false},"evaluation":"unverified","workflowConsent":"` + consent + `"}]}}`); err != nil {
		t.Fatal(err)
	}
	if app.Settings().AI.Profiles[0].WorkflowConsent != "" {
		t.Fatal("semantic profile edit retained prior workflow consent")
	}
}
