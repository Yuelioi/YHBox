package run_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/resource"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/stream"
)

func TestOwnerCancelsActiveResourcesAndPermanentlyClosesBroker(t *testing.T) {
	now := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	definition := streamCapability(t)
	plan := streamPlan(t, definition)
	grant, err := capability.SealRunGrant(capability.GrantRequest{
		ProgramHash: digest("program"), Plan: plan, RunID: testRunID, Principal: "user-1", PolicyGeneration: "policy-1",
		IssuedAt: now, ExpiresAt: now.Add(time.Minute), Bindings: []capability.Binding{{
			GraphID: "main", NodeID: "producer", RequirementID: "stream", ProviderID: stream.ProviderID, TargetID: "memory",
			TargetKind: "stream-session", ResourceKind: stream.Kind, PluginInstanceID: "builtin", SessionID: "session-1",
		}},
	}, catalog{definition.Ref().CapabilityID: definition})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := stream.NewProvider(stream.Limits{MaxCapacity: 2, MaxChunkBytes: 32})
	if err != nil {
		t.Fatal(err)
	}
	owner, err := run31.NewOwner(context.Background(), grant, map[string]resource.Provider{stream.ProviderID: provider}, resource.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	scope, err := owner.Scope("main", "producer", "stream", "invoke-1")
	if err != nil {
		t.Fatal(err)
	}
	config, _ := json.Marshal(stream.Config{Capacity: 1, MaxChunkBytes: 16})
	handle, err := owner.Broker().Open(owner.Context(), resource.OpenRequest{
		Scope: scope, ProviderID: stream.ProviderID, TargetID: "memory", Kind: stream.Kind,
		Operations: []string{stream.OperationSend, stream.OperationCancel}, ExpiresAt: now.Add(30 * time.Second), Config: config,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Broker().Invoke(owner.Context(), resource.Call{Scope: scope, Handle: handle, Operation: stream.OperationSend, Payload: []byte("one")}); err != nil {
		t.Fatal(err)
	}
	blocked := make(chan error, 1)
	go func() {
		_, err := owner.Broker().Invoke(owner.Context(), resource.Call{Scope: scope, Handle: handle, Operation: stream.OperationSend, Payload: []byte("two")})
		blocked <- err
	}()
	select {
	case err := <-blocked:
		t.Fatalf("send did not block before owner close: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if err := owner.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-blocked:
		if err == nil {
			t.Fatal("active send succeeded after Run close")
		}
	case <-time.After(time.Second):
		t.Fatal("Run close did not cancel active resource call")
	}
	if _, err := owner.Scope("main", "producer", "stream", "invoke-2"); !errors.Is(err, run31.ErrGrantDenied) {
		t.Fatalf("Scope after Close = %v", err)
	}
	if _, err := owner.Broker().Open(context.Background(), resource.OpenRequest{}); !errors.Is(err, resource.ErrBrokerClosed) {
		t.Fatalf("Broker Open after Close = %v", err)
	}
}
