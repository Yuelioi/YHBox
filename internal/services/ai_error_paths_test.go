package services

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/aiauthoring"
	"github.com/yottaapp/yotta/internal/apperr"
)

func TestAIServiceStableUnavailablePaths(t *testing.T) {
	service := &AIService{}
	tests := []struct {
		call func() error
		id   string
	}{
		{func() error { _, err := service.AcceptWorkflowProposal("review"); return err }, "ai.authoring.unavailable"},
		{func() error { _, err := service.RejectWorkflowProposal("review"); return err }, "ai.authoring.unavailable"},
		{func() error { _, err := service.GetWorkflowProposal("review"); return err }, "ai.authoring.unavailable"},
		{func() error { return service.SetAPIKey("slot", "secret") }, "ai.credential.unavailable"},
		{func() error { return service.DeleteAPIKey("slot") }, "ai.credential.unavailable"},
		{func() error { return service.ApplyEvaluation("slot", ai.EvalReportArtifact{}) }, "ai.evaluation.unavailable"},
		{func() error { return service.RevokeEvaluation("slot") }, "ai.evaluation.unavailable"},
	}
	for _, test := range tests {
		if got := apperr.From(test.call()); got.ID != test.id || got.OperationID == "" {
			t.Fatalf("problem = %#v, want %s", got, test.id)
		}
	}
	if _, err := service.ProposeWorkflow("slot", "workflow", 0, "do it", ""); err == nil {
		t.Fatal("proposal without authoring succeeded")
	}
	if _, err := service.ListWorkflowAIConversations("workflow"); err == nil {
		t.Fatal("conversation list without authoring succeeded")
	}
	if _, err := service.CreateWorkflowAIConversation("workflow"); err == nil {
		t.Fatal("conversation create without authoring succeeded")
	}
	if _, err := service.GetWorkflowAIConversation("workflow", "conversation"); err == nil {
		t.Fatal("conversation get without authoring succeeded")
	}
	if err := service.DeleteWorkflowAIConversation("workflow", "conversation"); err == nil {
		t.Fatal("conversation delete without authoring succeeded")
	}
	if _, err := service.SendWorkflowAIMessage("slot", "workflow", "conversation", 0, "do it", ""); err == nil {
		t.Fatal("conversation send without authoring succeeded")
	}
	if params := envelopeParams(apperr.Envelope{Params: map[string]any{"safe": true}}); params["safe"] != true {
		t.Fatalf("params = %#v", params)
	}
}

func TestAIAuthoringSentinelProjection(t *testing.T) {
	tests := []struct {
		cause error
		id    string
	}{
		{errors.New("private"), "ai.authoring.failed"},
		{apperr.New("existing", nil), "existing"},
		{aiauthoring.ErrConversationNotFound, "ai.authoring.conversation_not_found"},
		{aiauthoring.ErrConversationCapacity, "ai.authoring.conversation_capacity"},
		{aiauthoring.ErrDiagnosticRunUnavailable, "ai.authoring.run_unavailable"},
		{aiauthoring.ErrDiagnosticRunWorkflowMismatch, "ai.authoring.run_workflow_mismatch"},
	}
	for _, test := range tests {
		if got := apperr.From(projectAIAuthoringError(test.cause)); got.ID != test.id {
			t.Fatalf("problem = %#v, want %s", got, test.id)
		}
	}
	if projectAIAuthoringError(nil) != nil {
		t.Fatal("nil authoring error was projected")
	}
}
