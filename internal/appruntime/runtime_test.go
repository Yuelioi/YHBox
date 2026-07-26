package appruntime

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestRuntimeStartsInOrderAndClosesInReverse(t *testing.T) {
	var calls []string
	runtime := New(
		testResource("one", &calls, nil, nil),
		testResource("two", &calls, nil, nil),
		testResource("three", &calls, nil, nil),
	)

	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	if err := runtime.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := runtime.Close(context.Background()); err != nil {
		t.Fatalf("second Close: %v", err)
	}

	want := []string{"start:one", "start:two", "start:three", "close:three", "close:two", "close:one"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls=%v, want %v", calls, want)
	}
}

func TestRuntimeRollsBackStartedResourcesAndJoinsErrors(t *testing.T) {
	startFailure := errors.New("start failed")
	rollbackFailure := errors.New("rollback failed")
	var calls []string
	runtime := New(
		testResource("one", &calls, nil, rollbackFailure),
		testResource("two", &calls, startFailure, nil),
		testResource("three", &calls, nil, nil),
	)

	err := runtime.Start(context.Background())
	if !errors.Is(err, startFailure) || !errors.Is(err, rollbackFailure) {
		t.Fatalf("Start error=%v, want joined start and rollback failures", err)
	}
	want := []string{"start:one", "start:two", "close:one"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls=%v, want %v", calls, want)
	}
	if closeErr := runtime.Close(context.Background()); closeErr != err {
		t.Fatalf("Close error=%v, want same startup result %v", closeErr, err)
	}
	if retryErr := runtime.Start(context.Background()); retryErr != err {
		t.Fatalf("retry Start error=%v, want same startup result %v", retryErr, err)
	}
}

func TestRuntimeCloseAttemptsEveryResourceAndAggregatesErrors(t *testing.T) {
	closeOne := errors.New("close one")
	closeThree := errors.New("close three")
	var calls []string
	runtime := New(
		testResource("one", &calls, nil, closeOne),
		testResource("two", &calls, nil, nil),
		testResource("three", &calls, nil, closeThree),
	)
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	err := runtime.Close(context.Background())
	if !errors.Is(err, closeOne) || !errors.Is(err, closeThree) {
		t.Fatalf("Close error=%v, want both failures", err)
	}
	want := []string{"start:one", "start:two", "start:three", "close:three", "close:two", "close:one"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls=%v, want %v", calls, want)
	}
}

func TestRuntimeConcurrentCloseRunsResourcesOnce(t *testing.T) {
	var mu sync.Mutex
	closeCalls := 0
	runtime := New(Resource{
		Name:  "resource",
		Start: func(context.Context) error { return nil },
		Close: func(context.Context) error {
			mu.Lock()
			closeCalls++
			mu.Unlock()
			return nil
		},
	})
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := runtime.Close(context.Background()); err != nil {
				t.Errorf("Close: %v", err)
			}
		}()
	}
	wg.Wait()
	mu.Lock()
	defer mu.Unlock()
	if closeCalls != 1 {
		t.Fatalf("close calls=%d, want 1", closeCalls)
	}
}

func TestRuntimeValidatesAllResourcesBeforeStartingAny(t *testing.T) {
	started := false
	runtime := New(
		Resource{
			Name:  "valid",
			Start: func(context.Context) error { started = true; return nil },
			Close: func(context.Context) error { return nil },
		},
		Resource{Name: "invalid"},
	)

	if err := runtime.Start(context.Background()); err == nil {
		t.Fatal("Start succeeded with invalid resource")
	}
	if started {
		t.Fatal("valid resource started before later configuration was validated")
	}
}

func TestRuntimeConcurrentStartWaitHonorsContext(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	runtime := New(Resource{
		Name: "blocked-start",
		Start: func(context.Context) error {
			close(started)
			<-release
			return nil
		},
		Close: func(context.Context) error { return nil },
	})
	firstDone := make(chan error, 1)
	go func() { firstDone <- runtime.Start(context.Background()) }()
	waitSignal(t, "first Start", started)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := runtime.Start(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("concurrent Start error=%v, want deadline exceeded", err)
	}
	close(release)
	if err := waitError(t, "first Start completion", firstDone); err != nil {
		t.Fatalf("first Start: %v", err)
	}
}

func TestRuntimeConcurrentCloseWaitHonorsContext(t *testing.T) {
	closeStarted := make(chan struct{})
	release := make(chan struct{})
	runtime := New(Resource{
		Name:  "blocked-close",
		Start: func(context.Context) error { return nil },
		Close: func(context.Context) error {
			close(closeStarted)
			<-release
			return nil
		},
	})
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	firstDone := make(chan error, 1)
	go func() { firstDone <- runtime.Close(context.Background()) }()
	waitSignal(t, "first Close", closeStarted)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := runtime.Close(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("concurrent Close error=%v, want deadline exceeded", err)
	}
	close(release)
	if err := waitError(t, "first Close completion", firstDone); err != nil {
		t.Fatalf("first Close: %v", err)
	}
}

func TestRuntimeRollbackUsesIndependentBoundedContext(t *testing.T) {
	startFailure := errors.New("start failed")
	runtime := NewWithOptions(Options{RollbackTimeout: 10 * time.Millisecond},
		Resource{
			Name:  "slow-close",
			Start: func(context.Context) error { return nil },
			Close: func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
		},
		Resource{
			Name:  "failed-start",
			Start: func(context.Context) error { return startFailure },
			Close: func(context.Context) error { return nil },
		},
	)

	err := runtime.Start(context.Background())
	if !errors.Is(err, startFailure) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Start error=%v, want start failure and bounded rollback deadline", err)
	}
}

func testResource(name string, calls *[]string, startErr, closeErr error) Resource {
	return Resource{
		Name: name,
		Start: func(context.Context) error {
			*calls = append(*calls, "start:"+name)
			return startErr
		},
		Close: func(context.Context) error {
			*calls = append(*calls, "close:"+name)
			return closeErr
		},
	}
}

func waitSignal(t *testing.T, name string, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}

func waitError(t *testing.T, name string, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", name)
		return nil
	}
}
