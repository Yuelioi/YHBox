package resource_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/resource"
)

type authorizer struct{ err error }

func (a authorizer) AuthorizeOpen(context.Context, resource.OpenRequest) error       { return a.err }
func (a authorizer) AuthorizeBorrow(context.Context, resource.BorrowRequest) error   { return a.err }
func (a authorizer) AuthorizeCall(context.Context, resource.AuthorizationCall) error { return a.err }

type callDenyAuthorizer struct{}

func (callDenyAuthorizer) AuthorizeOpen(context.Context, resource.OpenRequest) error     { return nil }
func (callDenyAuthorizer) AuthorizeBorrow(context.Context, resource.BorrowRequest) error { return nil }
func (callDenyAuthorizer) AuthorizeCall(context.Context, resource.AuthorizationCall) error {
	return errors.New("grant revoked")
}

func testScope(run, invocation string) resource.Scope {
	return resource.Scope{
		ProgramHash:          artifact.Digest("sha256:" + strings.Repeat("1", 64)),
		CapabilityPlanDigest: artifact.Digest("sha256:" + strings.Repeat("2", 64)),
		RunID:                run, Principal: "user", PluginInstanceID: "builtin", SessionID: "session-1", InvocationID: invocation,
	}
}

type provider struct {
	mu      sync.Mutex
	opens   int
	closes  int
	invokes int
}

func (p *provider) Open(context.Context, resource.OpenRequest) (any, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.opens++
	return &struct{}{}, nil
}

func (p *provider) Invoke(_ context.Context, _ any, operation string, payload []byte) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.invokes++
	return append([]byte(operation+":"), payload...), nil
}

func (p *provider) Close(context.Context, any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closes++
	return nil
}

func (p *provider) counts() (int, int, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.opens, p.invokes, p.closes
}

func TestBrokerRejectsUnauthorizedOpenBeforeProviderSideEffects(t *testing.T) {
	p := &provider{}
	broker, err := resource.New(authorizer{err: errors.New("denied")}, map[string]resource.Provider{"test": p}, resource.Options{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = broker.Open(context.Background(), resource.OpenRequest{
		Scope:      testScope("run-1", "node-1"),
		ProviderID: "test", TargetID: "target-1", Kind: "test/session", Operations: []string{"read"},
		ExpiresAt: time.Now().Add(time.Minute),
	})
	if err == nil {
		t.Fatal("unauthorized resource was opened")
	}
	if opens, _, _ := p.counts(); opens != 0 {
		t.Fatalf("provider opened %d resources before authorization", opens)
	}
}

func TestBrokerReauthorizesEveryCallBeforeProviderSideEffects(t *testing.T) {
	p := &provider{}
	broker, err := resource.New(callDenyAuthorizer{}, map[string]resource.Provider{"test": p}, resource.Options{})
	if err != nil {
		t.Fatal(err)
	}
	scope := testScope("run-1", "node-1")
	handle, err := broker.Open(context.Background(), resource.OpenRequest{
		Scope: scope, ProviderID: "test", TargetID: "target-1", Kind: "test/session",
		Operations: []string{"read"}, ExpiresAt: time.Now().Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := broker.Invoke(context.Background(), resource.Call{Scope: scope, Handle: handle, Operation: "read"}); err == nil {
		t.Fatal("revoked call grant reached the provider")
	}
	if _, invokes, _ := p.counts(); invokes != 0 {
		t.Fatalf("provider received %d denied calls", invokes)
	}
}

func TestBrokerScopesCallsBorrowAndExactlyOnceClose(t *testing.T) {
	p := &provider{}
	broker, err := resource.New(authorizer{}, map[string]resource.Provider{"test": p}, resource.Options{MaxPayloadBytes: 64})
	if err != nil {
		t.Fatal(err)
	}
	owner := testScope("run-1", "owner")
	handle, err := broker.Open(context.Background(), resource.OpenRequest{
		Scope: owner, ProviderID: "test", TargetID: "target-1", Kind: "test/session",
		Operations: []string{"read", "write"}, ExpiresAt: time.Now().Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(handle.Token, "test") || len(handle.Token) < 40 {
		t.Fatalf("token is not opaque high-entropy data: %q", handle.Token)
	}
	if _, err := broker.Invoke(context.Background(), resource.Call{
		Scope: testScope("run-2", "owner"), Handle: handle,
		Operation: "read",
	}); err == nil {
		t.Fatal("cross-run resource call succeeded")
	}
	if _, err := broker.Invoke(context.Background(), resource.Call{
		Scope: testScope("run-1", "intruder"), Handle: handle,
		Operation: "read",
	}); err == nil {
		t.Fatal("cross-invocation resource call succeeded without borrow")
	}

	borrower := testScope("run-1", "borrower")
	borrowed, err := broker.Borrow(context.Background(), owner, handle, borrower, []string{"read"}, time.Now().Add(30*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := broker.Invoke(context.Background(), resource.Call{Scope: borrower, Handle: borrowed, Operation: "write"}); err == nil {
		t.Fatal("borrow widened its operation set")
	}
	result, err := broker.Invoke(context.Background(), resource.Call{Scope: borrower, Handle: borrowed, Operation: "read", Payload: []byte("x")})
	if err != nil || string(result) != "read:x" {
		t.Fatalf("borrowed invoke = %q, %v", result, err)
	}
	if err := broker.Drop(context.Background(), owner, handle); err != nil {
		t.Fatal(err)
	}
	if _, _, closes := p.counts(); closes != 0 {
		t.Fatalf("owner drop closed a still-borrowed resource %d times", closes)
	}
	if err := broker.Drop(context.Background(), borrower, borrowed); err != nil {
		t.Fatal(err)
	}
	if _, _, closes := p.counts(); closes != 1 {
		t.Fatalf("last drop closed resource %d times, want 1", closes)
	}
	if _, err := broker.Invoke(context.Background(), resource.Call{Scope: borrower, Handle: borrowed, Operation: "read"}); err == nil {
		t.Fatal("dropped handle remained callable")
	}
}

func TestBrokerExpiryAndRunRevocationCleanUpResources(t *testing.T) {
	now := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	p := &provider{}
	broker, err := resource.New(authorizer{}, map[string]resource.Provider{"test": p}, resource.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	open := func(run, invocation string, expiry time.Time) resource.Handle {
		t.Helper()
		handle, err := broker.Open(context.Background(), resource.OpenRequest{
			Scope: testScope(run, invocation), ProviderID: "test", TargetID: "target-1",
			Kind: "test/session", Operations: []string{"read"}, ExpiresAt: expiry,
		})
		if err != nil {
			t.Fatal(err)
		}
		return handle
	}
	expired := open("run-expired", "node", now.Add(time.Second))
	now = now.Add(2 * time.Second)
	revoked, err := broker.SweepExpired(context.Background())
	if err != nil || revoked != 1 {
		t.Fatalf("SweepExpired = %d, %v", revoked, err)
	}
	if _, err := broker.Invoke(context.Background(), resource.Call{
		Scope: testScope("run-expired", "node"), Handle: expired, Operation: "read",
	}); err == nil {
		t.Fatal("expired handle remained callable")
	}
	if _, _, closes := p.counts(); closes != 1 {
		t.Fatalf("expiry closed %d resources, want 1", closes)
	}

	open("run-crashed", "a", now.Add(time.Minute))
	open("run-crashed", "b", now.Add(time.Minute))
	other := open("run-live", "c", now.Add(time.Minute))
	if err := broker.RevokeRun(context.Background(), "run-crashed"); err != nil {
		t.Fatal(err)
	}
	if _, _, closes := p.counts(); closes != 3 {
		t.Fatalf("run revocation total closes = %d, want 3", closes)
	}
	if _, err := broker.Invoke(context.Background(), resource.Call{
		Scope: testScope("run-live", "c"), Handle: other, Operation: "read",
	}); err != nil {
		t.Fatalf("revoking one run affected another: %v", err)
	}
}

func TestBrokerCloseRevokesAllAuthorityAndIsPermanent(t *testing.T) {
	p := &provider{}
	broker, err := resource.New(authorizer{}, map[string]resource.Provider{"test": p}, resource.Options{})
	if err != nil {
		t.Fatal(err)
	}
	request := resource.OpenRequest{
		Scope: testScope("run-1", "node-1"), ProviderID: "test", TargetID: "target-1",
		Kind: "test/session", Operations: []string{"read"}, ExpiresAt: time.Now().Add(time.Minute),
	}
	handle, err := broker.Open(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if err := broker.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, _, closes := p.counts(); closes != 1 {
		t.Fatalf("broker close closed %d objects, want 1", closes)
	}
	if _, err := broker.Invoke(context.Background(), resource.Call{Scope: request.Scope, Handle: handle, Operation: "read"}); !errors.Is(err, resource.ErrBrokerClosed) {
		t.Fatalf("Invoke after Close = %v", err)
	}
	if _, err := broker.Open(context.Background(), request); !errors.Is(err, resource.ErrBrokerClosed) {
		t.Fatalf("Open after Close = %v", err)
	}
	if err := broker.Close(context.Background()); err != nil {
		t.Fatalf("second Close = %v", err)
	}
}
