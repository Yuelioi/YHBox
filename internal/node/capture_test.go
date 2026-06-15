package node

import (
	"context"
	"testing"
)

// captureRecvVars 记录式 VarStore — 只关心 SetScoped 调用情况.
type captureRecvVars struct {
	setName, setScope string
	setVal            any
	setCount          int
}

func (c *captureRecvVars) Get(string) (any, bool)            { return nil, false }
func (c *captureRecvVars) Set(string, any)                   {}
func (c *captureRecvVars) Inc(string, float64) float64       { return 0 }
func (c *captureRecvVars) GetScoped(string, string) (any, bool) { return nil, false }
func (c *captureRecvVars) SetScoped(name, scope string, v any) {
	c.setName, c.setScope, c.setVal = name, scope, v
	c.setCount++
}
func (c *captureRecvVars) IncScoped(string, string, float64) float64 { return 0 }
func (c *captureRecvVars) LastChange(string) int64                   { return 0 }

// newCaptureCtx 构造一个 Ctx, Vars 换成记录式 stub.
func newCaptureCtx(vars VarStore) Ctx {
	svc := StubServices()
	svc.Vars = vars
	return newCtx(context.Background(), svc, nil, nil)
}

func TestCapture_FilledNameWritesScopedAuto(t *testing.T) {
	vars := &captureRecvVars{}
	ctx := newCaptureCtx(vars)
	in := NewInputsFromConfig(map[string]any{"capture": "myVar"})

	Capture(ctx, in, "capture", 42)

	if vars.setCount != 1 {
		t.Fatalf("SetScoped called %d times, want 1", vars.setCount)
	}
	if vars.setName != "myVar" {
		t.Errorf("name = %q, want %q", vars.setName, "myVar")
	}
	if vars.setScope != "auto" {
		t.Errorf("scope = %q, want %q", vars.setScope, "auto")
	}
	if vars.setVal != 42 {
		t.Errorf("value = %v, want 42", vars.setVal)
	}
}

func TestCapture_EmptyOrBlankNameNoWrite(t *testing.T) {
	cases := map[string]string{
		"empty":    "",
		"blank":    "   ",
		"tabs":     "\t\n",
	}
	for name, field := range cases {
		t.Run(name, func(t *testing.T) {
			vars := &captureRecvVars{}
			ctx := newCaptureCtx(vars)
			in := NewInputsFromConfig(map[string]any{"capture": field})

			Capture(ctx, in, "capture", "value")

			if vars.setCount != 0 {
				t.Errorf("SetScoped called %d times, want 0 (blank name = no-op)", vars.setCount)
			}
		})
	}
}
