package runtime

import (
	"context"
	"testing"
	"time"
)

type fakeGame struct {
	attempts  int
	successOn int
}

func (f *fakeGame) BringToForeground(hwnd uintptr) bool {
	f.attempts++
	return f.attempts >= f.successOn
}

func TestBringToForegroundWithRetry_FirstTrySuccess(t *testing.T) {
	g := &fakeGame{successOn: 1}
	start := time.Now()
	ok := bringToForegroundWithRetry(context.Background(), g, 0xDEAD, 3, 50*time.Millisecond)
	if !ok {
		t.Errorf("expected success")
	}
	if g.attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", g.attempts)
	}
	if time.Since(start) > 30*time.Millisecond {
		t.Errorf("first-try success should not sleep, took %v", time.Since(start))
	}
}

func TestBringToForegroundWithRetry_RecoverOnThird(t *testing.T) {
	g := &fakeGame{successOn: 3}
	ok := bringToForegroundWithRetry(context.Background(), g, 0xDEAD, 3, 10*time.Millisecond)
	if !ok {
		t.Errorf("expected success on 3rd try")
	}
	if g.attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", g.attempts)
	}
}

func TestBringToForegroundWithRetry_AllFail(t *testing.T) {
	g := &fakeGame{successOn: 999}
	ok := bringToForegroundWithRetry(context.Background(), g, 0xDEAD, 3, 10*time.Millisecond)
	if ok {
		t.Errorf("expected failure")
	}
	if g.attempts != 3 {
		t.Errorf("expected exactly 3 attempts, got %d", g.attempts)
	}
}

func TestBringToForegroundWithRetry_ContextCancel(t *testing.T) {
	g := &fakeGame{successOn: 999}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ok := bringToForegroundWithRetry(ctx, g, 0xDEAD, 3, 50*time.Millisecond)
	if ok {
		t.Errorf("cancelled ctx should not succeed")
	}
	if g.attempts < 1 {
		t.Errorf("expected at least 1 attempt before cancel kicks in")
	}
}

func TestBringToForegroundWithRetry_ZeroHwnd(t *testing.T) {
	g := &fakeGame{successOn: 1}
	ok := bringToForegroundWithRetry(context.Background(), g, 0, 3, 10*time.Millisecond)
	if ok {
		t.Errorf("hwnd=0 should never succeed")
	}
	if g.attempts != 0 {
		t.Errorf("hwnd=0 should not call provider, got %d attempts", g.attempts)
	}
}
