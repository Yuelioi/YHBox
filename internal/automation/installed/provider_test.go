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
	capture   []byte
}

func (driver *fakeDriver) Execute(_ context.Context, operation string, request any) error {
	driver.operation, driver.request = operation, request
	return driver.err
}
func (driver *fakeDriver) Capture(_ context.Context) ([]byte, error) {
	return append([]byte(nil), driver.capture...), driver.err
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

func openWindowSession(t *testing.T, provider *provider, operation string) any {
	t.Helper()
	object, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindWindow, Operations: []string{operation}, CapabilityScope: []byte(`{"operation":"` + operation + `"}`), Config: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return object
}

func openCaptureSession(t *testing.T, provider *provider) any {
	t.Helper()
	object, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindCapture, Operations: CaptureOperations(), CapabilityScope: []byte(`{"operation":"capture"}`), Config: []byte(`{}`),
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

func TestProviderSeparatesWindowAuthorityFromInputAuthority(t *testing.T) {
	profile, _ := testProfile(t)
	driver := &fakeDriver{}
	provider := &provider{profile: profile, driver: driver}
	object := openWindowSession(t, provider, OperationActivate)
	raw, err := provider.Invoke(context.Background(), object, OperationActivate, []byte(`{}`))
	if err != nil || OpenEffectResponse(raw) != nil || driver.operation != OperationActivate {
		t.Fatalf("activate operation=%q response=%s error=%v", driver.operation, raw, err)
	}
	if _, ok := driver.request.(struct{}); !ok {
		t.Fatalf("activate request = %#v", driver.request)
	}
	for _, request := range []resource.ProviderOpenRequest{
		{Kind: KindInput, Operations: []string{OperationActivate}, CapabilityScope: []byte(`{"operation":"activate"}`), Config: []byte(`{}`)},
		{Kind: KindWindow, Operations: []string{OperationClick}, CapabilityScope: []byte(`{"operation":"click"}`), Config: []byte(`{}`)},
		{Kind: KindCapture, Operations: []string{OperationCapture}, CapabilityScope: []byte(`{"operation":"capture"}`), Config: []byte(`{}`)},
	} {
		if _, err := provider.Open(context.Background(), request); err == nil {
			t.Fatalf("provider accepted cross-capability request %#v", request)
		}
	}
}

func TestProviderCaptureIsBoundedAndRequiresCaptureBeforeRead(t *testing.T) {
	profile, _ := testProfile(t)
	driver := &fakeDriver{capture: []byte("png-bytes")}
	provider := &provider{profile: profile, driver: driver}
	object := openCaptureSession(t, provider)
	if _, err := provider.Invoke(context.Background(), object, OperationReadCapture, []byte(`{"offset":0,"length":1}`)); err == nil {
		t.Fatal("capture session allowed read before capture")
	}
	raw, err := provider.Invoke(context.Background(), object, OperationCapture, []byte(`{}`))
	response, decodeErr := OpenCaptureResponse(raw)
	if err != nil || decodeErr != nil || response.Size != 9 || response.MediaType != "image/png" {
		t.Fatalf("capture response=%s decoded=%#v error=%v decode=%v", raw, response, err, decodeErr)
	}
	chunk, err := provider.Invoke(context.Background(), object, OperationReadCapture, []byte(`{"offset":4,"length":5}`))
	if err != nil || string(chunk) != "bytes" {
		t.Fatalf("read capture=%q error=%v", chunk, err)
	}
	if _, err := provider.Invoke(context.Background(), object, OperationReadCapture, []byte(`{"offset":0,"length":65537}`)); err == nil {
		t.Fatal("capture session accepted oversized range")
	}
	if err := provider.Close(context.Background(), object); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Invoke(context.Background(), object, OperationCapture, []byte(`{}`)); err == nil {
		t.Fatal("closed capture session accepted capture")
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
