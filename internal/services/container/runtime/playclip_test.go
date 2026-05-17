package runtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
	"yhbox/internal/services/inputclip"
	"yhbox/internal/services/inputclip/backends"
)

type fakeClipResolver struct {
	clips map[string]*inputclip.InputClip
}

func (r *fakeClipResolver) Resolve(id string) (*inputclip.InputClip, bool) {
	c, ok := r.clips[id]
	return c, ok
}

type noopBackend struct{}

func (noopBackend) Send(inputclip.Event) error        { return nil }
func (noopBackend) SendBatch([]inputclip.Event) error { return nil }
func (noopBackend) ReleaseHeld() error                { return nil }
func (noopBackend) Name() string                      { return "noop" }

var _ backends.IInputBackend = noopBackend{}

// captureBackendLocal: 容器测试用的捕捉 backend (与 inputclip/runtime 包内 captureBackend 同义,
// 本地复制以免跨包依赖测试 helper).
type captureBackendLocal struct {
	mu   sync.Mutex
	sent []inputclip.Event
}

func (c *captureBackendLocal) Send(ev inputclip.Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, ev)
	return nil
}
func (c *captureBackendLocal) SendBatch(evs []inputclip.Event) error {
	for _, e := range evs {
		_ = c.Send(e)
	}
	return nil
}
func (c *captureBackendLocal) ReleaseHeld() error { return nil }
func (c *captureBackendLocal) Name() string       { return "capture" }

func (c *captureBackendLocal) snapshot() []inputclip.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]inputclip.Event, len(c.sent))
	copy(out, c.sent)
	return out
}

var _ backends.IInputBackend = (*captureBackendLocal)(nil)

func TestPlayClipNode_RunsAndReturnsOut(t *testing.T) {
	clip := &inputclip.InputClip{
		ID:         "test-clip",
		DurationUs: 50_000,
		Events: []inputclip.Event{
			{TUs: 0, Type: inputclip.EventTypeKeyDown, A: 0x57},
			{TUs: 30_000, Type: inputclip.EventTypeKeyUp, A: 0x57},
		},
	}
	c := &container.Container{
		ID: "c1",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "n1", Kind: "PlayClip", Config: map[string]any{"clipID": "test-clip"}},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopInputDriver{}, NoopColorDetector{}, nil, nil,
		&fakeClipResolver{clips: map[string]*inputclip.InputClip{"test-clip": clip}},
		noopBackend{}, 0)
	r := NewContainerRunner(rt)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	out, err := r.execPlayClip(ctx, &c.Graph.Nodes[0], ExecToken{NodeID: "n1", InPin: "in"})
	if err != nil {
		t.Fatalf("execPlayClip err: %v", err)
	}
	// n1.out 无下游 → out 应为 nil/空 (edges.next 返 nil).
	if len(out) != 0 {
		t.Errorf("expected 0 downstream tokens (no edges), got %d", len(out))
	}
}

func TestPlayClipNode_CancelOnCtxDone(t *testing.T) {
	// 100 events 跨 1s. ctx 100ms cancel. 应在 ctx 取消时返 ctx.Err.
	var evs []inputclip.Event
	for i := 0; i < 100; i++ {
		evs = append(evs, inputclip.Event{
			TUs: uint64(i) * 10_000, Type: inputclip.EventTypeKeyDown, A: 0x57,
		})
	}
	clip := &inputclip.InputClip{ID: "long", DurationUs: 1_000_000, Events: evs}
	c := &container.Container{
		ID: "c1",
		Graph: container.Graph{Nodes: []container.GraphNode{
			{ID: "n1", Kind: "PlayClip", Config: map[string]any{"clipID": "long"}},
		}},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopInputDriver{}, NoopColorDetector{}, nil, nil,
		&fakeClipResolver{clips: map[string]*inputclip.InputClip{"long": clip}},
		noopBackend{}, 0)
	r := NewContainerRunner(rt)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := r.execPlayClip(ctx, &c.Graph.Nodes[0], ExecToken{NodeID: "n1", InPin: "in"})
	if err == nil {
		t.Fatal("应在 ctx 取消时返错")
	}
}

func TestPlayClipNode_MissingClipID(t *testing.T) {
	c := &container.Container{
		ID: "c1",
		Graph: container.Graph{Nodes: []container.GraphNode{
			{ID: "n1", Kind: "PlayClip", Config: map[string]any{}},
		}},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopInputDriver{}, NoopColorDetector{}, nil, nil,
		&fakeClipResolver{}, noopBackend{}, 0)
	r := NewContainerRunner(rt)

	_, err := r.execPlayClip(context.Background(), &c.Graph.Nodes[0], ExecToken{NodeID: "n1", InPin: "in"})
	if err == nil {
		t.Fatal("missing clipID 应报错")
	}
}

func TestPlayClipNode_ClipNotFound(t *testing.T) {
	c := &container.Container{
		ID: "c1",
		Graph: container.Graph{Nodes: []container.GraphNode{
			{ID: "n1", Kind: "PlayClip", Config: map[string]any{"clipID": "ghost"}},
		}},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopInputDriver{}, NoopColorDetector{}, nil, nil,
		&fakeClipResolver{clips: map[string]*inputclip.InputClip{}}, noopBackend{}, 0)
	r := NewContainerRunner(rt)

	_, err := r.execPlayClip(context.Background(), &c.Graph.Nodes[0], ExecToken{NodeID: "n1", InPin: "in"})
	if err == nil {
		t.Fatal("clip not found 应报错")
	}
}

func TestPlayClipNode_KeepRangesParsed(t *testing.T) {
	// 校验 keepRanges JSON 解析 + 截播 — 通过 captureBackend 计数收到几个 events.
	clip := &inputclip.InputClip{
		ID:         "trim",
		DurationUs: 120_000,
		Events: []inputclip.Event{
			{TUs: 0, A: 1, Type: inputclip.EventTypeKeyDown},
			{TUs: 30_000, A: 2, Type: inputclip.EventTypeKeyDown},
			{TUs: 60_000, A: 3, Type: inputclip.EventTypeKeyDown},
			{TUs: 90_000, A: 4, Type: inputclip.EventTypeKeyDown},
			{TUs: 120_000, A: 5, Type: inputclip.EventTypeKeyDown},
		},
	}
	c := &container.Container{
		ID: "c1",
		Graph: container.Graph{Nodes: []container.GraphNode{
			{ID: "n1", Kind: "PlayClip", Config: map[string]any{
				"clipID": "trim",
				"keepRanges": []any{
					map[string]any{"fromUs": float64(0), "toUs": float64(30_000)},
					map[string]any{"fromUs": float64(90_000), "toUs": float64(120_000)},
				},
			}},
		}},
	}
	cap := &captureBackendLocal{}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopInputDriver{}, NoopColorDetector{}, nil, nil,
		&fakeClipResolver{clips: map[string]*inputclip.InputClip{"trim": clip}},
		cap, 0)
	r := NewContainerRunner(rt)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := r.execPlayClip(ctx, &c.Graph.Nodes[0], ExecToken{NodeID: "n1", InPin: "in"})
	if err != nil {
		t.Fatalf("execPlayClip: %v", err)
	}
	evs := cap.snapshot()
	if len(evs) != 4 {
		t.Fatalf("expected 4 events (A=3 cut), got %d: %+v", len(evs), evs)
	}
	if evs[0].A != 1 || evs[1].A != 2 || evs[2].A != 4 || evs[3].A != 5 {
		t.Errorf("event order wrong: %+v", evs)
	}
}
