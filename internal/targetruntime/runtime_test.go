package targetruntime_test

import (
	"context"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/resource"
	"github.com/yottaapp/yotta/internal/targetruntime"
)

type testProvider struct {
	opened int
	closed int
}

func (provider *testProvider) DescribeTarget(_ context.Context, targetID string) (targetruntime.Description, error) {
	if targetID != "target/editor" {
		return targetruntime.Description{}, errors.New("unexpected target")
	}
	return targetruntime.Description{Width: 2560, Height: 1440}, nil
}

func (provider *testProvider) Open(_ context.Context, request resource.ProviderOpenRequest) (any, error) {
	if request.TargetID != "target/editor" || request.Kind != "automation/input-session" ||
		len(request.Operations) != 1 || request.Operations[0] != "click" ||
		len(request.CapabilityScope) != 0 || request.CredentialBindingID != "" {
		return nil, errors.New("configured target request contains authorization data")
	}
	provider.opened++
	return &struct{}{}, nil
}

func (provider *testProvider) Invoke(_ context.Context, _ any, operation string, payload []byte) ([]byte, error) {
	if operation != "click" || string(payload) != `{}` {
		return nil, errors.New("unexpected configured target invocation")
	}
	return []byte(`{"ok":true}`), nil
}

func (provider *testProvider) Close(context.Context, any) error {
	provider.closed++
	return nil
}

func TestConfiguredTargetRunHasNoGrantScopeOrExpiration(t *testing.T) {
	provider := &testProvider{}
	snapshot, err := targetruntime.NewSnapshot([]targetruntime.Installation{{
		Slot: "editor", TargetID: "target/editor", Provider: provider,
	}})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := snapshot.NewRun()
	if err != nil {
		t.Fatal(err)
	}
	handle, err := runtime.Open(context.Background(), targetruntime.OpenRequest{
		Slot: "editor", Kind: "automation/input-session", Operations: []string{"click"}, Config: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := runtime.Invoke(context.Background(), handle, "click", []byte(`{}`))
	if err != nil || string(raw) != `{"ok":true}` {
		t.Fatalf("Invoke() = %s, %v", raw, err)
	}
	if err := runtime.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if provider.opened != 1 || provider.closed != 1 {
		t.Fatalf("provider opens/closes = %d/%d", provider.opened, provider.closed)
	}
}

func TestSnapshotDescribesTargetWithoutOpeningRunSession(t *testing.T) {
	provider := &testProvider{}
	snapshot, err := targetruntime.NewSnapshot([]targetruntime.Installation{{
		Slot: "editor", TargetID: "target/editor", Provider: provider,
	}})
	if err != nil {
		t.Fatal(err)
	}
	description, err := snapshot.Describe(context.Background(), "editor")
	if err != nil || description != (targetruntime.Description{Width: 2560, Height: 1440}) {
		t.Fatalf("description=%+v err=%v", description, err)
	}
	if provider.opened != 0 {
		t.Fatalf("description opened %d Run sessions", provider.opened)
	}
}
