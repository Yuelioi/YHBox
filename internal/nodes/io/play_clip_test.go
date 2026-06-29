package io

import (
	"context"
	"errors"
	"testing"
	"time"

	"yotta/internal/node"
)

// mockClipPlayer — 测试用 node.ClipPlayer. blockCtx=true 时阻塞到 ctx 取消 (模拟回放中途停).
type mockClipPlayer struct {
	err       error
	gotClipID string
	blockCtx  bool
}

func (m *mockClipPlayer) Play(ctx context.Context, clipID string) error {
	m.gotClipID = clipID
	if m.blockCtx {
		<-ctx.Done()
		return ctx.Err()
	}
	return m.err
}

func TestPlayClip_PlaysAndFiresDone(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&PlayClip{})
	rn, _ := node.Get("PlayClip")

	mock := &mockClipPlayer{}
	svc := node.StubServices()
	svc.Clip = mock

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{pcInClipID: "fishing_intro"},
		nil, svc, false)

	if r.Error != nil {
		t.Fatalf("unexpected error: %v", r.Error)
	}
	if r.ExitName != pcOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	if mock.gotClipID != "fishing_intro" {
		t.Errorf("Play 收到 clipID %q, want fishing_intro", mock.gotClipID)
	}
}

func TestPlayClip_PlayErrorPropagates(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&PlayClip{})
	rn, _ := node.Get("PlayClip")

	sentinel := errors.New("backend boom")
	svc := node.StubServices()
	svc.Clip = &mockClipPlayer{err: sentinel}

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{pcInClipID: "x"},
		nil, svc, false)

	if r.Error == nil || !errors.Is(r.Error, sentinel) {
		t.Errorf("error = %v, want wrap of %v", r.Error, sentinel)
	}
	if r.ExitName != "" {
		t.Errorf("exit = %q, 出错不该 Fire Done", r.ExitName)
	}
}

func TestPlayClip_CtxCancelReturnsCanceled(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&PlayClip{})
	rn, _ := node.Get("PlayClip")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	svc := node.StubServices()
	svc.Clip = &mockClipPlayer{blockCtx: true}

	r := node.RunNode(ctx, rn, nil,
		map[string]any{pcInClipID: "x"},
		nil, svc, false)

	if r.Error == nil || !errors.Is(r.Error, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", r.Error)
	}
	if r.ExitName != "" {
		t.Errorf("exit = %q, 取消不该 Fire Done", r.ExitName)
	}
}

func TestPlayClip_RequiredClipIDMissing(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&PlayClip{})
	rn, _ := node.Get("PlayClip")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices(), false)
	if len(r.Validation) == 0 {
		t.Fatal("expected REQUIRED_FIELD_MISSING for ClipID")
	}
	if r.Validation[0].Code != "REQUIRED_FIELD_MISSING" {
		t.Errorf("validation[0].Code = %q, want REQUIRED_FIELD_MISSING", r.Validation[0].Code)
	}
}

func TestPlayClip_DependenciesExtractsClip(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&PlayClip{})
	rn, _ := node.Get("PlayClip")

	if rn.Dependencies == nil {
		t.Fatal("PlayClip should implement Dependencies")
	}
}
