package aiauthoring

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/runid"
)

func TestConversationStoreIsolatesWorkflowHistoryAndReloads(t *testing.T) {
	now := time.Date(2026, 9, 3, 5, 0, 0, 0, time.UTC)
	root := filepath.Join(t.TempDir(), "ai-conversations")
	store, err := NewConversationStore(root, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.Create("workflow-a")
	if err != nil {
		t.Fatal(err)
	}
	duplicateEmpty, err := store.Create("workflow-a")
	if err != nil || duplicateEmpty.ID != first.ID {
		t.Fatalf("duplicate empty conversation = %#v, %v", duplicateEmpty, err)
	}
	messageID, _ := runid.New()
	first, err = store.Append("workflow-a", first.ID, ConversationMessage{ID: messageID, Role: "user", Content: "修复等待模板节点"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Title != "修复等待模板节点" || len(first.Messages) != 1 {
		t.Fatalf("unexpected conversation: %#v", first)
	}
	failureID, _ := runid.New()
	first, err = store.Append("workflow-a", first.ID, ConversationMessage{
		ID: failureID, Role: "assistant", Content: "ai.authoring.tool_input_invalid",
		ProblemID: "ai.authoring.tool_input_invalid", ProblemParams: map[string]any{"tool": "workflow_connect"}, OperationID: "operation-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	next, err := store.Create("workflow-a")
	if err != nil || next.ID == first.ID {
		t.Fatalf("new conversation after first message = %#v, %v", next, err)
	}
	if _, err := store.Get("workflow-b", first.ID); err != ErrConversationNotFound {
		t.Fatalf("cross-workflow read error = %v", err)
	}
	reloaded, err := NewConversationStore(root, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	items, err := reloaded.List("workflow-a")
	foundFirst := false
	for _, item := range items {
		foundFirst = foundFirst || (item.ID == first.ID && item.MessageCount == 2)
	}
	if err != nil || len(items) != 2 || !foundFirst {
		t.Fatalf("reloaded summaries = %#v, err = %v", items, err)
	}
	persisted, err := reloaded.Get("workflow-a", first.ID)
	if err != nil || persisted.Messages[1].ProblemParams["tool"] != "workflow_connect" || persisted.Messages[1].OperationID != "operation-1" {
		t.Fatalf("reloaded failure evidence = %#v, err = %v", persisted.Messages, err)
	}
}

func TestConversationStoreAcceptsUnicodeTitleByCharacterCount(t *testing.T) {
	store, err := NewConversationStore(filepath.Join(t.TempDir(), "ai-conversations"), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	conversation, err := store.Create("workflow-unicode")
	if err != nil {
		t.Fatal(err)
	}
	messageID, _ := runid.New()
	instruction := "诊断这次 Run，解释异常或未满足的节点条件，并准备最小、可验证的修复提案。"
	conversation, err = store.Append("workflow-unicode", conversation.ID, ConversationMessage{ID: messageID, Role: "user", Content: instruction})
	if err != nil {
		t.Fatalf("append Chinese conversation title: %v", err)
	}
	if conversation.Title != instruction || len(conversation.Messages) != 1 {
		t.Fatalf("conversation = %#v", conversation)
	}
}
