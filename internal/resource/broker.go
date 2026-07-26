// Package resource owns Run-scoped access to host and plugin resources.
package resource

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/yottaapp/yotta/internal/artifact"
)

const defaultMaxPayloadBytes = 4 << 20

var (
	identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)
	operationPattern  = regexp.MustCompile(`^[a-z][a-z0-9._/-]{0,63}$`)

	ErrUnknownHandle = errors.New("unknown resource handle")
	ErrScopeMismatch = errors.New("resource handle scope mismatch")
	ErrOperation     = errors.New("resource operation is not granted")
	ErrExpired       = errors.New("resource handle expired")
	ErrBrokerClosed  = errors.New("resource broker closed")
	ErrRunRevoked    = errors.New("resource run revoked")
)

type Scope struct {
	ProgramHash          artifact.Digest `json:"programHash"`
	CapabilityPlanDigest artifact.Digest `json:"capabilityPlanDigest"`
	GrantDigest          artifact.Digest `json:"grantDigest"`
	PolicyGeneration     string          `json:"policyGeneration"`
	RunID                string          `json:"runID"`
	Principal            string          `json:"principal"`
	PluginInstanceID     string          `json:"pluginInstanceID"`
	SessionID            string          `json:"sessionID"`
	GraphID              string          `json:"graphID"`
	NodeID               string          `json:"nodeID"`
	RequirementID        string          `json:"requirementID"`
	InvocationID         string          `json:"invocationID"`
}

func (s Scope) Validate() error {
	if !s.ProgramHash.Valid() || !s.CapabilityPlanDigest.Valid() || !s.GrantDigest.Valid() ||
		!identifierPattern.MatchString(s.PolicyGeneration) ||
		!identifierPattern.MatchString(s.RunID) || !identifierPattern.MatchString(s.Principal) ||
		!identifierPattern.MatchString(s.PluginInstanceID) || !identifierPattern.MatchString(s.SessionID) ||
		!validAttributionID(s.GraphID) || !validAttributionID(s.NodeID) ||
		!identifierPattern.MatchString(s.RequirementID) ||
		!identifierPattern.MatchString(s.InvocationID) {
		return errors.New("invalid resource scope")
	}
	return nil
}

func validAttributionID(value string) bool {
	if value == "" || len(value) > 256 || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) || character == '\u061c' || character == '\u200e' || character == '\u200f' ||
			character >= '\u202a' && character <= '\u202e' || character >= '\u2066' && character <= '\u2069' {
			return false
		}
	}
	return true
}

// Handle is the only resource value allowed to cross the workflow boundary.
// Token is random authority, never a path, pointer, descriptor, or provider ID.
type Handle struct {
	Token     string    `json:"token"`
	Kind      string    `json:"kind"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (h Handle) Validate() error {
	decoded, err := base64.RawURLEncoding.DecodeString(h.Token)
	if err != nil || len(decoded) != 32 || !identifierPattern.MatchString(h.Kind) || h.ExpiresAt.IsZero() || h.ExpiresAt.Location() != time.UTC {
		return errors.New("invalid resource handle")
	}
	return nil
}

type OpenRequest struct {
	Scope      Scope
	ProviderID string
	TargetID   string
	Kind       string
	Operations []string
	ExpiresAt  time.Time
	Config     []byte
}

type Call struct {
	Scope     Scope
	Handle    Handle
	Operation string
	Payload   []byte
}

type Authorizer interface {
	AuthorizeOpen(context.Context, OpenRequest) (OpenAuthorization, error)
	AuthorizeBorrow(context.Context, BorrowRequest) error
	AuthorizeCall(context.Context, AuthorizationCall) error
}

// OpenAuthorization is produced only by the host authorizer. Broker injects
// it into ProviderOpenRequest so workflow code cannot forge granted scope or
// credential metadata through OpenRequest.Config.
type OpenAuthorization struct {
	CapabilityScope     json.RawMessage
	CredentialBindingID string
}

type ProviderOpenRequest struct {
	Scope               Scope
	ProviderID          string
	TargetID            string
	Kind                string
	Operations          []string
	ExpiresAt           time.Time
	Config              []byte
	CapabilityScope     json.RawMessage
	CredentialBindingID string
}

type BorrowRequest struct {
	Owner      Scope
	Borrower   Scope
	ProviderID string
	TargetID   string
	Kind       string
	Operations []string
	ExpiresAt  time.Time
}

type AuthorizationCall struct {
	Scope      Scope
	ProviderID string
	TargetID   string
	Kind       string
	Operation  string
}

// Provider is a host-side adapter. Its object is retained inside Broker and is
// never returned to a workflow, Wasm module, or process node.
type Provider interface {
	Open(context.Context, ProviderOpenRequest) (any, error)
	Invoke(context.Context, any, string, []byte) ([]byte, error)
	Close(context.Context, any) error
}

type Options struct {
	Now             func() time.Time
	Random          io.Reader
	MaxPayloadBytes int
}

type Broker struct {
	mu              sync.Mutex
	authorizer      Authorizer
	providers       map[string]Provider
	leases          map[string]*lease
	now             func() time.Time
	random          io.Reader
	maxPayloadBytes int
	closed          bool
	closeOnce       sync.Once
	closeDone       chan struct{}
	closeErr        error
	opening         map[*openAttempt]struct{}
	openingChanged  chan struct{}
	revokedRuns     map[string]struct{}
	revocations     map[string]*runRevocation
	openCleanupErrs map[string][]error
}

type openAttempt struct {
	runID  string
	cancel context.CancelFunc
}

type runRevocation struct {
	done     chan struct{}
	err      error
	reported bool
}

type lease struct {
	handle     Handle
	scope      Scope
	operations map[string]struct{}
	providerID string
	targetID   string
	object     *objectState
}

type objectState struct {
	mu        sync.Mutex
	cond      *sync.Cond
	provider  Provider
	value     any
	lifetime  context.Context
	cancel    context.CancelFunc
	leases    int
	active    int
	closing   bool
	closed    bool
	closeOnce sync.Once
	closeDone chan struct{}
	closeErr  error
}

func New(authorizer Authorizer, providers map[string]Provider, options Options) (*Broker, error) {
	if authorizer == nil {
		return nil, errors.New("resource authorizer is required")
	}
	providerCopy := make(map[string]Provider, len(providers))
	for id, provider := range providers {
		if !identifierPattern.MatchString(id) || provider == nil {
			return nil, fmt.Errorf("invalid resource provider %q", id)
		}
		providerCopy[id] = provider
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Random == nil {
		options.Random = rand.Reader
	}
	if options.MaxPayloadBytes == 0 {
		options.MaxPayloadBytes = defaultMaxPayloadBytes
	}
	if options.MaxPayloadBytes < 0 {
		return nil, errors.New("invalid resource payload budget")
	}
	return &Broker{
		authorizer: authorizer, providers: providerCopy, leases: map[string]*lease{},
		now: options.Now, random: options.Random, maxPayloadBytes: options.MaxPayloadBytes,
		closeDone: make(chan struct{}), opening: map[*openAttempt]struct{}{},
		openingChanged: make(chan struct{}), revokedRuns: map[string]struct{}{}, revocations: map[string]*runRevocation{},
		openCleanupErrs: map[string][]error{},
	}, nil
}

func (b *Broker) Open(ctx context.Context, request OpenRequest) (Handle, error) {
	if ctx == nil {
		return Handle{}, errors.New("resource open context is required")
	}
	if err := ctx.Err(); err != nil {
		return Handle{}, err
	}
	request = cloneOpenRequest(request)
	provider, ok := b.providers[request.ProviderID]
	if !ok {
		return Handle{}, fmt.Errorf("unknown resource provider %q", request.ProviderID)
	}
	operations, err := validateOpenRequest(request, b.now())
	if err != nil {
		return Handle{}, err
	}
	if len(request.Config) > b.maxPayloadBytes {
		return Handle{}, errors.New("resource config exceeds byte budget")
	}
	openCtx, cancelOpen := context.WithCancel(ctx)
	attempt := &openAttempt{runID: request.Scope.RunID, cancel: cancelOpen}
	b.mu.Lock()
	if terminal := b.terminalOpenErrorLocked(attempt.runID); terminal != nil {
		b.mu.Unlock()
		cancelOpen()
		return Handle{}, terminal
	}
	b.opening[attempt] = struct{}{}
	b.mu.Unlock()
	defer b.finishOpen(attempt)
	authorization, err := b.authorizer.AuthorizeOpen(openCtx, cloneOpenRequest(request))
	if err != nil {
		if terminal := b.terminalOpenError(attempt.runID); terminal != nil {
			return Handle{}, terminal
		}
		return Handle{}, fmt.Errorf("authorize resource open: %w", err)
	}
	if err := validateOpenAuthorization(authorization); err != nil {
		return Handle{}, fmt.Errorf("invalid resource authorization: %w", err)
	}
	value, err := provider.Open(openCtx, providerOpenRequest(request, authorization))
	if err != nil {
		if terminal := b.terminalOpenError(attempt.runID); terminal != nil {
			return Handle{}, terminal
		}
		return Handle{}, fmt.Errorf("open resource: %w", err)
	}
	lifetime, cancel := context.WithCancel(context.Background())
	object := &objectState{provider: provider, value: value, lifetime: lifetime, cancel: cancel, leases: 1, closeDone: make(chan struct{})}
	object.cond = sync.NewCond(&object.mu)
	b.mu.Lock()
	if terminal := b.terminalOpenErrorLocked(attempt.runID); terminal != nil {
		b.mu.Unlock()
		cleanupErr := provider.Close(context.WithoutCancel(ctx), value)
		b.recordOpenCleanupError(attempt.runID, cleanupErr)
		return Handle{}, errors.Join(terminal, cleanupErr)
	}
	token, err := b.newTokenLocked()
	if err != nil {
		b.mu.Unlock()
		cleanupErr := provider.Close(context.WithoutCancel(ctx), value)
		return Handle{}, errors.Join(err, cleanupErr)
	}
	handle := Handle{Token: token, Kind: request.Kind, ExpiresAt: request.ExpiresAt.UTC()}
	b.leases[token] = &lease{handle: handle, scope: request.Scope, operations: operations, providerID: request.ProviderID, targetID: request.TargetID, object: object}
	b.mu.Unlock()
	return handle, nil
}

func (b *Broker) finishOpen(attempt *openAttempt) {
	attempt.cancel()
	b.mu.Lock()
	delete(b.opening, attempt)
	close(b.openingChanged)
	b.openingChanged = make(chan struct{})
	b.mu.Unlock()
}

func (b *Broker) terminalOpenError(runID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.terminalOpenErrorLocked(runID)
}

func (b *Broker) terminalOpenErrorLocked(runID string) error {
	if b.closed {
		return ErrBrokerClosed
	}
	if _, revoked := b.revokedRuns[runID]; revoked {
		return ErrRunRevoked
	}
	return nil
}

func (b *Broker) recordOpenCleanupError(runID string, err error) {
	if err == nil {
		return
	}
	b.mu.Lock()
	b.openCleanupErrs[runID] = append(b.openCleanupErrs[runID], err)
	b.mu.Unlock()
}

func (b *Broker) takeOpenCleanupErrors(runID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	errs := b.openCleanupErrs[runID]
	delete(b.openCleanupErrs, runID)
	return errors.Join(errs...)
}

func (b *Broker) Borrow(ctx context.Context, owner Scope, handle Handle, borrower Scope, operations []string, expiresAt time.Time) (Handle, error) {
	if err := owner.Validate(); err != nil {
		return Handle{}, err
	}
	if err := borrower.Validate(); err != nil {
		return Handle{}, err
	}
	if owner.RunID != borrower.RunID {
		return Handle{}, ErrScopeMismatch
	}
	if owner.ProgramHash != borrower.ProgramHash || owner.CapabilityPlanDigest != borrower.CapabilityPlanDigest ||
		owner.GrantDigest != borrower.GrantDigest || owner.PolicyGeneration != borrower.PolicyGeneration ||
		owner.Principal != borrower.Principal || owner.PluginInstanceID != borrower.PluginInstanceID || owner.SessionID != borrower.SessionID {
		return Handle{}, ErrScopeMismatch
	}
	requested, err := operationSet(operations)
	if err != nil {
		return Handle{}, err
	}
	now := b.now()
	b.mu.Lock()
	if terminal := b.terminalOpenErrorLocked(owner.RunID); terminal != nil {
		b.mu.Unlock()
		return Handle{}, terminal
	}
	parent, ok := b.leases[handle.Token]
	if !ok || parent.handle != handle {
		b.mu.Unlock()
		return Handle{}, ErrUnknownHandle
	}
	if parent.scope != owner {
		b.mu.Unlock()
		return Handle{}, ErrScopeMismatch
	}
	if !now.Before(parent.handle.ExpiresAt) {
		b.mu.Unlock()
		return Handle{}, ErrExpired
	}
	if !expiresAt.After(now) || expiresAt.After(parent.handle.ExpiresAt) {
		b.mu.Unlock()
		return Handle{}, errors.New("borrow expiry must narrow the parent lease")
	}
	for operation := range requested {
		if _, ok := parent.operations[operation]; !ok {
			b.mu.Unlock()
			return Handle{}, ErrOperation
		}
	}
	borrowRequest := BorrowRequest{
		Owner: owner, Borrower: borrower, ProviderID: parent.providerID, TargetID: parent.targetID,
		Kind: handle.Kind, Operations: operationNames(requested), ExpiresAt: expiresAt.UTC(),
	}
	b.mu.Unlock()
	if err := b.authorizer.AuthorizeBorrow(ctx, borrowRequest); err != nil {
		return Handle{}, fmt.Errorf("authorize resource borrow: %w", err)
	}
	b.mu.Lock()
	if terminal := b.terminalOpenErrorLocked(owner.RunID); terminal != nil {
		b.mu.Unlock()
		return Handle{}, terminal
	}
	current, stillCurrent := b.leases[handle.Token]
	if !stillCurrent || current != parent || current.handle != handle || current.scope != owner || !b.now().Before(current.handle.ExpiresAt) {
		b.mu.Unlock()
		return Handle{}, ErrUnknownHandle
	}
	token, err := b.newTokenLocked()
	if err != nil {
		b.mu.Unlock()
		return Handle{}, err
	}
	borrowed := Handle{Token: token, Kind: handle.Kind, ExpiresAt: expiresAt.UTC()}
	parent.object.mu.Lock()
	if parent.object.closing || parent.object.closed {
		parent.object.mu.Unlock()
		b.mu.Unlock()
		return Handle{}, ErrUnknownHandle
	}
	parent.object.leases++
	parent.object.mu.Unlock()
	b.leases[token] = &lease{handle: borrowed, scope: borrower, operations: requested, providerID: parent.providerID, targetID: parent.targetID, object: parent.object}
	b.mu.Unlock()
	return borrowed, nil
}

func (b *Broker) Invoke(ctx context.Context, call Call) ([]byte, error) {
	call.Payload = append([]byte(nil), call.Payload...)
	if err := call.Scope.Validate(); err != nil {
		return nil, err
	}
	if err := call.Handle.Validate(); err != nil {
		return nil, err
	}
	if !operationPattern.MatchString(call.Operation) {
		return nil, errors.New("invalid resource operation")
	}
	if len(call.Payload) > b.maxPayloadBytes {
		return nil, errors.New("resource payload exceeds byte budget")
	}
	b.mu.Lock()
	if terminal := b.terminalOpenErrorLocked(call.Scope.RunID); terminal != nil {
		b.mu.Unlock()
		return nil, terminal
	}
	current, ok := b.leases[call.Handle.Token]
	if !ok || current.handle != call.Handle {
		b.mu.Unlock()
		return nil, ErrUnknownHandle
	}
	if current.scope != call.Scope {
		b.mu.Unlock()
		return nil, ErrScopeMismatch
	}
	if !b.now().Before(current.handle.ExpiresAt) {
		b.mu.Unlock()
		closeErr := b.release(ctx, call.Handle.Token, nil)
		return nil, errors.Join(ErrExpired, closeErr)
	}
	if _, ok := current.operations[call.Operation]; !ok {
		b.mu.Unlock()
		return nil, ErrOperation
	}
	authorization := AuthorizationCall{
		Scope: call.Scope, ProviderID: current.providerID, TargetID: current.targetID,
		Kind: current.handle.Kind, Operation: call.Operation,
	}
	b.mu.Unlock()
	if err := b.authorizer.AuthorizeCall(ctx, authorization); err != nil {
		return nil, fmt.Errorf("authorize resource call: %w", err)
	}
	b.mu.Lock()
	if terminal := b.terminalOpenErrorLocked(call.Scope.RunID); terminal != nil {
		b.mu.Unlock()
		return nil, terminal
	}
	latest, stillCurrent := b.leases[call.Handle.Token]
	if !stillCurrent || latest != current || latest.handle != call.Handle || latest.scope != call.Scope || !b.now().Before(latest.handle.ExpiresAt) {
		b.mu.Unlock()
		return nil, ErrUnknownHandle
	}
	object := current.object
	object.mu.Lock()
	if object.closing || object.closed {
		object.mu.Unlock()
		b.mu.Unlock()
		return nil, ErrUnknownHandle
	}
	object.active++
	object.mu.Unlock()
	b.mu.Unlock()

	invokeCtx, cancelInvoke := context.WithCancel(ctx)
	stopLifetimeWatch := context.AfterFunc(object.lifetime, cancelInvoke)
	result, invokeErr := object.provider.Invoke(invokeCtx, object.value, call.Operation, call.Payload)
	stopLifetimeWatch()
	cancelInvoke()
	object.mu.Lock()
	object.active--
	object.cond.Broadcast()
	object.mu.Unlock()
	if invokeErr != nil {
		return nil, fmt.Errorf("invoke resource: %w", invokeErr)
	}
	if len(result) > b.maxPayloadBytes {
		return nil, errors.New("resource result exceeds byte budget")
	}
	return append([]byte(nil), result...), nil
}

func (b *Broker) Drop(ctx context.Context, scope Scope, handle Handle) error {
	if err := scope.Validate(); err != nil {
		return err
	}
	if err := handle.Validate(); err != nil {
		return err
	}
	return b.release(ctx, handle.Token, &scope)
}

func (b *Broker) RevokeRun(ctx context.Context, runID string) error {
	if ctx == nil {
		return errors.New("resource run revocation context is required")
	}
	if !identifierPattern.MatchString(runID) {
		return errors.New("invalid run ID")
	}
	b.mu.Lock()
	revocation, exists := b.revocations[runID]
	if !exists && b.closed {
		b.mu.Unlock()
		return ErrBrokerClosed
	}
	if !exists {
		revocation = &runRevocation{done: make(chan struct{})}
		b.revocations[runID] = revocation
		b.revokedRuns[runID] = struct{}{}
		for attempt := range b.opening {
			if attempt.runID == runID {
				attempt.cancel()
			}
		}
	}
	b.mu.Unlock()
	if !exists {
		go b.finishRunRevocation(runID, revocation)
	}
	select {
	case <-revocation.done:
		b.mu.Lock()
		revocation.reported = true
		b.mu.Unlock()
		return revocation.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *Broker) finishRunRevocation(runID string, revocation *runRevocation) {
	objects := b.takeRunObjectsAfterOpensFinish(runID)
	openCleanupErr := b.takeOpenCleanupErrors(runID)
	ordered := make([]*objectState, 0, len(objects))
	for object := range objects {
		ordered = append(ordered, object)
	}
	var closeErrors []error
	for _, object := range ordered {
		object.cancel()
		if err := closeObject(context.Background(), object); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	revocation.err = errors.Join(openCleanupErr, errors.Join(closeErrors...))
	close(revocation.done)
}

func (b *Broker) takeRunObjectsAfterOpensFinish(runID string) map[*objectState]struct{} {
	for {
		b.mu.Lock()
		opening := false
		for attempt := range b.opening {
			if attempt.runID == runID {
				opening = true
				break
			}
		}
		if opening {
			changed := b.openingChanged
			b.mu.Unlock()
			<-changed
			continue
		}
		objects := make(map[*objectState]struct{})
		for token, current := range b.leases {
			if current.scope.RunID != runID {
				continue
			}
			delete(b.leases, token)
			current.object.mu.Lock()
			current.object.leases--
			if current.object.leases == 0 {
				current.object.closing = true
				objects[current.object] = struct{}{}
			}
			current.object.mu.Unlock()
		}
		b.mu.Unlock()
		return objects
	}
}

// Close permanently rejects new authority and revokes every outstanding
// lease. It is the owner shutdown path; a closed Broker cannot be reopened.
func (b *Broker) Close(ctx context.Context) error {
	b.closeOnce.Do(func() {
		b.mu.Lock()
		b.closed = true
		for attempt := range b.opening {
			attempt.cancel()
		}
		revocations := b.pendingRunRevocationsLocked()
		b.mu.Unlock()
		go func() {
			objects := b.takeObjectsAfterOpensFinish()
			openCleanupErr := b.takeCloseOwnedOpenCleanupErrors()
			var closeErrors []error
			for object := range objects {
				object.cancel()
				if err := closeObject(context.Background(), object); err != nil {
					closeErrors = append(closeErrors, err)
				}
			}
			for _, revocation := range revocations {
				<-revocation.done
				if revocation.err != nil {
					closeErrors = append(closeErrors, revocation.err)
				}
			}
			b.closeErr = errors.Join(openCleanupErr, errors.Join(closeErrors...))
			close(b.closeDone)
		}()
	})
	select {
	case <-b.closeDone:
		return b.closeErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *Broker) pendingRunRevocationsLocked() []*runRevocation {
	revocations := make([]*runRevocation, 0, len(b.revocations))
	for _, revocation := range b.revocations {
		select {
		case <-revocation.done:
			if !revocation.reported {
				revocations = append(revocations, revocation)
			}
		default:
			revocations = append(revocations, revocation)
		}
	}
	return revocations
}

func (b *Broker) takeObjectsAfterOpensFinish() map[*objectState]struct{} {
	for {
		b.mu.Lock()
		if len(b.opening) == 0 {
			objects := make(map[*objectState]struct{})
			for token, current := range b.leases {
				if _, revoking := b.revocations[current.scope.RunID]; revoking {
					continue
				}
				delete(b.leases, token)
				current.object.mu.Lock()
				current.object.leases--
				if current.object.leases == 0 {
					current.object.closing = true
					objects[current.object] = struct{}{}
				}
				current.object.mu.Unlock()
			}
			b.mu.Unlock()
			return objects
		}
		changed := b.openingChanged
		b.mu.Unlock()
		<-changed
	}
}

func (b *Broker) takeCloseOwnedOpenCleanupErrors() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	var errs []error
	for runID, runErrors := range b.openCleanupErrs {
		if _, revoking := b.revocations[runID]; revoking {
			continue
		}
		errs = append(errs, runErrors...)
		delete(b.openCleanupErrs, runID)
	}
	return errors.Join(errs...)
}

func (b *Broker) release(ctx context.Context, token string, expected *Scope) error {
	b.mu.Lock()
	current, ok := b.leases[token]
	if !ok {
		b.mu.Unlock()
		return ErrUnknownHandle
	}
	if expected != nil && current.scope != *expected {
		b.mu.Unlock()
		return ErrScopeMismatch
	}
	delete(b.leases, token)
	object := current.object
	object.mu.Lock()
	object.leases--
	last := object.leases == 0
	if last {
		object.closing = true
	}
	object.mu.Unlock()
	b.mu.Unlock()
	if !last {
		return nil
	}
	object.cancel()
	return closeObject(ctx, object)
}

// SweepExpired revokes every expired lease and closes objects whose last
// authority expired. It is the periodic cleanup path for idle resources.
func (b *Broker) SweepExpired(ctx context.Context) (int, error) {
	now := b.now()
	b.mu.Lock()
	objects := make(map[*objectState]struct{})
	revoked := 0
	for token, current := range b.leases {
		if now.Before(current.handle.ExpiresAt) {
			continue
		}
		delete(b.leases, token)
		revoked++
		current.object.mu.Lock()
		current.object.leases--
		if current.object.leases == 0 {
			current.object.closing = true
			objects[current.object] = struct{}{}
		}
		current.object.mu.Unlock()
	}
	b.mu.Unlock()
	var closeErrors []error
	for object := range objects {
		object.cancel()
		if err := closeObject(ctx, object); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	return revoked, errors.Join(closeErrors...)
}

func closeObject(ctx context.Context, object *objectState) error {
	object.closeOnce.Do(func() {
		go func() {
			object.mu.Lock()
			for object.active > 0 {
				object.cond.Wait()
			}
			object.closed = true
			object.mu.Unlock()
			if err := object.provider.Close(context.Background(), object.value); err != nil {
				object.closeErr = fmt.Errorf("close resource: %w", err)
			}
			close(object.closeDone)
		}()
	})
	select {
	case <-object.closeDone:
		return object.closeErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *Broker) newTokenLocked() (string, error) {
	for attempts := 0; attempts < 8; attempts++ {
		raw := make([]byte, 32)
		if _, err := io.ReadFull(b.random, raw); err != nil {
			return "", fmt.Errorf("generate resource token: %w", err)
		}
		token := base64.RawURLEncoding.EncodeToString(raw)
		if _, exists := b.leases[token]; !exists {
			return token, nil
		}
	}
	return "", errors.New("resource token collision budget exhausted")
}

func validateOpenRequest(request OpenRequest, now time.Time) (map[string]struct{}, error) {
	if err := request.Scope.Validate(); err != nil {
		return nil, err
	}
	if !identifierPattern.MatchString(request.ProviderID) || !identifierPattern.MatchString(request.TargetID) || !identifierPattern.MatchString(request.Kind) {
		return nil, errors.New("invalid resource provider or kind")
	}
	if !request.ExpiresAt.After(now) {
		return nil, errors.New("resource expiry must be in the future")
	}
	return operationSet(request.Operations)
}

func cloneOpenRequest(request OpenRequest) OpenRequest {
	request.Operations = append([]string(nil), request.Operations...)
	request.Config = append([]byte(nil), request.Config...)
	return request
}

func providerOpenRequest(request OpenRequest, authorization OpenAuthorization) ProviderOpenRequest {
	return ProviderOpenRequest{
		Scope: request.Scope, ProviderID: request.ProviderID, TargetID: request.TargetID, Kind: request.Kind,
		Operations: append([]string(nil), request.Operations...), ExpiresAt: request.ExpiresAt,
		Config: append([]byte(nil), request.Config...), CapabilityScope: append([]byte(nil), authorization.CapabilityScope...),
		CredentialBindingID: authorization.CredentialBindingID,
	}
}

func validateOpenAuthorization(authorization OpenAuthorization) error {
	if len(authorization.CapabilityScope) == 0 || len(authorization.CapabilityScope) > 1<<20 ||
		authorization.CredentialBindingID != "" && !identifierPattern.MatchString(authorization.CredentialBindingID) {
		return errors.New("invalid granted capability scope")
	}
	canonical, err := artifact.Canonicalize(authorization.CapabilityScope)
	if err != nil || !bytes.Equal(canonical, authorization.CapabilityScope) {
		return errors.New("granted capability scope is not canonical JSON")
	}
	return nil
}

func operationSet(operations []string) (map[string]struct{}, error) {
	if len(operations) == 0 || len(operations) > 64 {
		return nil, errors.New("resource operations must contain 1..64 entries")
	}
	result := make(map[string]struct{}, len(operations))
	for _, operation := range operations {
		if !operationPattern.MatchString(operation) {
			return nil, fmt.Errorf("invalid resource operation %q", operation)
		}
		if _, exists := result[operation]; exists {
			return nil, fmt.Errorf("duplicate resource operation %q", operation)
		}
		result[operation] = struct{}{}
	}
	return result, nil
}

func operationNames(operations map[string]struct{}) []string {
	result := make([]string, 0, len(operations))
	for operation := range operations {
		result = append(result, operation)
	}
	sort.Strings(result)
	return result
}
