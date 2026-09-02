// Command ai-authoring-smoke exercises the complete storage-backed AI conversation path.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/yottaapp/yotta/internal/aiauthoring"
	"github.com/yottaapp/yotta/internal/localruntime"
	"github.com/yottaapp/yotta/internal/noderuntime"
	"github.com/yottaapp/yotta/internal/securestore"
	"github.com/yottaapp/yotta/internal/services"
	storagemigrate "github.com/yottaapp/yotta/internal/storage/migrate"
)

func main() {
	root := flag.String("root", "", "Yotta storage profile root")
	workflowID := flag.String("workflow", "", "workflow ID")
	runID := flag.String("run", "", "Run ID")
	slot := flag.String("slot", "", "AI profile slot")
	instruction := flag.String("instruction", "诊断这次 Run，解释异常或未满足的节点条件，并准备最小、可验证的修复提案。", "conversation instruction")
	flag.Parse()
	if *root == "" || *workflowID == "" || *slot == "" {
		fmt.Fprintln(os.Stderr, "root, workflow and slot are required")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if _, err := storagemigrate.Ensure(ctx, storagemigrate.Options{Root: *root, MaxRuns: 65536}); err != nil {
		fail(err)
	}
	runtime, err := localruntime.Open(ctx, localruntime.Config{
		StorageRoot: *root, Executable: filepath.Join(filepath.Dir(os.Args[0]), "Yotta.exe"),
		RootLog: zerolog.New(io.Discard), AISecrets: services.NewAISecrets(securestore.New()),
		WorkflowLog: discardLog{}, Now: time.Now,
	})
	if err != nil {
		fail(err)
	}
	defer func() { _ = runtime.Close(context.Background()) }()
	if err := runtime.Workflow.Application.Start(ctx); err != nil {
		fail(err)
	}
	manager, err := aiauthoring.NewManager(runtime.Workflow.Application, runtime.Workflow.Builtins, time.Now)
	if err != nil {
		fail(err)
	}
	store, err := aiauthoring.NewConversationStore(filepath.Join(runtime.Roots.Data, "ai-conversations"), time.Now)
	if err != nil {
		fail(err)
	}
	if err := manager.AttachConversationStore(store); err != nil {
		fail(err)
	}
	_ = runtime.Settings.AttachEmitter(func(name string, data any) {
		if name == "ai:conversation-progress" {
			fmt.Fprintf(os.Stderr, "progress=%+v\n", data)
		}
	})
	conversation, err := manager.CreateConversation(*workflowID)
	if err != nil {
		fail(err)
	}
	source, err := runtime.Workflow.Application.GetSource(*workflowID)
	if err != nil {
		fail(err)
	}
	service := services.NewAIService(runtime.Settings, services.NewAISecrets(securestore.New()), manager)
	result, err := service.SendWorkflowAIMessage(*slot, *workflowID, conversation.ID, source.Revision(), *instruction, *runID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem=%+v\n", err)
		if latest, readErr := manager.GetConversation(*workflowID, conversation.ID); readErr == nil {
			fmt.Fprintf(os.Stderr, "conversation=%+v\n", latest)
		}
		os.Exit(1)
	}
	fmt.Printf("conversation=%s messages=%d last=%q\n", result.ID, len(result.Messages), result.Messages[len(result.Messages)-1].Content)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

type discardLog struct{}

func (discardLog) EmitWorkflowLog(context.Context, noderuntime.LogEntry) error { return nil }
