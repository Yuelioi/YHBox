package androidadb

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/adbexec"
	"github.com/yottaapp/yotta/internal/automation/target"
)

type Runner interface {
	Run(ctx context.Context, serial string, args ...string) ([]byte, error)
}

type RunnerFunc func(ctx context.Context, serial string, args ...string) ([]byte, error)

func (f RunnerFunc) Run(ctx context.Context, serial string, args ...string) ([]byte, error) {
	return f(ctx, serial, args...)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, serial string, args ...string) ([]byte, error) {
	argv := make([]string, 0, len(args)+2)
	if strings.TrimSpace(serial) != "" {
		argv = append(argv, "-s", serial)
	}
	argv = append(argv, args...)
	out, err := adbexec.CommandContext(ctx, argv...).CombinedOutput()
	if err != nil && len(out) > 0 {
		return out, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return out, err
}

type Device struct {
	Serial     string
	State      string
	Product    string
	Model      string
	Device     string
	Resolution target.Size
}

type App struct {
	Package    string
	Label      string
	Foreground bool
}

type Service struct {
	Runner Runner
}

func NewService(runner Runner) *Service {
	return &Service{Runner: runner}
}

func (s *Service) ListDevices(ctx context.Context) ([]Device, error) {
	out, err := s.runner().Run(ctx, "", "devices", "-l")
	if err != nil {
		return nil, err
	}
	devices := ParseDevicesOutput(string(out))
	if !hasOnlineDevice(devices) {
		s.tryConnectCommonEmulators(ctx)
		if out, err = s.runner().Run(ctx, "", "devices", "-l"); err != nil {
			return nil, err
		}
		devices = ParseDevicesOutput(string(out))
	}
	for i := range devices {
		if devices[i].State != "device" {
			continue
		}
		devices[i].Resolution = s.currentResolution(ctx, devices[i].Serial)
	}
	return devices, nil
}

func (s *Service) ListApps(ctx context.Context, serial string) ([]App, error) {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		devices, err := s.ListDevices(ctx)
		if err != nil {
			return nil, err
		}
		for _, d := range devices {
			if d.State == "device" {
				serial = d.Serial
				break
			}
		}
	}
	if serial == "" {
		return nil, nil
	}
	currentOut, _ := s.runner().Run(ctx, serial, "shell", "dumpsys", "window")
	foregroundPackage := ParseCurrentFocusPackage(string(currentOut))

	thirdPartyOut, err := s.runner().Run(ctx, serial, "shell", "pm", "list", "packages", "-3")
	if err != nil {
		return nil, err
	}
	thirdParty := ParsePackageListOutput(string(thirdPartyOut))

	launcherOut, err := s.runner().Run(ctx, serial, "shell", "cmd", "package", "query-activities", "-a", "android.intent.action.MAIN", "-c", "android.intent.category.LAUNCHER")
	apps := ParseLauncherAppsOutput(string(launcherOut))
	if err != nil || len(apps) == 0 {
		apps = appsFromPackages(thirdParty)
	}
	filtered := make([]App, 0, len(apps))
	for _, app := range apps {
		if _, ok := thirdParty[app.Package]; !ok && app.Package != foregroundPackage {
			continue
		}
		if app.Package == foregroundPackage {
			app.Foreground = true
		}
		if app.Label == "" {
			app.Label = app.Package
		}
		filtered = append(filtered, app)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].Foreground != filtered[j].Foreground {
			return filtered[i].Foreground
		}
		if filtered[i].Label != filtered[j].Label {
			return filtered[i].Label < filtered[j].Label
		}
		return filtered[i].Package < filtered[j].Package
	})
	return filtered, nil
}

func appsFromPackages(packages map[string]struct{}) []App {
	apps := make([]App, 0, len(packages))
	for pkg := range packages {
		apps = append(apps, App{Package: pkg, Label: pkg})
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].Package < apps[j].Package })
	return apps
}

func (s *Service) currentResolution(ctx context.Context, serial string) target.Size {
	sizeOut, err := s.runner().Run(ctx, serial, "shell", "wm", "size")
	if err != nil {
		return target.Size{}
	}
	size, ok := ParseWMSizeOutput(string(sizeOut))
	if !ok {
		return target.Size{}
	}
	orientationOut, err := s.runner().Run(ctx, serial, "shell", "dumpsys", "input")
	if err != nil {
		return size
	}
	if orientation, ok := ParseSurfaceOrientation(string(orientationOut)); ok && orientation%2 != 0 {
		size.W, size.H = size.H, size.W
	}
	return size
}

func hasOnlineDevice(devices []Device) bool {
	for _, d := range devices {
		if d.State == "device" {
			return true
		}
	}
	return false
}

var commonEmulatorADBAddresses = []string{
	"127.0.0.1:16384", // MuMu 12
	"127.0.0.1:7555",  // MuMu classic / common Android emulator
	"127.0.0.1:5555",  // LDPlayer / common emulator default
	"127.0.0.1:62001", // Nox
	"127.0.0.1:21503", // MEmu
}

func (s *Service) tryConnectCommonEmulators(ctx context.Context) {
	for _, address := range commonEmulatorADBAddresses {
		if ctx.Err() != nil {
			return
		}
		_, _ = s.runner().Run(ctx, "", "connect", address)
	}
}

func (s *Service) runner() Runner {
	if s != nil && s.Runner != nil {
		return s.Runner
	}
	return execRunner{}
}

func ParseDevicesOutput(out string) []Device {
	lines := strings.Split(out, "\n")
	devices := make([]Device, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of devices attached") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		d := Device{Serial: fields[0], State: fields[1]}
		for _, field := range fields[2:] {
			key, value, ok := strings.Cut(field, ":")
			if !ok {
				continue
			}
			switch key {
			case "product":
				d.Product = value
			case "model":
				d.Model = value
			case "device":
				d.Device = value
			}
		}
		devices = append(devices, d)
	}
	return devices
}

var wmSizePattern = regexp.MustCompile(`(\d+)x(\d+)`)
var surfaceOrientationPattern = regexp.MustCompile(`SurfaceOrientation:\s*(\d+)`)
var currentFocusPackagePattern = regexp.MustCompile(`\s([A-Za-z][A-Za-z0-9_]*(?:\.[A-Za-z0-9_]+)+)/`)

func ParseWMSizeOutput(out string) (target.Size, bool) {
	var physical target.Size
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		size, ok := parseSizeLine(line)
		if !ok {
			continue
		}
		if strings.Contains(strings.ToLower(line), "override") {
			return size, true
		}
		if physical.W == 0 || strings.Contains(strings.ToLower(line), "physical") {
			physical = size
		}
	}
	if physical.W == 0 || physical.H == 0 {
		return target.Size{}, false
	}
	return physical, true
}

func parseSizeLine(line string) (target.Size, bool) {
	m := wmSizePattern.FindStringSubmatch(line)
	if len(m) != 3 {
		return target.Size{}, false
	}
	var w, h int
	if _, err := fmt.Sscanf(m[0], "%dx%d", &w, &h); err != nil {
		return target.Size{}, false
	}
	if w <= 0 || h <= 0 {
		return target.Size{}, false
	}
	return target.Size{W: w, H: h}, true
}

func ParseSurfaceOrientation(out string) (int, bool) {
	m := surfaceOrientationPattern.FindStringSubmatch(out)
	if len(m) != 2 {
		return 0, false
	}
	var orientation int
	if _, err := fmt.Sscanf(m[1], "%d", &orientation); err != nil {
		return 0, false
	}
	return orientation, true
}

func ParseCurrentFocusPackage(out string) string {
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "mCurrentFocus") && !strings.Contains(line, "mFocusedApp") {
			continue
		}
		if m := currentFocusPackagePattern.FindStringSubmatch(line); len(m) == 2 {
			return m[1]
		}
	}
	return ""
}

func ParsePackageListOutput(out string) map[string]struct{} {
	packages := map[string]struct{}{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		pkg, ok := strings.CutPrefix(line, "package:")
		if !ok {
			continue
		}
		pkg = strings.TrimSpace(pkg)
		if pkg != "" {
			packages[pkg] = struct{}{}
		}
	}
	return packages
}

func ParseLauncherAppsOutput(out string) []App {
	byPackage := map[string]App{}
	currentPackage := ""
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Activity #") {
			currentPackage = ""
			continue
		}
		if pkg, ok := strings.CutPrefix(line, "packageName="); ok {
			currentPackage = strings.TrimSpace(pkg)
			if currentPackage != "" {
				app := byPackage[currentPackage]
				app.Package = currentPackage
				byPackage[currentPackage] = app
			}
			continue
		}
		if label, ok := strings.CutPrefix(line, "nonLocalizedLabel="); ok && currentPackage != "" {
			label = strings.TrimSpace(label)
			if label == "" || label == "null" {
				continue
			}
			app := byPackage[currentPackage]
			app.Package = currentPackage
			app.Label = label
			byPackage[currentPackage] = app
		}
	}
	apps := make([]App, 0, len(byPackage))
	for _, app := range byPackage {
		apps = append(apps, app)
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].Package < apps[j].Package })
	return apps
}
