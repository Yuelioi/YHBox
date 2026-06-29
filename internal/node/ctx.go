// internal/node/ctx.go
package node

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type ctxImpl struct {
	ctx      context.Context
	services ServiceBundle
	spec     *Spec
	fired    bool // Run 内首次 Fire 后置 true; 后续任何 builder Fire → panic
	// captureBindings 路径② (Spec C): config.capture 解析出的 字段名→变量名。
	// region 节点 ctx.CaptureOutput 用; 来自 newCtx 传入的 config["capture"]。
	captureBindings map[string]string
}

func newCtx(ctx context.Context, services ServiceBundle, spec *Spec, config map[string]any) *ctxImpl {
	return &ctxImpl{ctx: ctx, services: services, spec: spec, captureBindings: parseCaptureBindings(config)}
}

// parseCaptureBindings 从 node config 取 capture 绑定 (字段名→变量名), nil-safe。
func parseCaptureBindings(config map[string]any) map[string]string {
	raw, ok := config["capture"].(map[string]any)
	if !ok {
		return nil
	}
	m := make(map[string]string, len(raw))
	for k, v := range raw {
		if s, ok := v.(string); ok {
			m[k] = s
		}
	}
	return m
}

// CaptureOutput 路径② region per-iteration 捕获 — 见 Ctx 接口。
func (c *ctxImpl) CaptureOutput(field string, value any) {
	name := strings.TrimSpace(c.captureBindings[field])
	if name == "" || c.services.Vars == nil {
		return
	}
	c.services.Vars.SetScoped(name, "auto", value)
}

func (c *ctxImpl) Context() context.Context    { return c.ctx }
func (c *ctxImpl) Now() time.Time              { return time.Now() }
func (c *ctxImpl) Vision() VisionService       { return c.services.Vision }
func (c *ctxImpl) Log() LogService             { return c.services.Log }
func (c *ctxImpl) Input() InputService         { return c.services.Input }
func (c *ctxImpl) Vars() VarStore              { return c.services.Vars }
func (c *ctxImpl) Params() ParamStore          { return c.services.Params }
func (c *ctxImpl) Window() WindowService       { return c.services.Window }
func (c *ctxImpl) Target() TargetService       { return c.services.Target }
func (c *ctxImpl) App() AppLifecycleService    { return c.services.App }
func (c *ctxImpl) Capture() CaptureService     { return c.services.Capture }
func (c *ctxImpl) Stopwatches() StopwatchStore { return c.services.Stopwatches }
func (c *ctxImpl) Clip() ClipPlayer            { return c.services.Clip }
func (c *ctxImpl) Subgraphs() SubgraphCaller   { return c.services.Subgraphs }
func (c *ctxImpl) AI() AIProviderService       { return c.services.AI }

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
