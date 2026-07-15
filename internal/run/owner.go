package run

import (
	"context"
	"errors"
	"sync"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/resource"
)

// Owner is the lifecycle owner for one admitted Run's ephemeral authority.
// It deliberately does not own durable RunRecord persistence.
type Owner struct {
	ctx        context.Context
	cancel     context.CancelFunc
	authorizer *GrantAuthorizer
	broker     *resource.Broker
	runID      string
	closeOnce  sync.Once
	closeDone  chan struct{}
	closeErr   error
}

func NewOwner(parent context.Context, grant capability.RunGrant, providers map[string]resource.Provider, options resource.Options) (*Owner, error) {
	if parent == nil {
		return nil, errors.New("run parent context is required")
	}
	authorizer, err := NewGrantAuthorizer(grant, options.Now)
	if err != nil {
		return nil, err
	}
	broker, err := resource.New(authorizer, providers, options)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(parent)
	return &Owner{
		ctx: ctx, cancel: cancel, authorizer: authorizer, broker: broker,
		runID: grant.RunID(), closeDone: make(chan struct{}),
	}, nil
}

func (o *Owner) Context() context.Context { return o.ctx }

func (o *Owner) Broker() *resource.Broker { return o.broker }

func (o *Owner) Scope(graphID, nodeID, requirementID, invocationID string) (resource.Scope, error) {
	return o.authorizer.Scope(graphID, nodeID, requirementID, invocationID)
}

func (o *Owner) Close(ctx context.Context) error {
	o.closeOnce.Do(func() {
		go func() {
			o.authorizer.Revoke()
			o.cancel()
			revokeErr := o.broker.RevokeRun(context.Background(), o.runID)
			closeErr := o.broker.Close(context.Background())
			o.closeErr = errors.Join(revokeErr, closeErr)
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
