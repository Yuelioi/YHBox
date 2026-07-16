package all

import (
	"slices"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

// TestBuiltInRuntimeCapabilityContract is the semantic guard for indirect
// dependencies hidden behind helpers such as ResolvePoint.
// Adding a service call to a built-in node requires updating its Spec and this
// reviewed matrix; otherwise a missing adapter can regress to a nil panic.
func TestBuiltInRuntimeCapabilityContract(t *testing.T) {
	expected := map[string][]node.RuntimeCapability{}
	add := func(capabilities []node.RuntimeCapability, kinds ...string) {
		for _, kind := range kinds {
			expected[kind] = capabilities
		}
	}
	add([]node.RuntimeCapability{node.RuntimeCapabilityWindow},
		"CloseWindow", "MoveResizeWindow", "WindowState", "Win32WindowTarget")
	add([]node.RuntimeCapability{node.RuntimeCapabilityVars},
		"Expr", "GetVar", "IncVar", "SetVar", "VarLastChange")
	add([]node.RuntimeCapability{node.RuntimeCapabilityLog},
		"Log", "ParseJSON", "RegexExtract", "RegexMatch", "ToJSON")
	add([]node.RuntimeCapability{node.RuntimeCapabilityStopwatches},
		"StopwatchRead", "StopwatchStart", "StopwatchStop")
	add([]node.RuntimeCapability{node.RuntimeCapabilityLog, node.RuntimeCapabilityParams, node.RuntimeCapabilityRegistry}, "Script")
	add([]node.RuntimeCapability{node.RuntimeCapabilityParams}, "GetParam")
	add([]node.RuntimeCapability{node.RuntimeCapabilityTarget}, "AndroidTarget")
	add([]node.RuntimeCapability{node.RuntimeCapabilityApp}, "AndroidStartApp", "AndroidStopApp")
	add([]node.RuntimeCapability{node.RuntimeCapabilityAI}, "AI")

	for _, registered := range node.All() {
		want := expected[registered.Spec.Kind]
		if !slices.Equal(registered.Spec.RuntimeCapabilities, want) {
			t.Errorf("%s RuntimeCapabilities = %v, want %v", registered.Spec.Kind, registered.Spec.RuntimeCapabilities, want)
		}
		delete(expected, registered.Spec.Kind)
	}
	for kind := range expected {
		t.Errorf("runtime capability contract references unregistered node %s", kind)
	}
}
