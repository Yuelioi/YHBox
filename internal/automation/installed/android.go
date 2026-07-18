package installed

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/adbexec"
	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
)

// AndroidDeviceDescriptor is an exact ADB transport identity. Serial alone is
// insufficient because an emulator or USB transport can be reused later.
type AndroidDeviceDescriptor struct {
	Serial      string `json:"serial"`
	State       string `json:"state"`
	Product     string `json:"product"`
	Model       string `json:"model"`
	Device      string `json:"device"`
	TransportID string `json:"transportId"`
}

func DiscoverAndroidDevices(ctx context.Context) ([]AndroidDeviceDescriptor, error) {
	out, err := adbexec.CommandContext(ctx, "devices", "-l").CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("ADB discovery failed (%s): %s", adbexec.ExecutablePath(), message)
	}
	return parseADBDevices(string(out)), nil
}

func parseADBDevices(out string) []AndroidDeviceDescriptor {
	devices := []AndroidDeviceDescriptor{}
	for _, raw := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		fields := strings.Fields(strings.TrimSpace(raw))
		if len(fields) < 2 || fields[0] == "List" || strings.HasPrefix(fields[0], "*") {
			continue
		}
		device := AndroidDeviceDescriptor{Serial: fields[0], State: fields[1]}
		for _, field := range fields[2:] {
			key, value, ok := strings.Cut(field, ":")
			if !ok {
				continue
			}
			switch key {
			case "product":
				device.Product = value
			case "model":
				device.Model = value
			case "device":
				device.Device = value
			case "transport_id":
				device.TransportID = value
			}
		}
		devices = append(devices, device)
	}
	return devices
}

type androidDriver struct {
	profile Profile
	mu      sync.Mutex
	closed  bool
}

// AndroidHealthProbe exposes only resolution/identity verification to the
// trusted Settings diagnostics surface, never input execution.
type AndroidHealthProbe struct{ driver *androidDriver }

func NewAndroidHealthProbe(profile Profile) (AndroidHealthProbe, error) {
	opened, err := newAndroidDriver(profile)
	if err != nil {
		return AndroidHealthProbe{}, err
	}
	return AndroidHealthProbe{driver: opened.(*androidDriver)}, nil
}

func (probe AndroidHealthProbe) Resolve(ctx context.Context) (target.Target, error) {
	if probe.driver == nil {
		return target.Target{}, errors.New("android health probe is unavailable")
	}
	return probe.driver.ResolveTarget(ctx)
}

func newAndroidDriver(profile Profile) (driver, error) {
	if !profile.Valid() || profile.AdapterKind() != AdapterKindAndroidADB || profile.TargetKind() != TargetKindAndroidDevice {
		return nil, failure(CodeContractViolation, errors.New("android ADB adapter requires an Android device profile"))
	}
	return &androidDriver{profile: profile}, nil
}

func (d *androidDriver) ResolveTarget(ctx context.Context) (target.Target, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.resolveLocked(ctx)
}

func (d *androidDriver) resolveLocked(ctx context.Context) (target.Target, error) {
	if d.closed {
		return target.Target{}, failure(CodeContractViolation, errors.New("android ADB automation driver is closed"))
	}
	machine, ok := AndroidProfile(d.profile)
	if !ok {
		return target.Target{}, failure(CodeContractViolation, errors.New("android driver received another adapter profile"))
	}
	timeout := time.Duration(machine.ResolveTimeoutMilliseconds) * time.Millisecond
	resolveCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	devices, err := DiscoverAndroidDevices(resolveCtx)
	if err != nil {
		return target.Target{}, failure(CodeTargetNotFound, err)
	}
	var found *AndroidDeviceDescriptor
	for index := range devices {
		if devices[index].Serial == machine.ADBSerial {
			found = &devices[index]
			break
		}
	}
	if found == nil {
		return target.Target{}, failure(CodeTargetNotFound, fmt.Errorf("ADB device %q is not connected", machine.ADBSerial))
	}
	if found.State != "device" {
		return target.Target{}, failure(CodeTargetNotFound, fmt.Errorf("ADB device %q is %s; unlock it and accept USB debugging authorization", machine.ADBSerial, found.State))
	}
	if found.Product != machine.ADBProduct || found.Model != machine.ADBModel || found.Device != machine.ADBDevice {
		return target.Target{}, failure(CodeIdentityChanged, fmt.Errorf("ADB device %q identity changed: expected %s/%s/%s, got %s/%s/%s", machine.ADBSerial, machine.ADBProduct, machine.ADBModel, machine.ADBDevice, found.Product, found.Model, found.Device))
	}
	size, err := androidDeviceSize(resolveCtx, machine.ADBSerial)
	if err != nil {
		return target.Target{}, failure(CodeTargetNotFound, err)
	}
	return target.Target{
		ID: "android:" + machine.ADBSerial, Kind: target.KindAndroidADB, DisplayName: machine.ADBModel,
		Ref: target.TargetRef{ADBSerial: machine.ADBSerial}, Resolution: size,
		Metadata: map[string]any{"product": machine.ADBProduct, "device": machine.ADBDevice, "package": machine.AndroidPackage},
	}, nil
}

var androidSizePattern = regexp.MustCompile(`(?m)(?:Physical|Override) size:\s*(\d+)x(\d+)`)

func androidDeviceSize(ctx context.Context, serial string) (target.Size, error) {
	out, err := adbexec.CommandContext(ctx, "-s", serial, "shell", "wm", "size").CombinedOutput()
	if err != nil {
		return target.Size{}, fmt.Errorf("read ADB device resolution: %s", strings.TrimSpace(string(out)))
	}
	matches := androidSizePattern.FindAllStringSubmatch(string(out), -1)
	if len(matches) == 0 {
		return target.Size{}, fmt.Errorf("ADB device %q did not report a display resolution", serial)
	}
	var size target.Size
	last := matches[len(matches)-1]
	if _, err := fmt.Sscanf(last[1]+"x"+last[2], "%dx%d", &size.W, &size.H); err != nil || size.W <= 0 || size.H <= 0 {
		return target.Size{}, fmt.Errorf("ADB device %q reported an invalid display resolution", serial)
	}
	return size, nil
}

func (d *androidDriver) Execute(ctx context.Context, operation string, raw any) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	resolvedTarget, err := d.resolveLocked(ctx)
	if err != nil {
		return err
	}
	resolved, err := controller.NewAndroidADBController(resolvedTarget, controller.AndroidADBDeps{Backend: AdapterKindAndroidADB})
	if err != nil {
		return failure(CodeContractViolation, err)
	}
	switch request := raw.(type) {
	case struct{}:
		profile, ok := AndroidProfile(d.profile)
		if !ok {
			return failure(CodeContractViolation, errors.New("android driver received another adapter profile"))
		}
		intent := profile.AndroidPackage
		if operation == OperationActivate {
			return resolved.StartApp(ctx, controller.StartAppRequest{Intent: intent})
		}
		if operation == OperationStopApp {
			return resolved.StopApp(ctx, controller.StopAppRequest{Intent: intent})
		}
	case ClickRequest:
		if request.Button != "left" {
			return failure(CodeInvalidRequest, errors.New("android ADB click supports the primary button only"))
		}
		return resolved.Click(ctx, controller.ClickRequest{Point: androidPoint(request.Point), Button: request.Button, DurationMs: int(request.DurationMilliseconds)})
	case MoveRequest:
		return resolved.Move(ctx, controller.MoveRequest{Point: androidPoint(request.Point)})
	case ScrollRequest:
		return resolved.Scroll(ctx, controller.ScrollRequest{Point: androidPoint(request.Point), Notches: int(request.Notches), Horizontal: request.Horizontal})
	case DragRequest:
		if request.Button != "left" {
			return failure(CodeInvalidRequest, errors.New("android ADB drag supports the primary button only"))
		}
		return resolved.Drag(ctx, controller.DragRequest{From: androidPoint(request.From), To: androidPoint(request.To), Button: request.Button, DurationMs: int(request.DurationMilliseconds)})
	case TypeTextRequest:
		return resolved.Text(ctx, controller.TextRequest{Text: request.Text})
	}
	return failure(CodeContractViolation, fmt.Errorf("android ADB operation %q is unsupported", operation))
}

func androidPoint(point Point) target.Point {
	if point.Unit == "px" {
		return target.Point{X: point.X, Y: point.Y, Space: target.SpaceAndroidDevice}
	}
	return target.NewNormalizedPoint(point.X, point.Y)
}

func (d *androidDriver) Capture(ctx context.Context) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	resolvedTarget, err := d.resolveLocked(ctx)
	if err != nil {
		return nil, err
	}
	resolved, err := controller.NewAndroidADBController(resolvedTarget, controller.AndroidADBDeps{Backend: AdapterKindAndroidADB})
	if err != nil {
		return nil, failure(CodeContractViolation, err)
	}
	frame, err := resolved.Screenshot(ctx, controller.ScreenshotRequest{Space: target.SpaceAndroidDevice})
	if err != nil {
		return nil, failure(CodeCaptureFailed, err)
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, frame.Image); err != nil {
		return nil, failure(CodeCaptureFailed, err)
	}
	if int64(encoded.Len()) > MaxCaptureBytes {
		return nil, failure(CodeCaptureFailed, errors.New("captured PNG exceeds byte budget"))
	}
	return encoded.Bytes(), nil
}

func (d *androidDriver) PlayEvent(context.Context, PlaybackEvent) error {
	return failure(CodePlaybackFailed, errors.New("android ADB does not support low-level recording playback"))
}

func (d *androidDriver) ReleaseInput() error { return nil }

func (d *androidDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	return nil
}
