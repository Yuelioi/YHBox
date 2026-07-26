package run_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/resource"
	run "github.com/yottaapp/yotta/internal/run"
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
			ProviderArtifactDigest: streamProviderDigest(t), ProviderABI: stream.ProviderABI,
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
	owner, err := run.NewOwner(context.Background(), grant, map[string]run.InstalledProvider{stream.ProviderID: {
		ArtifactDigest: streamProviderDigest(t), ABI: stream.ProviderABI, Provider: provider,
	}}, resource.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	session, err := owner.Session("main", "producer", "stream", "invoke-1")
	if err != nil {
		t.Fatal(err)
	}
	config, _ := json.Marshal(stream.Config{Capacity: 1, MaxChunkBytes: 16})
	handle, err := session.Open(owner.Context(), []string{stream.OperationSend, stream.OperationCancel}, config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.Invoke(owner.Context(), handle, stream.OperationSend, []byte("one")); err != nil {
		t.Fatal(err)
	}
	blocked := make(chan error, 1)
	go func() {
		_, err := session.Invoke(owner.Context(), handle, stream.OperationSend, []byte("two"))
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
	if _, err := owner.Session("main", "producer", "stream", "invoke-2"); !errors.Is(err, run.ErrGrantDenied) {
		t.Fatalf("Session after Close = %v", err)
	}
	if _, err := session.Open(context.Background(), []string{stream.OperationSend}, config); !errors.Is(err, run.ErrGrantDenied) {
		t.Fatalf("Session Open after Close = %v", err)
	}
}

func TestOwnerOwnsAndCancelsBackgroundTasks(t *testing.T) {
	owner, _, _ := ownerForTest(t)
	started := make(chan struct{})
	if err := owner.Go(func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := owner.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := owner.Go(func(context.Context) error { return nil }); err == nil {
		t.Fatal("closed Owner accepted a background task")
	}
}

func TestCanceledOwnerRejectsNewSessionsAndTasks(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	owner, _, _ := ownerForParent(t, parent)
	cancel()
	<-owner.Context().Done()
	if _, err := owner.Session("main", "producer", "stream", "invoke-canceled"); !errors.Is(err, run.ErrGrantDenied) {
		t.Fatalf("Session after parent cancellation = %v", err)
	}
	if err := owner.Go(func(context.Context) error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("Go after parent cancellation = %v", err)
	}
	if err := owner.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestOwnerRejectsAProviderImplementationThatDoesNotMatchTheGrant(t *testing.T) {
	owner, now, grant := ownerForTest(t)
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	provider, err := stream.NewProvider(stream.Limits{MaxCapacity: 2, MaxChunkBytes: 32})
	if err != nil {
		t.Fatal(err)
	}
	_, err = run.NewOwner(context.Background(), grant, map[string]run.InstalledProvider{stream.ProviderID: {
		ArtifactDigest: digest("different provider"), ABI: stream.ProviderABI, Provider: provider,
	}}, resource.Options{Now: func() time.Time { return now }})
	if !errors.Is(err, run.ErrGrantDenied) {
		t.Fatalf("mismatched provider implementation = %v", err)
	}
}

func ownerForTest(t *testing.T) (*run.Owner, time.Time, capability.RunGrant) {
	t.Helper()
	return ownerForParent(t, context.Background())
}

func ownerForParent(t *testing.T, parent context.Context) (*run.Owner, time.Time, capability.RunGrant) {
	t.Helper()
	now := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	definition := streamCapability(t)
	plan := streamPlan(t, definition)
	grant, err := capability.SealRunGrant(capability.GrantRequest{
		ProgramHash: digest("program"), Plan: plan, RunID: testRunID, Principal: "user-1", PolicyGeneration: "policy-1",
		IssuedAt: now, ExpiresAt: now.Add(time.Minute), Bindings: []capability.Binding{{
			GraphID: "main", NodeID: "producer", RequirementID: "stream", ProviderID: stream.ProviderID, TargetID: "memory",
			ProviderArtifactDigest: streamProviderDigest(t), ProviderABI: stream.ProviderABI,
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
	owner, err := run.NewOwner(parent, grant, map[string]run.InstalledProvider{stream.ProviderID: {
		ArtifactDigest: streamProviderDigest(t), ABI: stream.ProviderABI, Provider: provider,
	}}, resource.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	return owner, now, grant
}
