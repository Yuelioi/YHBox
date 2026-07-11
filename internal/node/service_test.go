// internal/node/service_test.go
package node

import "testing"

type serviceSpecNode struct {
	spec Spec
}

func (n serviceSpecNode) Spec() Spec { return n.spec }

func (n serviceSpecNode) Run(Ctx, Inputs) (Outputs, error) { return nil, nil }

func TestNodeService_GetAllNodeSpecs(t *testing.T) {
	registry := NewRegistry()
	registry.Register(stubNode{kind: "T1"})
	registry.Register(stubNode{kind: "T2"})

	svc := NewServiceWithRegistry(registry.Snapshot())
	specs := svc.GetAllNodeSpecs()
	if len(specs) != 2 {
		t.Errorf("len = %d, want 2", len(specs))
	}
}

func TestNodeServiceSnapshotsRegistryAtConstruction(t *testing.T) {
	registry := NewRegistry()
	registry.Register(stubNode{kind: "Before"})
	svc := NewServiceWithRegistry(registry)
	registry.Register(stubNode{kind: "After"})

	specs := svc.GetAllNodeSpecs()
	if len(specs) != 1 || specs[0].Kind != "Before" {
		t.Fatalf("service registry drifted after construction: %+v", specs)
	}
}

func TestNodeService_GetAllNodeSpecs_PopulatesSupportedTargets(t *testing.T) {
	registry := NewRegistry()
	registry.Register(serviceSpecNode{
		spec: Spec{
			Kind:               "ClickLike",
			Category:           "Input",
			NeedsTarget:        true,
			TargetCapabilities: []TargetCapability{TargetCapabilityClick},
		},
	})
	registry.Register(serviceSpecNode{
		spec: Spec{
			Kind:               "KeyStateLike",
			Category:           "Input",
			NeedsTarget:        true,
			TargetCapabilities: []TargetCapability{TargetCapabilityKeyState},
		},
	})
	registry.Register(serviceSpecNode{
		spec: Spec{
			Kind:        "WindowLike",
			Category:    "Window",
			NeedsWindow: true,
			Inputs:      []InputSpec{WindowInputSpec()},
		},
	})
	registry.Register(serviceSpecNode{
		spec: Spec{
			Kind:            "WindowsPlatformLike",
			Category:        "IO",
			PlatformTargets: []string{SupportedTargetWin32Window},
		},
	})
	registry.Register(serviceSpecNode{
		spec: Spec{Kind: "PureLike", Category: "Control"},
	})

	specs := NewServiceWithRegistry(registry.Snapshot()).GetAllNodeSpecs()
	byKind := map[string]Spec{}
	for _, spec := range specs {
		byKind[spec.Kind] = spec
	}

	assertTargets(t, byKind["ClickLike"].SupportedTargets, []string{"win32-window", "android-adb"})
	assertTargets(t, byKind["KeyStateLike"].SupportedTargets, []string{"win32-window"})
	assertTargets(t, byKind["WindowLike"].SupportedTargets, []string{"win32-window"})
	assertTargets(t, byKind["WindowsPlatformLike"].SupportedTargets, []string{"win32-window"})
	if byKind["WindowsPlatformLike"].NeedsWindow {
		t.Fatal("PlatformTargets must not imply NeedsWindow")
	}
	assertTargets(t, byKind["PureLike"].SupportedTargets, nil)
}

func assertTargets(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("targets = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("targets = %v, want %v", got, want)
		}
	}
}

func TestNodeService_AsyncOptions(t *testing.T) {
	svc := NewService()
	RegisterAsyncSource(svc, "test", func(nodeID, specKind string, params map[string]any) ([]EnumOption, error) {
		return []EnumOption{{Value: "x", Label: "X"}}, nil
	})

	opts, err := svc.AsyncOptions("n1", "Mock_Vision", "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) != 1 || opts[0].Value != "x" {
		t.Errorf("got %v", opts)
	}
}

func TestNodeService_AsyncOptions_UnknownSource(t *testing.T) {
	svc := NewService()
	_, err := svc.AsyncOptions("n1", "X", "unknown", nil)
	if err == nil {
		t.Error("expected error for unknown asyncSource")
	}
}

func TestNodeService_GetErrorCodes(t *testing.T) {
	svc := NewService()
	codes := svc.GetErrorCodes()
	if len(codes) != len(ErrorCodes) {
		t.Errorf("got %d codes, want %d", len(codes), len(ErrorCodes))
	}
	seen := map[ErrCode]bool{}
	for _, c := range codes {
		seen[c] = true
	}
	if !seen[CodeCaptureFailed] {
		t.Error("missing capture_failed")
	}
}
