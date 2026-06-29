package androidadb

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"yotta/internal/adbexec"
	"yotta/internal/automation/target"
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
		sizeOut, err := s.runner().Run(ctx, devices[i].Serial, "shell", "wm", "size")
		if err != nil {
			continue
		}
		if size, ok := ParseWMSizeOutput(string(sizeOut)); ok {
			devices[i].Resolution = size
		}
	}
	return devices, nil
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
