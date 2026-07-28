package noderuntime

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
)

func TestPollTemplateStateUsesFreshObservationsAndBoundedWaits(t *testing.T) {
	tests := []struct {
		name         string
		states       []bool
		wantPresent  bool
		timeout      time.Duration
		wantWaits    []time.Duration
		wantCaptures int
		wantMatched  bool
	}{
		{name: "appears", states: []bool{false, false, true}, wantPresent: true, timeout: time.Second, wantWaits: []time.Duration{100 * time.Millisecond, 100 * time.Millisecond}, wantCaptures: 3, wantMatched: true},
		{name: "disappears", states: []bool{true, false}, wantPresent: false, timeout: time.Second, wantWaits: []time.Duration{100 * time.Millisecond}, wantCaptures: 2, wantMatched: false},
		{name: "bounded timeout", states: []bool{false, false, false, false}, wantPresent: true, timeout: 250 * time.Millisecond, wantWaits: []time.Duration{100 * time.Millisecond, 100 * time.Millisecond, 50 * time.Millisecond}, wantCaptures: 4, wantMatched: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			index := 0
			var waits []time.Duration
			match, captures, err := pollTemplateState(context.Background(), func(_ context.Context, delay time.Duration) error {
				waits = append(waits, delay)
				return nil
			}, test.timeout, 100*time.Millisecond, test.wantPresent, func(context.Context) (visionMatchResult, error) {
				matched := test.states[index]
				index++
				return visionMatchResult{Matched: matched}, nil
			})
			if err != nil || captures != test.wantCaptures || match.Matched != test.wantMatched || !reflect.DeepEqual(waits, test.wantWaits) {
				t.Fatalf("match=%#v captures=%d waits=%v err=%v", match, captures, waits, err)
			}
		})
	}
}

func TestPollTemplateStatePropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, captures, err := pollTemplateState(ctx, func(ctx context.Context, _ time.Duration) error {
		return ctx.Err()
	}, time.Second, 100*time.Millisecond, true, func(context.Context) (visionMatchResult, error) {
		return visionMatchResult{}, nil
	})
	if err != context.Canceled || captures != 1 {
		t.Fatalf("captures=%d err=%v", captures, err)
	}
}

func TestEmitTemplateMatchStatusDistinguishesMatchAndTimeout(t *testing.T) {
	tests := []struct {
		name    string
		matched bool
		want    string
	}{
		{name: "matched", matched: true, want: nodes.AutomationTemplateMatchedStatus},
		{name: "timeout", matched: false, want: nodes.AutomationTemplateTimeoutStatus},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotCode string
			var gotCounters map[string]int64
			invocation := nodeadapter.Invocation{EmitStatus: func(_ context.Context, code string, counters map[string]int64) error {
				gotCode = code
				gotCounters = counters
				return nil
			}}
			if err := emitTemplateMatchStatus(context.Background(), invocation, test.matched, map[string]int64{
				"captures": 4, "capture_ms": 12, "match_ms": 7,
			}); err != nil {
				t.Fatal(err)
			}
			if gotCode != test.want || gotCounters["captures"] != 4 || gotCounters["capture_ms"] != 12 || gotCounters["match_ms"] != 7 {
				t.Fatalf("code=%q counters=%v", gotCode, gotCounters)
			}
		})
	}
}
