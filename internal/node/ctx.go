// internal/node/ctx.go
package node

import (
	"context"
	"time"
)

type ctxImpl struct {
	ctx    context.Context
	vision VisionService
	log    LogService
	spec   *Spec
}

func newCtx(ctx context.Context, vision VisionService, log LogService, spec *Spec) *ctxImpl {
	return &ctxImpl{ctx: ctx, vision: vision, log: log, spec: spec}
}

func (c *ctxImpl) Context() context.Context { return c.ctx }
func (c *ctxImpl) Vision() VisionService    { return c.vision }
func (c *ctxImpl) Log() LogService          { return c.log }
func (c *ctxImpl) Now() time.Time           { return time.Now() }

func (c *ctxImpl) Out(exitName string) OutBuilder {
	validateExitName(c.spec, exitName)
	return &outBuilderImpl{
		exitName: exitName,
		spec:     c.spec,
		data:     map[string]any{},
	}
}
