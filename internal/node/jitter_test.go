package node

import (
	"testing"
	"time"
)

func TestJitterInt_ZeroPct_Unchanged(t *testing.T) {
	if got := JitterInt(500, 0); got != 500 {
		t.Errorf("JitterInt(500,0) = %d, want 500 (pct=0 不变)", got)
	}
}

func TestJitterInt_BoundedWithinPct(t *testing.T) {
	const base, pct = 1000, 20
	lo, hi := 800, 1200 // factor ∈ [-p,+p] 保证 (N 个 [-p,p] uniform 的均值仍在 [-p,p])
	for i := 0; i < 10000; i++ {
		got := JitterInt(base, pct)
		if got < lo || got > hi {
			t.Fatalf("JitterInt 越界: got %d, want [%d,%d]", got, lo, hi)
		}
	}
}

func TestJitterDuration_ZeroPct_Unchanged(t *testing.T) {
	d := 500 * time.Millisecond
	if got := JitterDuration(d, 0); got != d {
		t.Errorf("JitterDuration(d,0) = %v, want %v", got, d)
	}
}

func TestJitterDuration_BoundedWithinPct(t *testing.T) {
	base := 1000 * time.Millisecond
	lo, hi := 800*time.Millisecond, 1200*time.Millisecond
	for i := 0; i < 10000; i++ {
		got := JitterDuration(base, 20)
		if got < lo || got > hi {
			t.Fatalf("JitterDuration 越界: got %v, want [%v,%v]", got, lo, hi)
		}
	}
}
