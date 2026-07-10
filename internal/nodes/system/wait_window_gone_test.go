package system

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/pkg/winutil"
)

// TestWaitWindowGone_Gone: seam 返 nil → 节点走 Gone 出口
func TestWaitWindowGone_Gone(t *testing.T) {
	orig := waitWindowGone
	waitWindowGone = func(_ context.Context, _ winutil.MatchSpec, _, _ time.Duration) error {
		return nil
	}
	defer func() { waitWindowGone = orig }()

	node.ResetRegistryForTest()
	node.Register(&WaitWindowGone{})
	rn, _ := node.Get("WaitWindowGone")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{wwgInTitle: "SomeWindow"},
		nil, node.StubServices(), false)
	if r.Error != nil {
		t.Fatalf("unexpected error: %v", r.Error)
	}
	if r.ExitName != wwgOutGone {
		t.Errorf("exit = %q, want %q", r.ExitName, wwgOutGone)
	}
}

// TestWaitWindowGone_Timeout: seam 返 ErrWindowStillPresent → 节点走 Timeout 出口
func TestWaitWindowGone_Timeout(t *testing.T) {
	orig := waitWindowGone
	waitWindowGone = func(_ context.Context, _ winutil.MatchSpec, _, _ time.Duration) error {
		return winutil.ErrWindowStillPresent
	}
	defer func() { waitWindowGone = orig }()

	node.ResetRegistryForTest()
	node.Register(&WaitWindowGone{})
	rn, _ := node.Get("WaitWindowGone")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{wwgInTitle: "SomeWindow"},
		nil, node.StubServices(), false)
	if r.Error != nil {
		t.Fatalf("unexpected error: %v", r.Error)
	}
	if r.ExitName != wwgOutTimeout {
		t.Errorf("exit = %q, want %q", r.ExitName, wwgOutTimeout)
	}
}

// TestWaitWindowGone_BareError: seam 返 generic error → 节点裸冒泡
func TestWaitWindowGone_BareError(t *testing.T) {
	orig := waitWindowGone
	someErr := errors.New("something went wrong")
	waitWindowGone = func(_ context.Context, _ winutil.MatchSpec, _, _ time.Duration) error {
		return someErr
	}
	defer func() { waitWindowGone = orig }()

	node.ResetRegistryForTest()
	node.Register(&WaitWindowGone{})
	rn, _ := node.Get("WaitWindowGone")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{wwgInTitle: "SomeWindow"},
		nil, node.StubServices(), false)
	if !errors.Is(r.Error, someErr) {
		t.Errorf("expected bare error propagation, got %v", r.Error)
	}
}
