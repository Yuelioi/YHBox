package installed

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/adbexec"
	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
)

// AndroidDeviceDescriptor is discovery metadata used to fill target settings.
type AndroidDeviceDescriptor struct {
	Serial      string `json:"serial"`
	State       string `json:"state"`
	Product     string `json:"product"`
	Model       string `json:"model"`
	Device      string `json:"device"`
	TransportID string `json:"transportId"`
}

// AndroidAppDescriptor is install-time discovery metadata. The durable
// profile stores only Package; labels and foreground state are presentation.
type AndroidAppDescriptor struct {
	Package    string `json:"package"`
	Label      string `json:"label"`
	Foreground bool   `json:"foreground"`
}

type androidCommandRunner struct{}

func (androidCommandRunner) Run(ctx context.Context, serial string, args ...string) ([]byte, error) {
	command := make([]string, 0, len(args)+2)
	if serial != "" {
		command = append(command, "-s", serial)
	}
	command = append(command, args...)
	out, err := adbexec.CommandContext(ctx, command...).CombinedOutput()
	if err != nil {
		if message := strings.TrimSpace(string(out)); message != "" {
			return out, fmt.Errorf("%w: %s", err, message)
		}
	}
	return out, err
}

const defaultAndroidADBCommandTimeout = 10 * time.Second

type boundedADBRunner struct {
	runner  controller.ADBRunner
	timeout time.Duration
}

func (runner boundedADBRunner) Run(ctx context.Context, serial string, args ...string) ([]byte, error) {
	timeout := runner.timeout
	if timeout <= 0 {
		timeout = defaultAndroidADBCommandTimeout
	}
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return runner.runner.Run(commandCtx, serial, args...)
}

func adbCommandFailure(action string, out []byte, err error) error {
	message := strings.TrimSpace(string(out))
	if message == "" || strings.Contains(err.Error(), message) {
		return fmt.Errorf("%s: %w", action, err)
	}
	return fmt.Errorf("%s: %w: %s", action, err, message)
}

func DiscoverAndroidDevices(ctx context.Context) ([]AndroidDeviceDescriptor, error) {
	return discoverAndroidDevices(ctx, boundedADBRunner{runner: androidCommandRunner{}})
}

var commonADBEmulatorAddresses = []string{
	"127.0.0.1:16384",
	"127.0.0.1:7555",
	"127.0.0.1:5555",
	"127.0.0.1:62001",
	"127.0.0.1:21503",
}

func discoverAndroidDevices(ctx context.Context, runner controller.ADBRunner) ([]AndroidDeviceDescriptor, error) {
	out, err := runner.Run(ctx, "", "devices", "-l")
	if err != nil {
		return nil, adbCommandFailure("ADB discovery failed ("+adbexec.ExecutablePath()+")", out, err)
	}
	devices := parseADBDevices(string(out))
	if hasReadyAndroidDevice(devices) {
		return devices, nil
	}
	for _, address := range commonADBEmulatorAddresses {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		_, _ = runner.Run(ctx, "", "connect", address)
	}
	out, err = runner.Run(ctx, "", "devices", "-l")
	if err != nil {
		return nil, adbCommandFailure("ADB discovery after emulator reconnect failed ("+adbexec.ExecutablePath()+")", out, err)
	}
	return parseADBDevices(string(out)), nil
}

func DiscoverAndroidApps(ctx context.Context, serial string) ([]AndroidAppDescriptor, error) {
	return discoverAndroidApps(ctx, serial, boundedADBRunner{runner: androidCommandRunner{}})
}

func discoverAndroidApps(ctx context.Context, serial string, runner controller.ADBRunner) ([]AndroidAppDescriptor, error) {
	serial = strings.TrimSpace(serial)
	if !adbValuePattern.MatchString(serial) {
		return nil, errors.New("ADB application discovery requires an exact device serial")
	}
	foregroundOutput, _ := runner.Run(ctx, serial, "shell", "dumpsys", "window")
	foreground := parseAndroidForegroundPackage(string(foregroundOutput))
	thirdPartyOutput, err := runner.Run(ctx, serial, "shell", "pm", "list", "packages", "-3")
	if err != nil {
		return nil, adbCommandFailure(fmt.Sprintf("list ADB third-party applications on %q", serial), thirdPartyOutput, err)
	}
	thirdParty := parseAndroidPackageList(string(thirdPartyOutput))
	launcherOutput, launcherErr := runner.Run(ctx, serial, "shell", "cmd", "package", "query-activities", "-a", "android.intent.action.MAIN", "-c", "android.intent.category.LAUNCHER")
	apps := parseAndroidLauncherApps(string(launcherOutput))
	if launcherErr != nil || len(apps) == 0 {
		apps = make([]AndroidAppDescriptor, 0, len(thirdParty))
		for packageName := range thirdParty {
			apps = append(apps, AndroidAppDescriptor{Package: packageName, Label: packageName})
		}
	}
	byPackage := make(map[string]AndroidAppDescriptor, len(apps)+1)
	for _, app := range apps {
		if !androidPackagePattern.MatchString(app.Package) {
			continue
		}
		if _, installed := thirdParty[app.Package]; !installed && app.Package != foreground {
			continue
		}
		if app.Label == "" {
			app.Label = app.Package
		}
		app.Foreground = app.Package == foreground
		byPackage[app.Package] = app
	}
	if androidPackagePattern.MatchString(foreground) {
		app := byPackage[foreground]
		app.Package, app.Foreground = foreground, true
		if app.Label == "" {
			app.Label = foreground
		}
		byPackage[foreground] = app
	}
	result := make([]AndroidAppDescriptor, 0, len(byPackage))
	for _, app := range byPackage {
		result = append(result, app)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Foreground != result[j].Foreground {
			return result[i].Foreground
		}
		left, right := strings.ToLower(result[i].Label), strings.ToLower(result[j].Label)
		if left != right {
			return left < right
		}
		return result[i].Package < result[j].Package
	})
	if len(result) > 4096 {
		return nil, errors.New("ADB application discovery exceeds item budget")
	}
	return result, nil
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

func hasReadyAndroidDevice(devices []AndroidDeviceDescriptor) bool {
	for _, device := range devices {
		if device.State == "device" {
			return true
		}
	}
	return false
}

var androidForegroundPackagePattern = regexp.MustCompile(`\s([A-Za-z][A-Za-z0-9_]*(?:\.[A-Za-z0-9_]+)+)/`)

func parseAndroidForegroundPackage(out string) string {
	for _, line := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		if !strings.Contains(line, "mCurrentFocus") && !strings.Contains(line, "mFocusedApp") {
			continue
		}
		if match := androidForegroundPackagePattern.FindStringSubmatch(line); len(match) == 2 {
			return match[1]
		}
	}
	return ""
}

func parseAndroidPackageList(out string) map[string]struct{} {
	result := map[string]struct{}{}
	for _, line := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		packageName, ok := strings.CutPrefix(strings.TrimSpace(line), "package:")
		packageName = strings.TrimSpace(packageName)
		if ok && androidPackagePattern.MatchString(packageName) {
			result[packageName] = struct{}{}
		}
	}
	return result
}

func parseAndroidLauncherApps(out string) []AndroidAppDescriptor {
	byPackage := map[string]AndroidAppDescriptor{}
	current := ""
	for _, line := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Activity #") {
			current = ""
			continue
		}
		if packageName, ok := strings.CutPrefix(line, "packageName="); ok {
			current = strings.TrimSpace(packageName)
			if androidPackagePattern.MatchString(current) {
				byPackage[current] = AndroidAppDescriptor{Package: current, Label: current}
			} else {
				current = ""
			}
			continue
		}
		if label, ok := strings.CutPrefix(line, "nonLocalizedLabel="); ok && current != "" {
			label = strings.TrimSpace(label)
			if label != "" && label != "null" {
				app := byPackage[current]
				app.Label = label
				byPackage[current] = app
			}
		}
	}
	result := make([]AndroidAppDescriptor, 0, len(byPackage))
	for _, app := range byPackage {
		result = append(result, app)
	}
	return result
}

type androidDriver struct {
	profile        Profile
	mu             sync.Mutex
	closed         bool
	runner         controller.ADBRunner
	commandTimeout time.Duration
	playback       *androidPlaybackState
}

type androidPlaybackState struct {
	target  target.Target
	keys    map[uint32]struct{}
	button  string
	started target.Point
	current target.Point
}

// AndroidHealthProbe resolves a configured target for Settings diagnostics.
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
	devices, err := discoverAndroidDevices(resolveCtx, d.adbRunner())
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
	size, err := androidDeviceSize(resolveCtx, machine.ADBSerial, d.adbRunner())
	if err != nil {
		return target.Target{}, failure(CodeTargetNotFound, err)
	}
	return target.Target{
		ID: "android:" + machine.ADBSerial, Kind: target.KindAndroidADB, DisplayName: found.Model,
		Ref: target.TargetRef{ADBSerial: machine.ADBSerial}, Resolution: size,
		Metadata: map[string]any{"product": found.Product, "device": found.Device, "package": machine.AndroidPackage},
	}, nil
}

var androidSizePattern = regexp.MustCompile(`(?m)(?:Physical|Override) size:\s*(\d+)x(\d+)`)

func androidDeviceSize(ctx context.Context, serial string, runner controller.ADBRunner) (target.Size, error) {
	out, err := runner.Run(ctx, serial, "shell", "wm", "size")
	if err != nil {
		return target.Size{}, adbCommandFailure("read ADB device resolution", out, err)
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
	resolved, err := controller.NewAndroidADBController(resolvedTarget, controller.AndroidADBDeps{Runner: d.adbRunner(), Backend: AdapterKindAndroidADB})
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
	resolved, err := controller.NewAndroidADBController(resolvedTarget, controller.AndroidADBDeps{Runner: d.adbRunner(), Backend: AdapterKindAndroidADB})
	if err != nil {
		return nil, failure(CodeContractViolation, err)
	}
	frame, err := resolved.Screenshot(ctx, controller.ScreenshotRequest{Space: target.SpaceAndroidDevice})
	if err != nil {
		return nil, failure(CodeCaptureFailed, err)
	}
	encoded, err := encodeCapturePNG(frame.Image)
	if err != nil {
		return nil, failure(CodeCaptureFailed, err)
	}
	if int64(len(encoded)) > MaxCaptureBytes {
		return nil, failure(CodeCaptureFailed, errors.New("captured PNG exceeds byte budget"))
	}
	return encoded, nil
}

func (d *androidDriver) PlayEvent(ctx context.Context, event PlaybackEvent) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return failure(CodeContractViolation, errors.New("android ADB automation driver is closed"))
	}
	if d.playback == nil {
		resolved, err := d.resolveLocked(ctx)
		if err != nil {
			return err
		}
		d.playback = &androidPlaybackState{target: resolved, keys: map[uint32]struct{}{}}
	}
	state := d.playback
	control, err := controller.NewAndroidADBController(state.target, controller.AndroidADBDeps{Runner: d.adbRunner(), Backend: AdapterKindAndroidADB})
	if err != nil {
		return failure(CodeContractViolation, err)
	}
	switch event.Kind {
	case PlaybackKeyDown:
		if _, exists := state.keys[event.KeyCode]; exists {
			return failure(CodePlaybackFailed, errors.New("android playback key is already down"))
		}
		state.keys[event.KeyCode] = struct{}{}
		return nil
	case PlaybackKeyUp:
		if _, exists := state.keys[event.KeyCode]; !exists {
			return failure(CodePlaybackFailed, errors.New("android playback key-up has no matching key-down"))
		}
		delete(state.keys, event.KeyCode)
		if len(state.keys) != 0 {
			return failure(CodePlaybackFailed, errors.New("android ADB playback does not support recorded key chords"))
		}
		keyCode, ok := androidPlaybackKeyCode(event.KeyCode)
		if !ok {
			return failure(CodePlaybackFailed, fmt.Errorf("android ADB playback cannot map virtual key 0x%X", event.KeyCode))
		}
		_, err := d.runADB(ctx, state.target.Ref.ADBSerial, "shell", "input", "keyevent", strconv.Itoa(keyCode))
		if err != nil {
			return failure(CodePlaybackFailed, err)
		}
		return nil
	case PlaybackButtonDown:
		if event.Button != "left" || state.button != "" {
			return failure(CodePlaybackFailed, errors.New("android ADB playback supports one primary touch at a time"))
		}
		state.button = event.Button
		state.started = androidPoint(*event.Point)
		state.current = state.started
		return nil
	case PlaybackMove:
		if state.button != "" {
			state.current = androidPoint(*event.Point)
		}
		return nil
	case PlaybackButtonUp:
		if state.button == "" || event.Button != state.button {
			return failure(CodePlaybackFailed, errors.New("android playback button-up has no matching primary touch"))
		}
		started := state.started
		ended := androidPoint(*event.Point)
		state.button = ""
		state.started, state.current = target.Point{}, target.Point{}
		if sameAndroidPoint(started, ended) {
			err = control.Click(ctx, controller.ClickRequest{Point: ended, Button: "left", DurationMs: 10})
		} else {
			err = control.Drag(ctx, controller.DragRequest{From: started, To: ended, Button: "left", DurationMs: 100})
		}
	case PlaybackScroll:
		err = control.Scroll(ctx, controller.ScrollRequest{Point: androidPoint(*event.Point), Notches: int(event.Notches)})
	case PlaybackMoveRelative:
		return failure(CodePlaybackFailed, errors.New("relative mouse playback is not representable on an Android touch target"))
	default:
		return failure(CodeContractViolation, errors.New("android playback event is unsupported"))
	}
	if err != nil {
		return failure(CodePlaybackFailed, err)
	}
	return nil
}

func (d *androidDriver) OpenPlayback(ctx context.Context) (playbackSessionDriver, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	resolved, err := d.resolveLocked(ctx)
	if err != nil {
		return nil, err
	}
	d.playback = &androidPlaybackState{target: resolved, keys: map[uint32]struct{}{}}
	return d, nil
}

func (d *androidDriver) ReleaseInput() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.playback = nil
	return nil
}

func (d *androidDriver) runADB(ctx context.Context, serial string, args ...string) ([]byte, error) {
	return d.adbRunner().Run(ctx, serial, args...)
}

func (d *androidDriver) adbRunner() controller.ADBRunner {
	underlying := d.runner
	if underlying == nil {
		underlying = androidCommandRunner{}
	}
	timeout := d.commandTimeout
	if timeout <= 0 {
		timeout = defaultAndroidADBCommandTimeout
	}
	return boundedADBRunner{runner: underlying, timeout: timeout}
}

func sameAndroidPoint(left, right target.Point) bool {
	return left.Space == right.Space && left.X == right.X && left.Y == right.Y
}

func androidPlaybackKeyCode(virtualKey uint32) (int, bool) {
	if virtualKey >= 0x30 && virtualKey <= 0x39 {
		return 7 + int(virtualKey-0x30), true
	}
	if virtualKey >= 0x41 && virtualKey <= 0x5A {
		return 29 + int(virtualKey-0x41), true
	}
	if virtualKey >= 0x70 && virtualKey <= 0x7B {
		return 131 + int(virtualKey-0x70), true
	}
	keyCodes := map[uint32]int{
		0x08: 67, 0x09: 61, 0x0D: 66, 0x1B: 4, 0x20: 62,
		0x21: 92, 0x22: 93, 0x23: 123, 0x24: 122,
		0x25: 21, 0x26: 19, 0x27: 22, 0x28: 20,
		0x2D: 124, 0x2E: 112,
	}
	code, ok := keyCodes[virtualKey]
	return code, ok
}

func (d *androidDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	d.playback = nil
	return nil
}
