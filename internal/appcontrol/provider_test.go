package appcontrol

import (
	"context"
	"os"
	"testing"

	"github.com/yottaapp/yotta/internal/resource"
)

type fakePlatform struct {
	launched   int
	terminated int
	err        error
}

func (p *fakePlatform) Launch(context.Context, Profile) (uint32, error) {
	p.launched++
	return 4123, p.err
}
func (p *fakePlatform) Terminate(context.Context, Profile) (int, error) {
	p.terminated++
	return 2, p.err
}

func openSession(t *testing.T, provider *provider, operation string) any {
	t.Helper()
	object, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindApplication, Operations: []string{operation}, Config: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return object
}

func TestProviderUsesExactInstalledOperation(t *testing.T) {
	profile, _ := testProfile(t)
	platform := &fakePlatform{}
	provider := &provider{profile: profile, platform: platform}

	launchObject := openSession(t, provider, OperationLaunch)
	raw, err := provider.Invoke(context.Background(), launchObject, OperationLaunch, []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	launch, err := OpenLaunchResponse(raw)
	if err != nil || launch.ProcessID != 4123 || platform.launched != 1 {
		t.Fatalf("launch = %#v, calls=%d, err=%v", launch, platform.launched, err)
	}
	if _, err := provider.Invoke(context.Background(), launchObject, OperationTerminate, []byte(`{}`)); err == nil {
		t.Fatal("launch session accepted terminate operation")
	}

	terminateObject := openSession(t, provider, OperationTerminate)
	raw, err = provider.Invoke(context.Background(), terminateObject, OperationTerminate, []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	terminated, err := OpenTerminateResponse(raw)
	if err != nil || terminated.TerminatedCount != 2 || platform.terminated != 1 {
		t.Fatalf("terminate = %#v, calls=%d, err=%v", terminated, platform.terminated, err)
	}
}

func TestProviderContinuesAfterConfiguredExecutableUpdate(t *testing.T) {
	profile, path := testProfile(t)
	platform := &fakePlatform{}
	provider := &provider{profile: profile, platform: platform}
	object := openSession(t, provider, OperationLaunch)
	if err := os.WriteFile(path, []byte("installed-tool-v2"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Invoke(context.Background(), object, OperationLaunch, []byte(`{}`)); err != nil || platform.launched != 1 {
		t.Fatalf("Invoke() error = %v, calls=%d", err, platform.launched)
	}
}

func TestProviderLeavesExecutableAvailabilityToTheConfiguredOperation(t *testing.T) {
	profile, path := testProfile(t)
	platform := &fakePlatform{}
	provider := &provider{profile: profile, platform: platform}
	object := openSession(t, provider, OperationLaunch)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	_, err := provider.Invoke(context.Background(), object, OperationLaunch, []byte(`{}`))
	if err != nil || platform.launched != 1 {
		t.Fatalf("Invoke() error = %v, calls=%d", err, platform.launched)
	}
}

func TestProviderRejectsUnsupportedOperationAndWorkflowArguments(t *testing.T) {
	profile, _ := testProfile(t)
	provider := &provider{profile: profile, platform: &fakePlatform{}}
	if _, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindApplication, Operations: []string{"forged"}, Config: []byte(`{}`),
	}); err == nil {
		t.Fatal("Open accepted unsupported operation")
	}
	object := openSession(t, provider, OperationLaunch)
	if _, err := provider.Invoke(context.Background(), object, OperationLaunch, []byte(`{"args":["forged"]}`)); err == nil {
		t.Fatal("Invoke accepted workflow arguments")
	}
}
