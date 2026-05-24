// internal/node/ctx.go
package node

import (
	"context"
	"fmt"
	"time"
)

type ctxImpl struct {
	ctx    context.Context
	vision VisionService
	log    LogService
	spec   *Spec
	fired  bool // Run 内首次 Fire 后置 true; 后续任何 builder Fire → panic
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
		ctx:      c,
	}
}

// markFired ctx 级 fire 守卫. 节点 Run 内任何 builder 第二次 Fire → panic.
func (c *ctxImpl) markFired(exitName string) {
	if c.fired {
		panic(fmt.Sprintf("multiple Fire() in single Run (node %q, exit %q)", c.spec.Kind, exitName))
	}
	c.fired = true
}
