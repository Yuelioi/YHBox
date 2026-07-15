package run

import (
	"context"
	"errors"
	"sync"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/resource"
)

// Owner is the lifecycle owner for one admitted Run's ephemeral authority.
// It deliberately does not own durable RunRecord persistence.
type Owner struct {
	ctx         context.Context
	cancel      context.CancelFunc
	authorizer  *GrantAuthorizer
	broker      *resource.Broker
	runID       string
	closeOnce   sync.Once
	closeDone   chan struct{}
	closeErr    error
	taskMu      sync.Mutex
	taskCount   int
	taskErrors  []error
	taskChanged chan struct{}
	closing     bool
}

type Session struct {
	owner *Owner
	scope resource.Scope
	entry capability.GrantEntry
}

type InstalledProvider struct {
	ArtifactDigest artifact.Digest
	ABI            string
	Provider       resource.Provider
}

func NewOwner(parent context.Context, grant capability.RunGrant, providers map[string]InstalledProvider, options resource.Options) (*Owner, error) {
	if parent == nil {
		return nil, errors.New("run parent context is required")
	}
	authorizer, err := NewGrantAuthorizer(grant, options.Now)
	if err != nil {
		return nil, err
	}
	brokerProviders := make(map[string]resource.Provider, len(providers))
	for _, entry := range grant.Entries() {
		installed, ok := providers[entry.Binding.ProviderID]
		if !ok || installed.Provider == nil || installed.ArtifactDigest != entry.Binding.ProviderArtifactDigest || installed.ABI != entry.Binding.ProviderABI {
			return nil, ErrGrantDenied
		}
		brokerProviders[entry.Binding.ProviderID] = installed.Provider
	}
	broker, err := resource.New(authorizer, brokerProviders, options)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(parent)
	return &Owner{
		ctx: ctx, cancel: cancel, authorizer: authorizer, broker: broker,
		runID: grant.RunID(), closeDone: make(chan struct{}), taskChanged: make(chan struct{}),
	}, nil
}

func (o *Owner) Context() context.Context { return o.ctx }

func (o *Owner) ValidateProgram(programHash, planDigest artifact.Digest) error {
	if o.ctx.Err() != nil || !programHash.Valid() || !planDigest.Valid() || o.authorizer.grant.ProgramHash() != programHash || o.authorizer.grant.PlanDigest() != planDigest {
		return ErrGrantDenied
	}
	return nil
}

func (o *Owner) ValidateAdmission(admission Admission) error {
	if o == nil || o.ctx.Err() != nil {
		return ErrGrantDenied
	}
	grant := o.authorizer.grant
	if admission.RunID != grant.RunID() || admission.ProgramHash != grant.ProgramHash() ||
		admission.CapabilityPlanDigest != grant.PlanDigest() || admission.GrantDigest != grant.Digest() ||
		admission.PolicyGeneration != grant.PolicyGeneration() || admission.Principal != grant.Principal() {
		return ErrGrantDenied
	}
	return nil
}

func (o *Owner) Session(graphID, nodeID, requirementID, invocationID string) (*Session, error) {
	if o.ctx.Err() != nil {
		return nil, ErrGrantDenied
	}
	scope, entry, err := o.authorizer.session(graphID, nodeID, requirementID, invocationID)
	if err != nil {
		return nil, err
	}
	return &Session{owner: o, scope: scope, entry: entry}, nil
}

func (s *Session) Open(ctx context.Context, operations []string, config []byte) (resource.Handle, error) {
	callCtx, cancel, err := s.owner.callContext(ctx)
	if err != nil {
		return resource.Handle{}, err
	}
	defer cancel()
	binding := s.entry.Binding
	return s.owner.broker.Open(callCtx, resource.OpenRequest{
		Scope: s.scope, ProviderID: binding.ProviderID, TargetID: binding.TargetID, Kind: binding.ResourceKind,
		Operations: append([]string(nil), operations...), ExpiresAt: s.owner.authorizer.grant.ExpiresAt(), Config: append([]byte(nil), config...),
	})
}

func (s *Session) Invoke(ctx context.Context, handle resource.Handle, operation string, payload []byte) ([]byte, error) {
	callCtx, cancel, err := s.owner.callContext(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()
	return s.owner.broker.Invoke(callCtx, resource.Call{Scope: s.scope, Handle: handle, Operation: operation, Payload: append([]byte(nil), payload...)})
}

func (s *Session) Drop(ctx context.Context, handle resource.Handle) error {
	callCtx, cancel, err := s.owner.callContext(ctx)
	if err != nil {
		return err
	}
	defer cancel()
	return s.owner.broker.Drop(callCtx, s.scope, handle)
}

func (o *Owner) Borrow(ctx context.Context, lender *Session, handle resource.Handle, borrower *Session, operations []string) (resource.Handle, error) {
	if lender == nil || borrower == nil || lender.owner != o || borrower.owner != o {
		return resource.Handle{}, ErrGrantDenied
	}
	callCtx, cancel, err := o.callContext(ctx)
	if err != nil {
		return resource.Handle{}, err
	}
	defer cancel()
	return o.broker.Borrow(callCtx, lender.scope, handle, borrower.scope, append([]string(nil), operations...), handle.ExpiresAt)
}

func (o *Owner) Go(task func(context.Context) error) error {
	if task == nil {
		return errors.New("run task is required")
	}
	o.taskMu.Lock()
	if err := o.ctx.Err(); err != nil {
		o.taskMu.Unlock()
		return err
	}
	if o.closing {
		o.taskMu.Unlock()
		return errors.New("run owner is closing")
	}
	o.taskCount++
	o.signalTaskChangeLocked()
	o.taskMu.Unlock()
	go func() {
		err := task(o.ctx)
		o.taskMu.Lock()
		cancelledByOwner := err != nil && errors.Is(err, context.Canceled) && o.ctx.Err() != nil
		if err != nil && !cancelledByOwner {
			o.taskErrors = append(o.taskErrors, err)
		}
		o.taskCount--
		o.signalTaskChangeLocked()
		o.taskMu.Unlock()
		if err != nil && !cancelledByOwner {
			o.cancel()
		}
	}()
	return nil
}

func (o *Owner) Wait(ctx context.Context) error {
	for {
		o.taskMu.Lock()
		if o.taskCount == 0 {
			err := errors.Join(o.taskErrors...)
			o.taskMu.Unlock()
			return err
		}
		changed := o.taskChanged
		o.taskMu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (o *Owner) signalTaskChangeLocked() {
	close(o.taskChanged)
	o.taskChanged = make(chan struct{})
}

func (o *Owner) callContext(ctx context.Context) (context.Context, context.CancelFunc, error) {
	if ctx == nil {
		return nil, nil, errors.New("resource call context is required")
	}
	if err := o.ctx.Err(); err != nil {
		return nil, nil, ErrGrantDenied
	}
	callCtx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(o.ctx, cancel)
	return callCtx, func() { stop(); cancel() }, nil
}

func (o *Owner) Close(ctx context.Context) error {
	o.closeOnce.Do(func() {
		o.taskMu.Lock()
		o.closing = true
		o.signalTaskChangeLocked()
		o.taskMu.Unlock()
		go func() {
			o.authorizer.Revoke()
			o.cancel()
			revokeErr := o.broker.RevokeRun(context.Background(), o.runID)
			taskErr := o.Wait(context.Background())
			closeErr := o.broker.Close(context.Background())
			o.closeErr = errors.Join(taskErr, revokeErr, closeErr)
			close(o.closeDone)
		}()
	})
	select {
	case <-o.closeDone:
		return o.closeErr
	case <-ctx.Done():
		return ctx.Err()
	}
}
