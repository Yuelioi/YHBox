// internal/node/ctx.go
package node

import (
	"context"
	"fmt"
	"time"
)

type ctxImpl struct {
	ctx      context.Context
	services ServiceBundle
	spec     *Spec
	fired    bool // Run 内首次 Fire 后置 true; 后续任何 builder Fire → panic
}

func newCtx(ctx context.Context, services ServiceBundle, spec *Spec) *ctxImpl {
	return &ctxImpl{ctx: ctx, services: services, spec: spec}
}

func (c *ctxImpl) Context() context.Context     { return c.ctx }
func (c *ctxImpl) Now() time.Time               { return time.Now() }
func (c *ctxImpl) Vision() VisionService        { return c.services.Vision }
func (c *ctxImpl) Log() LogService              { return c.services.Log }
func (c *ctxImpl) Input() InputService          { return c.services.Input }
func (c *ctxImpl) Vars() VarStore               { return c.services.Vars }
func (c *ctxImpl) Sys() SysStore                { return c.services.Sys }
func (c *ctxImpl) Params() ParamStore           { return c.services.Params }
func (c *ctxImpl) Window() WindowService        { return c.services.Window }
func (c *ctxImpl) Capture() CaptureService      { return c.services.Capture }
func (c *ctxImpl) Stopwatches() StopwatchStore  { return c.services.Stopwatches }

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
