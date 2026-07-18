package appbootstrap_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/appbootstrap"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/services/asset"
	"github.com/yottaapp/yotta/internal/services/workflow"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
)

type rehearsalCredentialStore struct {
	binding string
	secret  string
}

func (store rehearsalCredentialStore) Get(binding string) (string, error) {
	if binding != store.binding {
		return "", errors.New("unexpected AI credential binding")
	}
	return store.secret, nil
}

func TestCustomAIEndpointWorkflowSmoke(t *testing.T) {
	const (
		slot   = "custom"
		secret = "release-rehearsal-secret"
	)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.URL.Path != "/v1/responses" || request.Header.Get("Authorization") != "Bearer "+secret {
			t.Errorf("AI request = %s, authorization = %q", request.URL.Path, request.Header.Get("Authorization"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
			"id":"response-release","model":"release-model","status":"completed",
			"output":[{"type":"message","content":[{"type":"output_text","text":"release-ready"}]}],
			"usage":{"input_tokens":4,"output_tokens":1}
		}`))
	}))
	defer server.Close()

	installations := evaluatedAIInstallations(t, slot, server.URL+"/v1/responses", secret)
	runtime := buildReleaseRehearsalRuntime(t, installations)
	service, err := workflow.NewService(runtime.Application)
	if err != nil {
		t.Fatal(err)
	}
	source, err := service.CreateSource("Custom AI endpoint")
	if err != nil {
		t.Fatal(err)
	}
	patched, err := service.ApplyPatch(source.WorkflowID, source.Revision, []authoring.Command{
		addNode("generate", nodes.AIGenerateNodeID, 320),
		setSlot("generate", slot),
		bindValue("generate", "prompt", "confirm release"),
		connect("run-started", "started", "$generate", "in"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(patched.Source.SourceJSON, server.URL) || strings.Contains(patched.Source.SourceJSON, secret) {
		t.Fatal("Workflow Source captured AI endpoint or credential authority")
	}
	run := startAndWaitForPlatformRun(t, service, patched.Source.WorkflowID)
	assertPlatformActions(t, run, "ai.provider-response")
	if requests.Load() != 1 {
		t.Fatalf("AI endpoint request count = %d", requests.Load())
	}
}

func TestAssetCleanupProtectsWorkflowBlobRoots(t *testing.T) {
	runtime := buildReleaseRehearsalRuntime(t, emptyAIInstallations(t))
	service, err := workflow.NewService(runtime.Application)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	live, err := runtime.BlobStore.Put(ctx, "application/octet-stream", strings.NewReader("compiled workflow root"))
	if err != nil {
		t.Fatal(err)
	}
	orphan, err := runtime.BlobStore.Put(ctx, "application/octet-stream", strings.NewReader("unreferenced object"))
	if err != nil {
		t.Fatal(err)
	}
	source, err := service.CreateSource("Asset cleanup root")
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ApplyPatch(source.WorkflowID, source.Revision, []authoring.Command{
		addNode("reader", nodes.BlobToStreamNodeID, 320),
		bindBlob("reader", "blob", live),
	})
	if err != nil {
		t.Fatal(err)
	}
	assetStore, err := asset.NewStore(filepath.Join(t.TempDir(), "assets"), runtime.BlobStore)
	if err != nil {
		t.Fatal(err)
	}
	assets := asset.NewService(assetStore, nil, runtime.Application)
	preview, err := assets.PreviewCleanup()
	if err != nil || preview.LiveCount != 1 || preview.CandidateCount != 1 {
		t.Fatalf("asset cleanup preview = %#v, %v", preview, err)
	}
	result, err := assets.CommitCleanup(preview.Token)
	if err != nil || result.Reclaimed != 1 {
		t.Fatalf("asset cleanup result = %#v, %v", result, err)
	}
	if err := runtime.BlobStore.Verify(ctx, live); err != nil {
		t.Fatalf("compiled workflow root was reclaimed: %v", err)
	}
	if err := runtime.BlobStore.Verify(ctx, orphan); err == nil {
		t.Fatal("unreferenced object survived asset cleanup")
	}
}

func evaluatedAIInstallations(t *testing.T, slot, endpoint, secret string) ai.Installations {
	t.Helper()
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	suite, err := ai.BuiltinEvalSuite()
	if err != nil {
		t.Fatal(err)
	}
	draft := ai.ModelProfileDraft{
		Provider: ai.ProviderOpenAIResponses, Endpoint: endpoint, AllowLocalHTTP: true,
		Model: "release-model", Capabilities: ai.ProfileCapabilities{StructuredOutput: true},
		MaxOutputTokens: 4096, Evaluation: ai.EvaluationUnverified, ProviderMetadata: json.RawMessage(`{}`),
	}
	profile, err := ai.SealModelProfile(draft)
	if err != nil {
		t.Fatal(err)
	}
	subject, err := ai.EvaluationSubjectDigest(profile)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := ai.NewEvalCandidate(subject, builtins.AIEvaluationArtifacts())
	if err != nil {
		t.Fatal(err)
	}
	observations := make([]ai.EvalObservation, 0, len(suite.Machine().Cases))
	for _, evalCase := range suite.Machine().Cases {
		observations = append(observations, ai.EvalObservation{
			CaseID: evalCase.ID, Output: append(json.RawMessage(nil), evalCase.Expected...), Refused: evalCase.RequireRefusal,
			InputTokens: 10, OutputTokens: 5, CostMicrounits: 100, LatencyMillis: 10,
		})
	}
	evaluation, err := ai.GradeEvalSuite(suite, candidate, observations)
	if err != nil {
		t.Fatal(err)
	}
	draft.Evaluation = ai.EvaluationApproved
	draft.EvaluationSuite = suite.Digest()
	draft.EvaluationReport = evaluation.Digest
	profile, err = ai.SealModelProfile(draft)
	if err != nil {
		t.Fatal(err)
	}
	consent, err := ai.WorkflowConsentDigest(slot, profile)
	if err != nil {
		t.Fatal(err)
	}
	installations, err := ai.Install([]ai.InstallationDraft{{
		Slot: slot, Profile: draft, Evaluation: evaluation, Consent: consent,
	}}, rehearsalCredentialStore{binding: ai.CredentialBindingID(slot), secret: secret})
	if err != nil {
		t.Fatal(err)
	}
	return installations
}

func buildReleaseRehearsalRuntime(t *testing.T, aiInstallations ai.Installations) *appbootstrap.Runtime {
	t.Helper()
	runtime, err := appbootstrap.Build(appbootstrap.Config{
		DataRoot: t.TempDir(), BlobStore: testWorkflowBlobStore(t), Limits: testLimits(),
		AIInstallations: aiInstallations, HTTPInstallations: emptyHTTPInstallations(t),
		ApplicationInstallations: emptyApplicationInstallations(t), AutomationInstallations: emptyAutomationInstallations(t),
		ScriptRuntime: bootstrapScriptRuntime(t), LogEmitter: discardWorkflowLog{},
		GrantTTL: 5 * time.Minute, OwnerCloseTimeout: 5 * time.Second, Now: time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := runtime.Close(ctx); err != nil {
			t.Errorf("close release rehearsal runtime: %v", err)
		}
	})
	return runtime
}
