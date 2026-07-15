package installed

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/resource"
)

type fakeDriver struct {
	operation string
	request   any
	err       error
	closed    int
}

func (driver *fakeDriver) Execute(_ context.Context, operation string, request any) error {
	driver.operation, driver.request = operation, request
	return driver.err
}
func (driver *fakeDriver) Close() error { driver.closed++; return driver.err }

func openInputSession(t *testing.T, provider *provider, operation string) any {
	t.Helper()
	object, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindInput, Operations: []string{operation}, CapabilityScope: []byte(`{"operation":"` + operation + `"}`), Config: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return object
}

func TestProviderUsesExactGrantedOperationAndCanonicalPayload(t *testing.T) {
	profile, _ := testProfile(t)
	driver := &fakeDriver{}
	provider := &provider{profile: profile, driver: driver}
	object := openInputSession(t, provider, OperationClick)
	payload, err := artifact.Marshal(ClickRequest{Point: Point{X: 0.5, Y: 0.25, Unit: "ratio"}, Button: "left", DurationMilliseconds: 25})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := provider.Invoke(context.Background(), object, OperationClick, payload)
	if err != nil || OpenEffectResponse(raw) != nil || driver.operation != OperationClick {
		t.Fatalf("Invoke() operation=%q response=%s error=%v", driver.operation, raw, err)
	}
	request, ok := driver.request.(ClickRequest)
	if !ok || request.Point.X != 0.5 || request.DurationMilliseconds != 25 {
		t.Fatalf("driver request = %#v", driver.request)
	}
	if _, err := provider.Invoke(context.Background(), object, OperationMove, []byte(`{"point":{"x":0.5,"y":0.5,"unit":"ratio"}}`)); err == nil {
		t.Fatal("click session accepted move operation")
	}
}

func TestProviderRejectsForgedScopeAndPayload(t *testing.T) {
	profile, _ := testProfile(t)
	provider := &provider{profile: profile, driver: &fakeDriver{}}
	if _, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindInput, Operations: []string{OperationClick}, CapabilityScope: []byte(`{"operation":"move"}`), Config: []byte(`{}`),
	}); err == nil {
		t.Fatal("provider accepted mismatched capability scope")
	}
	object := openInputSession(t, provider, OperationPressKeys)
	for _, payload := range [][]byte{
		[]byte(`{"keys":[],"durationMilliseconds":10}`),
		[]byte(`{"keys":["CTRL","ctrl"],"durationMilliseconds":10}`),
		[]byte(`{"keys":["NOT-A-KEY"],"durationMilliseconds":10}`),
		[]byte(`{"keys":["CTRL"],"durationMilliseconds":10,"command":"forged"}`),
	} {
		if _, err := provider.Invoke(context.Background(), object, OperationPressKeys, payload); err == nil {
			t.Fatalf("provider accepted payload %s", payload)
		}
	}
}

func TestProviderFailsClosedOnExecutableDrift(t *testing.T) {
	profile, path := testProfile(t)
	driver := &fakeDriver{}
	provider := &provider{profile: profile, driver: driver}
	object := openInputSession(t, provider, OperationTypeText)
	if err := os.WriteFile(path, []byte("tampered"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := provider.Invoke(context.Background(), object, OperationTypeText, []byte(`{"text":"hello"}`))
	var failure *Failure
	if !errors.As(err, &failure) || failure.Code != CodeIdentityChanged || driver.operation != "" {
		t.Fatalf("Invoke() error=%v operation=%q", err, driver.operation)
	}
}

func TestInstallationConsentAndUnsupportedHost(t *testing.T) {
	profile, _ := testProfile(t)
	consent, err := WorkflowConsentDigest("editor", profile)
	if err != nil || !consent.Valid() {
		t.Fatalf("WorkflowConsentDigest() = %q, %v", consent, err)
	}
	draft := InstallationDraft{Slot: "editor", Profile: profile.Machine(), Consent: consent}
	installations, err := Install([]InstallationDraft{draft})
	if !PlatformSupported() {
		if err == nil {
			t.Fatal("Install succeeded on unsupported host")
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(installations.Entries()) != 1 {
		t.Fatalf("entries = %d", len(installations.Entries()))
	}
	if err := installations.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := artifact.ParseDigest(consent.String()); err != nil {
		t.Fatal(err)
	}
}
