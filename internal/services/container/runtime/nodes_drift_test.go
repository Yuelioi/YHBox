package runtime

import (
	"context"
	"strings"
	"testing"

	"yhbox/internal/services/container"
	"yhbox/internal/services/container/nodekind"
	_ "yhbox/internal/services/container/nodekind/specs"
)

// TestExecNodeCoversAllExecKinds runs every registered exec-eligible kind through
// execNode and asserts the switch has a case for it. PureData/VisualOnly kinds are
// skipped (gatekeep rejects them; their dispatch lives in evalDataSource / no-op).
// The runtime call may fail for various legitimate reasons (no WindowTarget configured,
// missing template, etc.) — we only fail if the error message contains "drift!", which
// means the switch fell through.
//
// Bare ContainerRunner has nil maps / nil rt, so most dispatch handlers will nil-deref
// once they try to actually do work. That's fine: a panic past the switch proves the
// switch dispatched correctly. We recover the panic and only fail on the "drift!"
// sentinel error returned by the switch fall-through.
func TestExecNodeCoversAllExecKinds(t *testing.T) {
	r := &ContainerRunner{} // minimal runner; most dispatch paths will error/panic elsewhere
	ctx := context.Background()
	for _, kind := range nodekind.Kinds() {
		spec, _ := nodekind.Get(kind)
		if spec.IsPureData || spec.IsVisualOnly {
			continue // these can't reach execNode legally
		}
		t.Run(kind, func(t *testing.T) {
			n := &container.GraphNode{ID: "n1", Kind: kind, Config: map[string]any{}}
			err := callExecNodeSafely(r, ctx, n)
			if err != nil && strings.Contains(err.Error(), "drift!") {
				t.Errorf("kind %q registered but execNode switch has no case (drift): %v", kind, err)
			}
			// Any other error (or panic, or no error) is acceptable — we only fail on switch drift.
		})
	}
}

// callExecNodeSafely invokes execNode but recovers any panic from downstream handlers
// (bare ContainerRunner has nil rt/maps; e.g. MouseHoldStart will nil-deref r.rt.InputBus).
// A panic past the switch is proof the switch dispatched — exactly what the drift test wants.
func callExecNodeSafely(r *ContainerRunner, ctx context.Context, n *container.GraphNode) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			// Dispatch reached a handler that crashed on missing runtime state — not drift.
			err = nil
		}
	}()
	_, err = r.execNode(ctx, n, ExecToken{})
	return err
}

func TestExecNodeRejectsUnknownKind(t *testing.T) {
	r := &ContainerRunner{}
	n := &container.GraphNode{ID: "n1", Kind: "Bogus"}
	_, err := r.execNode(context.Background(), n, ExecToken{})
	if err == nil || !strings.Contains(err.Error(), "unknown kind") {
		t.Errorf("expected unknown-kind error, got %v", err)
	}
}

func TestExecNodeRejectsPureDataKind(t *testing.T) {
	r := &ContainerRunner{}
	n := &container.GraphNode{ID: "n1", Kind: "GetVar"} // IsPureData=true
	_, err := r.execNode(context.Background(), n, ExecToken{})
	if err == nil || !strings.Contains(err.Error(), "pure-data") {
		t.Errorf("expected pure-data error, got %v", err)
	}
}
