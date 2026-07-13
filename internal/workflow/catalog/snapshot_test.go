package catalog

import (
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/node"
)

type snapshotTestNode struct{ spec node.Spec }

func (n snapshotTestNode) Spec() node.Spec                               { return n.spec }
func (snapshotTestNode) Run(node.Ctx, node.Inputs) (node.Outputs, error) { return nil, nil }

func TestSnapshotIsDeterministicAndExcludesPresentationMetadata(t *testing.T) {
	leftRegistry := node.NewRegistry()
	leftRegistry.Register(snapshotTestNode{spec: snapshotSpec("B", "second")})
	leftRegistry.Register(snapshotTestNode{spec: snapshotSpec("A", "first")})
	rightRegistry := node.NewRegistry()
	rightRegistry.Register(snapshotTestNode{spec: snapshotSpec("A", "changed presentation")})
	rightRegistry.Register(snapshotTestNode{spec: snapshotSpec("B", "also changed")})

	left, err := NewSnapshot(leftRegistry.Snapshot(), implementationSet(t, "test"))
	if err != nil {
		t.Fatal(err)
	}
	right, err := NewSnapshot(rightRegistry.Snapshot(), implementationSet(t, "test"))
	if err != nil {
		t.Fatal(err)
	}
	if left.Hash() != right.Hash() || string(left.Bytes()) != string(right.Bytes()) {
		t.Fatalf("catalog identity depends on registration order or presentation: %s != %s", left.Hash(), right.Hash())
	}
	if got := left.All(); len(got) != 2 || got[0].Kind != "A" || got[1].Kind != "B" {
		t.Fatalf("contracts = %#v", got)
	}
}

func TestSnapshotIdentityChangesWithMachineContractOrImplementation(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(snapshotTestNode{spec: snapshotSpec("A", "presentation")})
	base, err := NewSnapshot(registry.Snapshot(), implementationSet(t, "one"))
	if err != nil {
		t.Fatal(err)
	}
	otherBuild, err := NewSnapshot(registry.Snapshot(), implementationSet(t, "two"))
	if err != nil {
		t.Fatal(err)
	}
	changedRegistry := node.NewRegistry()
	changed := snapshotSpec("A", "presentation")
	changed.Inputs[0].Required = false
	changedRegistry.Register(snapshotTestNode{spec: changed})
	otherContract, err := NewSnapshot(changedRegistry.Snapshot(), implementationSet(t, "one"))
	if err != nil {
		t.Fatal(err)
	}
	if base.Hash() == otherBuild.Hash() || base.Hash() == otherContract.Hash() {
		t.Fatal("catalog identity did not change")
	}
}

func TestSnapshotIdentityIncludesDeclarativeDynamicPorts(t *testing.T) {
	leftSpec := snapshotSpec("A", "presentation")
	leftSpec.DynamicPorts = []node.DynamicPortSpec{{
		Role: node.DynamicPortOutput, ConfigKey: "cases", Shape: node.DynamicPortNames,
		FixedType: node.TypeExec, MinItems: 1, MaxItems: 8,
	}}
	rightSpec := snapshotSpec("A", "presentation")
	rightSpec.DynamicPorts = []node.DynamicPortSpec{{
		Role: node.DynamicPortOutput, ConfigKey: "branches", Shape: node.DynamicPortNames,
		FixedType: node.TypeExec, MinItems: 1, MaxItems: 8,
	}}
	leftRegistry, rightRegistry := node.NewRegistry(), node.NewRegistry()
	leftRegistry.Register(snapshotTestNode{spec: leftSpec})
	rightRegistry.Register(snapshotTestNode{spec: rightSpec})
	left, leftErr := NewSnapshot(leftRegistry.Snapshot(), implementationSet(t, "test"))
	right, rightErr := NewSnapshot(rightRegistry.Snapshot(), implementationSet(t, "test"))
	if leftErr != nil || rightErr != nil {
		t.Fatalf("snapshot errors: left=%v right=%v", leftErr, rightErr)
	}
	if left.Hash() == right.Hash() {
		t.Fatal("dynamic port derivation did not affect catalog identity")
	}
}

func TestSnapshotIdentityIncludesInputConstraints(t *testing.T) {
	leftSpec := snapshotSpec("A", "presentation")
	leftSpec.Inputs[0].Constraints = []node.InputConstraint{{
		Kind: node.InputConstraintNumberGreaterThan, Threshold: "0",
	}}
	rightSpec := snapshotSpec("A", "presentation")
	rightSpec.Inputs[0].Constraints = []node.InputConstraint{{
		Kind: node.InputConstraintNumberGreaterThan, Threshold: "-1",
	}}
	leftRegistry, rightRegistry := node.NewRegistry(), node.NewRegistry()
	leftRegistry.Register(snapshotTestNode{spec: leftSpec})
	rightRegistry.Register(snapshotTestNode{spec: rightSpec})
	left, leftErr := NewSnapshot(leftRegistry.Snapshot(), implementationSet(t, "test"))
	right, rightErr := NewSnapshot(rightRegistry.Snapshot(), implementationSet(t, "test"))
	if leftErr != nil || rightErr != nil {
		t.Fatalf("snapshot errors: left=%v right=%v", leftErr, rightErr)
	}
	if left.Hash() == right.Hash() {
		t.Fatal("input constraints did not affect catalog identity")
	}
}

func TestSnapshotRejectsInputDefaultViolatingConstraint(t *testing.T) {
	spec := snapshotSpec("A", "presentation")
	spec.Inputs[0].Default = 0
	spec.Inputs[0].Constraints = []node.InputConstraint{{
		Kind: node.InputConstraintNumberGreaterThan, Threshold: "0",
	}}
	entry := &node.RegisteredNode{Spec: spec, Run: func(node.Ctx, node.Inputs) (node.Outputs, error) { return nil, nil }}
	if _, err := NewSnapshot(malformedReader{entry}, implementationSet(t, "test")); err == nil {
		t.Fatal("accepted input default that violates its constraint")
	}
}

func TestSnapshotCanonicalizesEquivalentInputConstraintThresholds(t *testing.T) {
	leftSpec := snapshotSpec("A", "presentation")
	leftSpec.Inputs[0].Constraints = []node.InputConstraint{{
		Kind: node.InputConstraintNumberGreaterThan, Threshold: "0",
	}}
	rightSpec := snapshotSpec("A", "presentation")
	rightSpec.Inputs[0].Constraints = []node.InputConstraint{{
		Kind: node.InputConstraintNumberGreaterThan, Threshold: "0.0",
	}}
	leftRegistry, rightRegistry := node.NewRegistry(), node.NewRegistry()
	leftRegistry.Register(snapshotTestNode{spec: leftSpec})
	rightRegistry.Register(snapshotTestNode{spec: rightSpec})
	left, _ := NewSnapshot(leftRegistry.Snapshot(), implementationSet(t, "test"))
	right, _ := NewSnapshot(rightRegistry.Snapshot(), implementationSet(t, "test"))
	if left.Hash() != right.Hash() {
		t.Fatal("equivalent numeric thresholds changed catalog identity")
	}
}

func TestSnapshotRejectsMalformedInputConstraints(t *testing.T) {
	tests := []struct {
		name        string
		inputType   string
		constraints []node.InputConstraint
	}{
		{"unknown kind", "Number", []node.InputConstraint{{Kind: "unknown"}}},
		{"non blank on number", "Number", []node.InputConstraint{{Kind: node.InputConstraintNonBlank}}},
		{"non blank threshold", "String", []node.InputConstraint{{Kind: node.InputConstraintNonBlank, Threshold: "0"}}},
		{"missing numeric threshold", "Number", []node.InputConstraint{{Kind: node.InputConstraintNumberGreaterThan}}},
		{"invalid numeric threshold", "Number", []node.InputConstraint{{Kind: node.InputConstraintNumberGreaterThan, Threshold: "NaN"}}},
		{"numeric constraint on string", "String", []node.InputConstraint{{Kind: node.InputConstraintNumberGreaterThan, Threshold: "0"}}},
		{"duplicate kind", "Number", []node.InputConstraint{
			{Kind: node.InputConstraintNumberGreaterThan, Threshold: "0"},
			{Kind: node.InputConstraintNumberGreaterThan, Threshold: "1"},
		}},
		{"constraint budget", "Number", make([]node.InputConstraint, node.MaxInputConstraints+1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := snapshotSpec("A", "presentation")
			spec.Inputs[0].Type = test.inputType
			spec.Inputs[0].Constraints = test.constraints
			entry := &node.RegisteredNode{Spec: spec, Run: func(node.Ctx, node.Inputs) (node.Outputs, error) { return nil, nil }}
			if _, err := NewSnapshot(malformedReader{entry}, implementationSet(t, "test")); err == nil {
				t.Fatalf("accepted malformed constraints %#v", test.constraints)
			}
		})
	}
}

func TestSnapshotCanonicalizesDynamicPortOrder(t *testing.T) {
	leftSpec := snapshotSpec("A", "presentation")
	leftSpec.DynamicPorts = []node.DynamicPortSpec{
		{Role: node.DynamicPortInput, ConfigKey: "Inputs", Shape: node.DynamicPortNameTypeRecords, MaxItems: 8},
		{Role: node.DynamicPortOutputData, ConfigKey: "Outputs", Shape: node.DynamicPortNameTypeRecords, ParentOutput: "Next", MaxItems: 8},
	}
	rightSpec := snapshotSpec("A", "presentation")
	rightSpec.DynamicPorts = []node.DynamicPortSpec{leftSpec.DynamicPorts[1], leftSpec.DynamicPorts[0]}
	leftRegistry, rightRegistry := node.NewRegistry(), node.NewRegistry()
	leftRegistry.Register(snapshotTestNode{spec: leftSpec})
	rightRegistry.Register(snapshotTestNode{spec: rightSpec})
	left, leftErr := NewSnapshot(leftRegistry.Snapshot(), implementationSet(t, "test"))
	right, rightErr := NewSnapshot(rightRegistry.Snapshot(), implementationSet(t, "test"))
	if leftErr != nil || rightErr != nil {
		t.Fatalf("snapshot errors: left=%v right=%v", leftErr, rightErr)
	}
	if left.Hash() != right.Hash() {
		t.Fatal("dynamic port declaration order changed catalog identity")
	}
}

func TestSnapshotRejectsMalformedDynamicPorts(t *testing.T) {
	tests := []node.DynamicPortSpec{
		{Role: node.DynamicPortOutput, ConfigKey: "cases", Shape: node.DynamicPortNames, FixedType: node.TypeExec, MinItems: 9, MaxItems: 8},
		{Role: node.DynamicPortOutput, ConfigKey: "cases", Shape: node.DynamicPortNames, FixedType: node.TypeExec, MaxItems: MaxNodePins},
		{Role: node.DynamicPortOutputData, ConfigKey: "Outputs", Shape: node.DynamicPortNameTypeRecords, ParentOutput: "missing", MaxItems: 8},
		{Role: node.DynamicPortOutput, ConfigKey: "cases", Shape: node.DynamicPortNames, FixedType: node.TypeExec, MaxItems: int(^uint(0) >> 1)},
	}
	for _, dynamic := range tests {
		spec := snapshotSpec("A", "presentation")
		spec.DynamicPorts = []node.DynamicPortSpec{dynamic}
		entry := &node.RegisteredNode{Spec: spec, Run: func(node.Ctx, node.Inputs) (node.Outputs, error) { return nil, nil }}
		if _, err := NewSnapshot(malformedReader{entry}, implementationSet(t, "test")); err == nil {
			t.Fatalf("accepted malformed dynamic port contract %#v", dynamic)
		}
	}
}

func TestSnapshotAccessorsAreDefensive(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(snapshotTestNode{spec: snapshotSpec("A", "presentation")})
	snapshot, err := NewSnapshot(registry.Snapshot(), implementationSet(t, "test"))
	if err != nil {
		t.Fatal(err)
	}
	original := append([]byte(nil), snapshot.Bytes()...)
	contract, ok := snapshot.Lookup("A")
	if !ok {
		t.Fatal("missing contract")
	}
	contract.Inputs[0].Name = "mutated"
	contract.TargetCapabilities[0] = "mutated"
	bytes := snapshot.Bytes()
	bytes[0] = 'x'
	again, _ := snapshot.Lookup("A")
	if again.Inputs[0].Name == "mutated" || again.TargetCapabilities[0] == "mutated" || string(snapshot.Bytes()) != string(original) {
		t.Fatal("caller mutated immutable catalog snapshot")
	}
}

func TestSnapshotRejectsAmbiguousRegistryReaders(t *testing.T) {
	entry := &node.RegisteredNode{Spec: snapshotSpec("A", "presentation"), Run: func(node.Ctx, node.Inputs) (node.Outputs, error) { return nil, nil }}
	for _, reader := range []malformedReader{{nil}, {entry, entry}} {
		if _, err := NewSnapshot(reader, implementationSet(t, "test")); err == nil {
			t.Fatalf("accepted malformed reader %#v", reader)
		}
	}
	duplicateCapability := snapshotSpec("A", "presentation")
	duplicateCapability.TargetCapabilities = append(duplicateCapability.TargetCapabilities, duplicateCapability.TargetCapabilities[0])
	entry = &node.RegisteredNode{Spec: duplicateCapability, Run: func(node.Ctx, node.Inputs) (node.Outputs, error) { return nil, nil }}
	if _, err := NewSnapshot(malformedReader{entry}, implementationSet(t, "test")); err == nil {
		t.Fatal("accepted duplicate capability")
	}
	if _, err := NewSnapshot(malformedReader{}, "not-a-digest"); err == nil {
		t.Fatal("accepted unverified implementation identity")
	}
	reserved := &node.RegisteredNode{Spec: snapshotSpec("core.call-subgraph", "presentation"), Run: func(node.Ctx, node.Inputs) (node.Outputs, error) { return nil, nil }}
	if _, err := NewSnapshot(malformedReader{reserved}, implementationSet(t, "test")); err == nil {
		t.Fatal("accepted compiler intrinsic in registry catalog")
	}
}

func TestSnapshotHashIncludesMachineSemantic(t *testing.T) {
	leftRegistry := node.NewRegistry()
	leftRegistry.Register(snapshotTestNode{spec: snapshotSpec("A", "presentation")})
	rightRegistry := node.NewRegistry()
	changed := snapshotSpec("A", "presentation")
	changed.Inputs[0].Semantic = "TemplateKey"
	rightRegistry.Register(snapshotTestNode{spec: changed})
	left, _ := NewSnapshot(leftRegistry.Snapshot(), implementationSet(t, "test"))
	right, _ := NewSnapshot(rightRegistry.Snapshot(), implementationSet(t, "test"))
	if left.Hash() == right.Hash() {
		t.Fatal("machine semantic did not affect catalog identity")
	}
}

func TestSnapshotExcludesNestedFieldPresentationMetadata(t *testing.T) {
	leftSpec := snapshotSpec("A", "presentation")
	leftSpec.Inputs[0].Schema = &node.FieldSchema{Type: "object", Widget: "presentation-a"}
	rightSpec := snapshotSpec("A", "presentation")
	rightSpec.Inputs[0].Schema = &node.FieldSchema{Type: "object", Widget: "presentation-b"}
	leftRegistry, rightRegistry := node.NewRegistry(), node.NewRegistry()
	leftRegistry.Register(snapshotTestNode{spec: leftSpec})
	rightRegistry.Register(snapshotTestNode{spec: rightSpec})
	left, _ := NewSnapshot(leftRegistry.Snapshot(), implementationSet(t, "test"))
	right, _ := NewSnapshot(rightRegistry.Snapshot(), implementationSet(t, "test"))
	if left.Hash() != right.Hash() {
		t.Fatal("field widget leaked into machine catalog identity")
	}
}

func TestSnapshotPreservesMachineFieldShape(t *testing.T) {
	leftSpec, rightSpec := snapshotSpec("A", "presentation"), snapshotSpec("A", "presentation")
	leftSpec.Inputs[0].Schema, rightSpec.Inputs[0].Schema = node.PointSchema(), node.GeometrySchema()
	leftRegistry, rightRegistry := node.NewRegistry(), node.NewRegistry()
	leftRegistry.Register(snapshotTestNode{spec: leftSpec})
	rightRegistry.Register(snapshotTestNode{spec: rightSpec})
	left, _ := NewSnapshot(leftRegistry.Snapshot(), implementationSet(t, "test"))
	right, _ := NewSnapshot(rightRegistry.Snapshot(), implementationSet(t, "test"))
	if left.Hash() == right.Hash() {
		t.Fatal("point and geometry collapsed to one machine contract")
	}
}

func TestSnapshotRejectsMalformedOrExcessiveFieldSchema(t *testing.T) {
	for _, field := range []*node.FieldSchema{{Type: "array"}, deepFieldSchema(MaxFieldSchemaDepth + 2)} {
		spec := snapshotSpec("A", "presentation")
		spec.Inputs[0].Schema = field
		entry := &node.RegisteredNode{Spec: spec, Run: func(node.Ctx, node.Inputs) (node.Outputs, error) { return nil, nil }}
		if _, err := NewSnapshot(malformedReader{entry}, implementationSet(t, "test")); err == nil {
			t.Fatalf("accepted schema %#v", field)
		}
	}
}

func deepFieldSchema(depth int) *node.FieldSchema {
	field := node.NumberSchema()
	for range depth {
		field = node.ArraySchema(field)
	}
	return field
}

type malformedReader []*node.RegisteredNode

func (m malformedReader) Get(string) (*node.RegisteredNode, bool) { return nil, false }
func (m malformedReader) All() []*node.RegisteredNode             { return m }

func implementationSet(t testing.TB, value string) artifact.Digest {
	t.Helper()
	digest, err := artifact.Sum("yotta/test-implementation/v1", []byte(value))
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func snapshotSpec(kind, category string) node.Spec {
	return node.Spec{
		Kind: kind, Category: category,
		Inputs:              []node.InputSpec{{Name: "Value", Type: "Number", Required: true, Default: 1, Widget: node.WidgetSpec{Kind: "slider"}}},
		Outputs:             []node.OutputSpec{{Name: "Next", Type: node.TypeExec}},
		NeedsTarget:         true,
		TargetCapabilities:  []node.TargetCapability{node.TargetCapabilityClick},
		RuntimeCapabilities: []node.RuntimeCapability{node.RuntimeCapabilityLog},
	}
}
