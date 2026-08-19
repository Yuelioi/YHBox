package noderuntime

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
)

func TestSettleTemplateMatchUsesTheFreshResultAndPropagatesFailure(t *testing.T) {
	waited := time.Duration(0)
	fresh, err := settleTemplateMatch(context.Background(), 25*time.Millisecond, func(_ context.Context, delay time.Duration) error {
		waited = delay
		return nil
	}, func(context.Context) (visionMatchResult, error) {
		return visionMatchResult{Matched: false, Score: 0.2}, nil
	})
	if err != nil || waited != 25*time.Millisecond || fresh.Matched || fresh.Score != 0.2 {
		t.Fatalf("fresh=%#v waited=%s err=%v", fresh, waited, err)
	}

	wantErr := errors.New("capture failed")
	if _, err := settleTemplateMatch(context.Background(), time.Millisecond, func(context.Context, time.Duration) error {
		return nil
	}, func(context.Context) (visionMatchResult, error) {
		return visionMatchResult{}, wantErr
	}); !errors.Is(err, wantErr) {
		t.Fatalf("settle relocation error = %v, want %v", err, wantErr)
	}
}

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
		{name: "present first frame", states: []bool{true}, wantPresent: true, timeout: time.Second, wantWaits: nil, wantCaptures: 1, wantMatched: true},
		{name: "appears", states: []bool{false, false, true}, wantPresent: true, timeout: time.Second, wantWaits: []time.Duration{100 * time.Millisecond, 100 * time.Millisecond}, wantCaptures: 3, wantMatched: true},
		{name: "disappears", states: []bool{true, false}, wantPresent: false, timeout: time.Second, wantWaits: []time.Duration{100 * time.Millisecond}, wantCaptures: 2, wantMatched: false},
		{name: "bounded timeout", states: []bool{false, false, false}, wantPresent: true, timeout: 250 * time.Millisecond, wantWaits: []time.Duration{100 * time.Millisecond, 100 * time.Millisecond, 50 * time.Millisecond}, wantCaptures: 3, wantMatched: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			index := 0
			var waits []time.Duration
			now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
			match, captures, err := pollTemplateStateWithClock(context.Background(), func(_ context.Context, delay time.Duration) error {
				waits = append(waits, delay)
				now = now.Add(delay)
				return nil
			}, func() time.Time { return now }, test.timeout, 100*time.Millisecond, test.wantPresent, func(context.Context) (visionMatchResult, error) {
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

func TestPollTemplateStateTimeoutIncludesObservationTime(t *testing.T) {
	match, captures, err := pollTemplateState(context.Background(), func(_ context.Context, delay time.Duration) error {
		time.Sleep(delay)
		return nil
	}, 10*time.Millisecond, 10*time.Millisecond, true, func(context.Context) (visionMatchResult, error) {
		time.Sleep(20 * time.Millisecond)
		return visionMatchResult{}, nil
	})
	if err != nil || captures != 1 || match.Matched {
		t.Fatalf("match=%#v captures=%d err=%v, want one observation bounded by the wall-clock timeout", match, captures, err)
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
