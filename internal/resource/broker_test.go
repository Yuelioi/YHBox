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

func (a authorizer) AuthorizeOpen(context.Context, resource.OpenRequest) (resource.OpenAuthorization, error) {
	return resource.OpenAuthorization{CapabilityScope: []byte(`{}`)}, a.err
}
func (a authorizer) AuthorizeBorrow(context.Context, resource.BorrowRequest) error   { return a.err }
func (a authorizer) AuthorizeCall(context.Context, resource.AuthorizationCall) error { return a.err }

type callDenyAuthorizer struct{}

func (callDenyAuthorizer) AuthorizeOpen(context.Context, resource.OpenRequest) (resource.OpenAuthorization, error) {
	return resource.OpenAuthorization{CapabilityScope: []byte(`{}`)}, nil
}
func (callDenyAuthorizer) AuthorizeBorrow(context.Context, resource.BorrowRequest) error { return nil }
func (callDenyAuthorizer) AuthorizeCall(context.Context, resource.AuthorizationCall) error {
	return errors.New("grant revoked")
}

func testScope(run, invocation string) resource.Scope {
	return resource.Scope{
		ProgramHash:          artifact.Digest("sha256:" + strings.Repeat("1", 64)),
		CapabilityPlanDigest: artifact.Digest("sha256:" + strings.Repeat("2", 64)),
		GrantDigest:          artifact.Digest("sha256:" + strings.Repeat("3", 64)),
		PolicyGeneration:     "policy-1",
		RunID:                run, Principal: "user", PluginInstanceID: "builtin", SessionID: "session-1",
		GraphID: "main", NodeID: invocation, RequirementID: "resource", InvocationID: invocation,
	}
}

type provider struct {
	mu       sync.Mutex
	opens    int
	closes   int
	invokes  int
	lastOpen resource.ProviderOpenRequest
	closeErr error
}

type blockingOpenProvider struct {
	provider
	started            chan struct{}
	proceed            chan struct{}
	ignoreCancellation bool
}

type blockingCloseProvider struct {
	provider
	started chan struct{}
	proceed chan struct{}
}

func (p *blockingOpenProvider) Open(ctx context.Context, request resource.ProviderOpenRequest) (any, error) {
	close(p.started)
	if p.ignoreCancellation {
		<-p.proceed
	} else {
		select {
		case <-p.proceed:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return p.provider.Open(ctx, request)
}

func (p *blockingCloseProvider) Close(ctx context.Context, value any) error {
	close(p.started)
	<-p.proceed
	return p.provider.Close(ctx, value)
}

func (p *provider) Open(_ context.Context, request resource.ProviderOpenRequest) (any, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.opens++
	p.lastOpen = request
	return &struct{}{}, nil
}

func (p *provider) openedRequest() resource.ProviderOpenRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastOpen
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
	return p.closeErr
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
		Operations: []string{"read"},
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

func TestBrokerInjectsGrantedScopeIntoProviderRequest(t *testing.T) {
	p := &provider{}
	broker, err := resource.New(authorizer{}, map[string]resource.Provider{"test": p}, resource.Options{})
	if err != nil {
		t.Fatal(err)
	}
	request := resource.OpenRequest{
		Scope: testScope("run-1", "node-1"), ProviderID: "test", TargetID: "target-1", Kind: "test/session",
		Operations: []string{"read"}, Config: []byte(`{"untrusted":true}`),
	}
	if _, err := broker.Open(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	opened := p.openedRequest()
	if string(opened.CapabilityScope) != `{}` || string(opened.Config) != `{"untrusted":true}` {
		t.Fatalf("provider authority/config = %s / %s", opened.CapabilityScope, opened.Config)
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
		Operations: []string{"read", "write"},
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
	borrowed, err := broker.Borrow(context.Background(), owner, handle, borrower, []string{"read"})
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

func TestBrokerRunRevocationCleansUpResources(t *testing.T) {
	p := &provider{}
	broker, err := resource.New(authorizer{}, map[string]resource.Provider{"test": p}, resource.Options{})
	if err != nil {
		t.Fatal(err)
	}
	open := func(run, invocation string) resource.Handle {
		t.Helper()
		handle, err := broker.Open(context.Background(), resource.OpenRequest{
			Scope: testScope(run, invocation), ProviderID: "test", TargetID: "target-1",
			Kind: "test/session", Operations: []string{"read"},
		})
		if err != nil {
			t.Fatal(err)
		}
		return handle
	}
	open("run-crashed", "a")
	open("run-crashed", "b")
	other := open("run-live", "c")
	if err := broker.RevokeRun(context.Background(), "run-crashed"); err != nil {
		t.Fatal(err)
	}
	if _, _, closes := p.counts(); closes != 2 {
		t.Fatalf("run revocation total closes = %d, want 2", closes)
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
		Kind: "test/session", Operations: []string{"read"},
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

func TestBrokerCloseCancelsInflightOpenBeforeReturning(t *testing.T) {
	p := &blockingOpenProvider{started: make(chan struct{}), proceed: make(chan struct{})}
	broker, err := resource.New(authorizer{}, map[string]resource.Provider{"test": p}, resource.Options{})
	if err != nil {
		t.Fatal(err)
	}
	request := resource.OpenRequest{
		Scope: testScope("run-1", "node-1"), ProviderID: "test", TargetID: "target-1",
		Kind: "test/session", Operations: []string{"read"},
	}
	openDone := make(chan error, 1)
	go func() {
		_, err := broker.Open(context.Background(), request)
		openDone <- err
	}()
	<-p.started
	closeDone := make(chan error, 1)
	go func() { closeDone <- broker.Close(context.Background()) }()
	if err := <-openDone; !errors.Is(err, resource.ErrBrokerClosed) {
		t.Fatalf("in-flight Open after Close = %v", err)
	}
	if err := <-closeDone; err != nil {
		t.Fatal(err)
	}
	if opens, _, closes := p.counts(); opens != 0 || closes != 0 {
		t.Fatalf("canceled provider open/close = %d/%d, want 0/0", opens, closes)
	}
}

func TestBrokerRunRevocationWaitsForInflightOpenAndPermanentlyRejectsRun(t *testing.T) {
	cleanupErr := errors.New("provider cleanup failed")
	p := &blockingOpenProvider{
		provider: provider{closeErr: cleanupErr}, started: make(chan struct{}), proceed: make(chan struct{}), ignoreCancellation: true,
	}
	broker, err := resource.New(authorizer{}, map[string]resource.Provider{"test": p}, resource.Options{})
	if err != nil {
		t.Fatal(err)
	}
	request := resource.OpenRequest{
		Scope: testScope("run-revoked", "node-1"), ProviderID: "test", TargetID: "target-1",
		Kind: "test/session", Operations: []string{"read"},
	}
	openDone := make(chan error, 1)
	go func() {
		_, err := broker.Open(context.Background(), request)
		openDone <- err
	}()
	<-p.started
	revokeDone := make(chan error, 1)
	go func() { revokeDone <- broker.RevokeRun(context.Background(), "run-revoked") }()
	select {
	case err := <-revokeDone:
		t.Fatalf("RevokeRun returned before in-flight Open completed: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(p.proceed)
	if err := <-openDone; !errors.Is(err, resource.ErrRunRevoked) || !errors.Is(err, cleanupErr) {
		t.Fatalf("in-flight Open after RevokeRun = %v", err)
	}
	if err := <-revokeDone; !errors.Is(err, cleanupErr) {
		t.Fatalf("RevokeRun cleanup error = %v", err)
	}
	if opens, _, closes := p.counts(); opens != 1 || closes != 1 {
		t.Fatalf("revoked provider open/close = %d/%d, want 1/1", opens, closes)
	}
	if _, err := broker.Open(context.Background(), request); !errors.Is(err, resource.ErrRunRevoked) {
		t.Fatalf("Open after RevokeRun = %v", err)
	}
	if err := broker.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestBrokerRunRevocationContinuesAfterCallerTimeout(t *testing.T) {
	normal := &provider{}
	blocked := &blockingOpenProvider{
		started: make(chan struct{}), proceed: make(chan struct{}), ignoreCancellation: true,
	}
	broker, err := resource.New(authorizer{}, map[string]resource.Provider{
		"normal":  normal,
		"blocked": blocked,
	}, resource.Options{})
	if err != nil {
		t.Fatal(err)
	}
	scope := testScope("run-timeout", "node-normal")
	handle, err := broker.Open(context.Background(), resource.OpenRequest{
		Scope: scope, ProviderID: "normal", TargetID: "target-1", Kind: "test/session",
		Operations: []string{"read"},
	})
	if err != nil {
		t.Fatal(err)
	}
	blockedRequest := resource.OpenRequest{
		Scope: testScope("run-timeout", "node-blocked"), ProviderID: "blocked", TargetID: "target-2",
		Kind: "test/session", Operations: []string{"read"},
	}
	openDone := make(chan error, 1)
	go func() {
		_, err := broker.Open(context.Background(), blockedRequest)
		openDone <- err
	}()
	<-blocked.started

	revokeCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := broker.RevokeRun(revokeCtx, "run-timeout"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timed-out RevokeRun = %v", err)
	}
	if _, err := broker.Invoke(context.Background(), resource.Call{
		Scope: scope, Handle: handle, Operation: "read",
	}); !errors.Is(err, resource.ErrRunRevoked) {
		t.Fatalf("Invoke after timed-out RevokeRun = %v", err)
	}
	if _, err := broker.Borrow(
		context.Background(), scope, handle, testScope("run-timeout", "node-borrower"),
		[]string{"read"},
	); !errors.Is(err, resource.ErrRunRevoked) {
		t.Fatalf("Borrow after timed-out RevokeRun = %v", err)
	}

	close(blocked.proceed)
	if err := <-openDone; !errors.Is(err, resource.ErrRunRevoked) {
		t.Fatalf("in-flight Open after timed-out RevokeRun = %v", err)
	}
	if err := broker.RevokeRun(context.Background(), "run-timeout"); err != nil {
		t.Fatalf("wait for background RevokeRun = %v", err)
	}
	if opens, invokes, closes := normal.counts(); opens != 1 || invokes != 0 || closes != 1 {
		t.Fatalf("normal provider open/invoke/close = %d/%d/%d, want 1/0/1", opens, invokes, closes)
	}
	if opens, invokes, closes := blocked.counts(); opens != 1 || invokes != 0 || closes != 1 {
		t.Fatalf("blocked provider open/invoke/close = %d/%d/%d, want 1/0/1", opens, invokes, closes)
	}
	if err := broker.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestBrokerCloseWaitsForConcurrentRunRevocation(t *testing.T) {
	cleanupErr := errors.New("provider close failed")
	p := &blockingCloseProvider{
		provider: provider{closeErr: cleanupErr}, started: make(chan struct{}), proceed: make(chan struct{}),
	}
	broker, err := resource.New(authorizer{}, map[string]resource.Provider{"test": p}, resource.Options{})
	if err != nil {
		t.Fatal(err)
	}
	request := resource.OpenRequest{
		Scope: testScope("run-closing", "node-1"), ProviderID: "test", TargetID: "target-1",
		Kind: "test/session", Operations: []string{"read"},
	}
	handle, err := broker.Open(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	revokeDone := make(chan error, 1)
	go func() { revokeDone <- broker.RevokeRun(context.Background(), "run-closing") }()
	deadline := time.Now().Add(time.Second)
	for {
		_, invokeErr := broker.Invoke(context.Background(), resource.Call{
			Scope: request.Scope, Handle: handle, Operation: "read",
		})
		if errors.Is(invokeErr, resource.ErrRunRevoked) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("run revocation did not become terminal: %v", invokeErr)
		}
		time.Sleep(time.Millisecond)
	}
	closeDone := make(chan error, 1)
	go func() { closeDone <- broker.Close(context.Background()) }()
	<-p.started
	select {
	case err := <-revokeDone:
		t.Fatalf("RevokeRun returned before provider cleanup: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	select {
	case err := <-closeDone:
		t.Fatalf("Close returned before run revocation cleanup: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(p.proceed)
	if err := <-revokeDone; !errors.Is(err, cleanupErr) {
		t.Fatalf("RevokeRun cleanup error = %v", err)
	}
	if err := <-closeDone; !errors.Is(err, cleanupErr) {
		t.Fatalf("Close concurrent revocation error = %v", err)
	}
	if opens, _, closes := p.counts(); opens != 1 || closes != 1 {
		t.Fatalf("provider open/close = %d/%d, want 1/1", opens, closes)
	}
	if err := broker.RevokeRun(context.Background(), "new-run"); !errors.Is(err, resource.ErrBrokerClosed) {
		t.Fatalf("new RevokeRun after Close = %v", err)
	}
}
