package compiler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"sort"

	"github.com/yottaapp/yotta/internal/artifact"
	nodespec "github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/workflow/catalog"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

const (
	ProgramFormat       = "yotta.program"
	ProgramVersion      = 3
	MaxProgramBytes     = 16 << 20
	MaxProgramJSONDepth = 128
	programHashDomain   = "yotta/program/v3"
)

var (
	ErrInvalidProgramArtifact = errors.New("invalid program artifact")
	ErrNonCanonicalProgram    = errors.New("non-canonical program artifact")
	ErrProgramHashMismatch    = errors.New("program hash mismatch")
	ErrProgramTooLarge        = errors.New("program artifact exceeds size limit")
	ErrProgramTooDeep         = errors.New("program artifact exceeds JSON depth limit")
	ErrCatalogHashMismatch    = errors.New("program catalog hash mismatch")
	ErrCompilerBuildMismatch  = errors.New("program compiler build mismatch")
	ErrNodeLockMismatch       = errors.New("program node lock mismatch")
	ErrCapabilityMismatch     = errors.New("program capability manifest mismatch")
)

type NodeLock struct {
	Kind         string          `json:"kind"`
	ContractHash artifact.Digest `json:"contractHash"`
}

type programNode struct {
	ID           string         `json:"id"`
	Kind         string         `json:"kind"`
	Config       map[string]any `json:"config"`
	Disabled     bool           `json:"disabled"`
	Call         *programCall   `json:"call"`
	DynamicPorts []programPort  `json:"dynamicPorts"`
}

type programPort struct {
	Role         nodespec.DynamicPortRole `json:"role"`
	Name         string                   `json:"name"`
	Type         string                   `json:"type"`
	ParentOutput string                   `json:"parentOutput"`
}

type programCall struct {
	GraphID    string            `json:"graphId"`
	Entry      programCallPort   `json:"entry"`
	Inputs     []programCallPort `json:"inputs"`
	Outputs    []programCallPort `json:"outputs"`
	FailurePin string            `json:"failurePin"`
}

type programCallPort struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	NodeID string `json:"nodeId"`
}

type programGraph struct {
	ID      string             `json:"id"`
	Kind    schema.GraphKind   `json:"kind"`
	Nodes   []programNode      `json:"nodes"`
	Edges   []schema.Edge      `json:"edges"`
	Inputs  []schema.GraphPort `json:"inputs"`
	Outputs []schema.GraphPort `json:"outputs"`
}

type programBody struct {
	SourceHash            artifact.Digest    `json:"sourceHash"`
	CatalogHash           artifact.Digest    `json:"catalogHash"`
	CompilerBuild         artifact.Digest    `json:"compilerBuild"`
	ImplementationSet     artifact.Digest    `json:"implementationSet"`
	WorkflowID            string             `json:"workflowId"`
	Revision              int64              `json:"revision"`
	EntryGraph            string             `json:"entryGraph"`
	NodeLocks             []NodeLock         `json:"nodeLocks"`
	RequestedCapabilities []string           `json:"requestedCapabilities"`
	RequiredCapabilities  []string           `json:"requiredCapabilities"`
	Variables             []schema.Variable  `json:"variables"`
	SecretRefs            []schema.SecretRef `json:"secretRefs"`
	Graphs                []programGraph     `json:"graphs"`
}

type programEnvelope struct {
	Format      string          `json:"format"`
	Version     int             `json:"version"`
	ProgramHash artifact.Digest `json:"programHash"`
	Program     programBody     `json:"program"`
}

type programState struct {
	envelope programEnvelope
	artifact []byte
}

// ProgramSnapshot is an immutable, sealed compiler output. Its zero value is
// invalid and no public constructor can assemble one from mutable graph data.
type ProgramSnapshot struct{ state *programState }

func (p ProgramSnapshot) Valid() bool {
	return p.state != nil && p.state.envelope.ProgramHash.Valid()
}

func (p ProgramSnapshot) Hash() artifact.Digest {
	if !p.Valid() {
		return ""
	}
	return p.state.envelope.ProgramHash
}

func (p ProgramSnapshot) SourceHash() artifact.Digest {
	if !p.Valid() {
		return ""
	}
	return p.state.envelope.Program.SourceHash
}

func (p ProgramSnapshot) CatalogHash() artifact.Digest {
	if !p.Valid() {
		return ""
	}
	return p.state.envelope.Program.CatalogHash
}

func (p ProgramSnapshot) CompilerBuild() artifact.Digest {
	if !p.Valid() {
		return ""
	}
	return p.state.envelope.Program.CompilerBuild
}

func (p ProgramSnapshot) RequestedCapabilities() []string {
	if !p.Valid() {
		return nil
	}
	return append([]string(nil), p.state.envelope.Program.RequestedCapabilities...)
}

func (p ProgramSnapshot) NodeLocks() []NodeLock {
	if !p.Valid() {
		return nil
	}
	return append([]NodeLock(nil), p.state.envelope.Program.NodeLocks...)
}

func (p ProgramSnapshot) RequiredCapabilities() []string {
	if !p.Valid() {
		return nil
	}
	return append([]string(nil), p.state.envelope.Program.RequiredCapabilities...)
}

func (p ProgramSnapshot) Artifact() []byte {
	if !p.Valid() {
		return nil
	}
	return append([]byte(nil), p.state.artifact...)
}

func sealProgram(body programBody) (ProgramSnapshot, error) {
	canonicalBody, err := artifact.Marshal(body)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	hash, err := artifact.Sum(programHashDomain, canonicalBody)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	envelope := programEnvelope{Format: ProgramFormat, Version: ProgramVersion, ProgramHash: hash, Program: body}
	if err := validateEnvelope(envelope); err != nil {
		return ProgramSnapshot{}, fmt.Errorf("compiler produced invalid program: %w", err)
	}
	canonical, err := artifact.Marshal(envelope)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	if len(canonical) > MaxProgramBytes {
		return ProgramSnapshot{}, ErrProgramTooLarge
	}
	if exceedsJSONDepth(canonical, MaxProgramJSONDepth) {
		return ProgramSnapshot{}, ErrProgramTooDeep
	}
	return ProgramSnapshot{state: &programState{envelope: envelope, artifact: canonical}}, nil
}

// OpenProgram verifies canonical encoding, the integrity seal and structural
// invariants before returning an opaque snapshot.
func OpenProgram(raw []byte, trustedCatalog catalog.Snapshot, expectedCompilerBuild artifact.Digest) (ProgramSnapshot, error) {
	if len(raw) == 0 {
		return ProgramSnapshot{}, fmt.Errorf("%w: empty", ErrInvalidProgramArtifact)
	}
	if len(raw) > MaxProgramBytes {
		return ProgramSnapshot{}, ErrProgramTooLarge
	}
	if exceedsJSONDepth(raw, MaxProgramJSONDepth) {
		return ProgramSnapshot{}, ErrProgramTooDeep
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil {
		return ProgramSnapshot{}, fmt.Errorf("%w: %v", ErrInvalidProgramArtifact, err)
	}
	if !bytes.Equal(raw, canonical) {
		return ProgramSnapshot{}, ErrNonCanonicalProgram
	}
	if err := validateRequiredFields(raw); err != nil {
		return ProgramSnapshot{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var envelope programEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return ProgramSnapshot{}, fmt.Errorf("%w: %v", ErrInvalidProgramArtifact, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return ProgramSnapshot{}, fmt.Errorf("%w: trailing value", ErrInvalidProgramArtifact)
	}
	if err := validateEnvelope(envelope); err != nil {
		return ProgramSnapshot{}, err
	}
	bodyBytes, err := artifact.Marshal(envelope.Program)
	if err != nil {
		return ProgramSnapshot{}, fmt.Errorf("%w: %v", ErrInvalidProgramArtifact, err)
	}
	want, err := artifact.Sum(programHashDomain, bodyBytes)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	if envelope.ProgramHash != want {
		return ProgramSnapshot{}, ErrProgramHashMismatch
	}
	if err := validateTrustedBindings(envelope.Program, trustedCatalog, expectedCompilerBuild); err != nil {
		return ProgramSnapshot{}, err
	}
	return ProgramSnapshot{state: &programState{envelope: envelope, artifact: append([]byte(nil), raw...)}}, nil
}

func validateRequiredFields(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidProgramArtifact, err)
	}
	root, ok := document.(map[string]any)
	if !ok || !hasFields(root, "format", "version", "programHash", "program") {
		return fmt.Errorf("%w: missing envelope field", ErrInvalidProgramArtifact)
	}
	body, ok := root["program"].(map[string]any)
	if !ok || !hasFields(body, "sourceHash", "catalogHash", "compilerBuild", "implementationSet", "workflowId", "revision", "entryGraph", "nodeLocks", "requestedCapabilities", "requiredCapabilities", "variables", "secretRefs", "graphs") {
		return fmt.Errorf("%w: missing program field", ErrInvalidProgramArtifact)
	}
	for _, lock := range asObjects(body["nodeLocks"]) {
		if !hasFields(lock, "kind", "contractHash") {
			return fmt.Errorf("%w: missing node lock field", ErrInvalidProgramArtifact)
		}
	}
	for _, variable := range asObjects(body["variables"]) {
		if !hasFields(variable, "name", "type") {
			return fmt.Errorf("%w: missing variable field", ErrInvalidProgramArtifact)
		}
	}
	for _, secret := range asObjects(body["secretRefs"]) {
		if !hasFields(secret, "id", "purpose") {
			return fmt.Errorf("%w: missing secret field", ErrInvalidProgramArtifact)
		}
	}
	for _, graph := range asObjects(body["graphs"]) {
		if !hasFields(graph, "id", "kind", "nodes", "edges", "inputs", "outputs") {
			return fmt.Errorf("%w: missing graph field", ErrInvalidProgramArtifact)
		}
		for _, node := range asObjects(graph["nodes"]) {
			if !hasFields(node, "id", "kind", "config", "disabled", "call", "dynamicPorts") {
				return fmt.Errorf("%w: missing node field", ErrInvalidProgramArtifact)
			}
			for _, port := range asObjects(node["dynamicPorts"]) {
				if !hasFields(port, "role", "name", "type", "parentOutput") {
					return fmt.Errorf("%w: missing dynamic port field", ErrInvalidProgramArtifact)
				}
			}
		}
		for _, edge := range asObjects(graph["edges"]) {
			if !hasFields(edge, "from", "to") {
				return fmt.Errorf("%w: missing edge field", ErrInvalidProgramArtifact)
			}
		}
		for _, name := range []string{"inputs", "outputs"} {
			for _, port := range asObjects(graph[name]) {
				if !hasFields(port, "id", "name", "type", "nodeId") {
					return fmt.Errorf("%w: missing graph port field", ErrInvalidProgramArtifact)
				}
			}
		}
	}
	return nil
}

func hasFields(object map[string]any, fields ...string) bool {
	for _, field := range fields {
		if _, ok := object[field]; !ok {
			return false
		}
	}
	return true
}

func asObjects(value any) []map[string]any {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(values))
	for _, value := range values {
		object, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		out = append(out, object)
	}
	return out
}

func validateTrustedBindings(body programBody, snapshot catalog.Snapshot, expectedBuild artifact.Digest) error {
	if !snapshot.Valid() || body.CatalogHash != snapshot.Hash() || body.ImplementationSet != snapshot.ImplementationSet() {
		return ErrCatalogHashMismatch
	}
	if !expectedBuild.Valid() || body.CompilerBuild != expectedBuild {
		return ErrCompilerBuildMismatch
	}
	source := sourceFromProgram(body)
	binding, err := bindSource(context.Background(), source, snapshot)
	if err != nil {
		return err
	}
	if len(binding.diagnostics) > 0 {
		return fmt.Errorf("%w: compiler contract validation failed", ErrInvalidProgramArtifact)
	}
	if !equalLocks(binding.locks, body.NodeLocks) {
		return ErrNodeLockMismatch
	}
	if len(validateCapabilityDeclaration(body.RequestedCapabilities, binding.capabilities)) != 0 {
		return ErrCapabilityMismatch
	}
	if !slices.Equal(binding.capabilities, body.RequiredCapabilities) {
		return ErrCapabilityMismatch
	}
	if !reflect.DeepEqual(lowerGraphs(source.Graphs, binding), body.Graphs) {
		return fmt.Errorf("%w: lowered graph plan mismatch", ErrInvalidProgramArtifact)
	}
	return nil
}

func sourceFromProgram(body programBody) schema.WorkflowSource {
	graphs := make([]schema.Graph, len(body.Graphs))
	for graphIndex, graph := range body.Graphs {
		nodes := make([]schema.Node, len(graph.Nodes))
		for nodeIndex, node := range graph.Nodes {
			nodes[nodeIndex] = schema.Node{ID: node.ID, Kind: node.Kind, Config: node.Config, Disabled: node.Disabled}
		}
		graphs[graphIndex] = schema.Graph{ID: graph.ID, Kind: graph.Kind, Nodes: nodes, Edges: graph.Edges, Inputs: graph.Inputs, Outputs: graph.Outputs}
	}
	requested := make([]schema.Capability, len(body.RequestedCapabilities))
	for index, capability := range body.RequestedCapabilities {
		requested[index] = schema.Capability(capability)
	}
	return schema.WorkflowSource{Workflow: schema.Workflow{ID: body.WorkflowID}, Revision: body.Revision, EntryGraph: body.EntryGraph, Graphs: graphs, Variables: body.Variables, SecretRefs: body.SecretRefs, RequestedCapabilities: requested}
}

func equalLocks(left, right []NodeLock) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func validateEnvelope(envelope programEnvelope) error {
	if envelope.Format != ProgramFormat || envelope.Version != ProgramVersion {
		return fmt.Errorf("%w: unsupported format", ErrInvalidProgramArtifact)
	}
	if !envelope.ProgramHash.Valid() || !envelope.Program.SourceHash.Valid() || !envelope.Program.CatalogHash.Valid() {
		return fmt.Errorf("%w: invalid digest", ErrInvalidProgramArtifact)
	}
	body := envelope.Program
	if !body.CompilerBuild.Valid() || !body.ImplementationSet.Valid() || body.WorkflowID == "" || body.EntryGraph == "" || body.Revision < 0 || body.Revision > schema.MaxRevision {
		return fmt.Errorf("%w: missing program identity", ErrInvalidProgramArtifact)
	}
	if body.NodeLocks == nil || body.RequestedCapabilities == nil || body.RequiredCapabilities == nil || body.Variables == nil || body.SecretRefs == nil || len(body.Graphs) == 0 {
		return fmt.Errorf("%w: missing collections", ErrInvalidProgramArtifact)
	}
	if len(body.Variables) > schema.MaxVariables || len(body.SecretRefs) > schema.MaxSecretRefs || len(body.RequestedCapabilities) > schema.MaxRequestedCapabilities || len(body.RequiredCapabilities) > schema.MaxRequestedCapabilities {
		return fmt.Errorf("%w: scalar collection budget", ErrInvalidProgramArtifact)
	}
	if len(body.Graphs) > MaxGraphs {
		return fmt.Errorf("%w: graph budget", ErrInvalidProgramArtifact)
	}
	variableNames := map[string]bool{}
	for _, variable := range body.Variables {
		if variable.Name == "" || variable.Type == "" || variableNames[variable.Name] {
			return fmt.Errorf("%w: invalid variable", ErrInvalidProgramArtifact)
		}
		variableNames[variable.Name] = true
	}
	secretIDs := map[string]bool{}
	for _, secret := range body.SecretRefs {
		if secret.ID == "" || secret.Purpose == "" || secretIDs[secret.ID] {
			return fmt.Errorf("%w: invalid secret ref", ErrInvalidProgramArtifact)
		}
		secretIDs[secret.ID] = true
	}
	if !strictlySortedLocks(body.NodeLocks) || !strictlySortedStrings(body.RequestedCapabilities) || !strictlySortedStrings(body.RequiredCapabilities) {
		return fmt.Errorf("%w: unordered or duplicate set", ErrInvalidProgramArtifact)
	}
	graphs := make(map[string]schema.GraphKind, len(body.Graphs))
	lockedKinds := make(map[string]bool, len(body.NodeLocks))
	for _, lock := range body.NodeLocks {
		lockedKinds[lock.Kind] = true
	}
	usedKinds := map[string]bool{}
	mainGraphs := 0
	totalNodes, totalEdges, totalPorts := 0, 0, 0
	totalDynamicPorts, totalDynamicPortBytes := 0, 0
	for _, graph := range body.Graphs {
		if len(graph.Nodes) > MaxNodesPerGraph || len(graph.Edges) > MaxEdgesPerGraph || len(graph.Inputs)+len(graph.Outputs) > MaxPortsPerGraph {
			return fmt.Errorf("%w: graph collection budget", ErrInvalidProgramArtifact)
		}
		totalNodes += len(graph.Nodes)
		totalEdges += len(graph.Edges)
		totalPorts += len(graph.Inputs) + len(graph.Outputs)
		if graph.ID == "" || graph.Nodes == nil || graph.Edges == nil || graph.Inputs == nil || graph.Outputs == nil {
			return fmt.Errorf("%w: invalid graph %q collections nodes=%t edges=%t inputs=%t outputs=%t", ErrInvalidProgramArtifact, graph.ID, graph.Nodes != nil, graph.Edges != nil, graph.Inputs != nil, graph.Outputs != nil)
		}
		if _, duplicate := graphs[graph.ID]; duplicate {
			return fmt.Errorf("%w: duplicate graph", ErrInvalidProgramArtifact)
		}
		if graph.Kind != schema.GraphKindMain && graph.Kind != schema.GraphKindSubgraph {
			return fmt.Errorf("%w: invalid graph kind", ErrInvalidProgramArtifact)
		}
		graphs[graph.ID] = graph.Kind
		if graph.Kind == schema.GraphKindMain {
			mainGraphs++
		}
		nodes := map[string]bool{}
		nodeIDs := make(map[string]bool, len(graph.Nodes))
		for _, node := range graph.Nodes {
			if node.ID == "" || node.Kind == "" || node.Config == nil || node.DynamicPorts == nil || nodes[node.ID] {
				return fmt.Errorf("%w: invalid node", ErrInvalidProgramArtifact)
			}
			if len(node.DynamicPorts) > MaxDynamicPortsPerNode || len(node.DynamicPorts) > MaxTotalDynamicPorts-totalDynamicPorts {
				return fmt.Errorf("%w: dynamic port collection budget", ErrInvalidProgramArtifact)
			}
			planBytes := dynamicPortPlanSize(node.DynamicPorts)
			if planBytes > MaxDynamicPortPlanBytes-totalDynamicPortBytes {
				return fmt.Errorf("%w: dynamic port byte budget", ErrInvalidProgramArtifact)
			}
			totalDynamicPorts += len(node.DynamicPorts)
			totalDynamicPortBytes += planBytes
			dynamicNames := map[string]bool{}
			for _, port := range node.DynamicPorts {
				if port.Role != nodespec.DynamicPortOutput || port.Type != nodespec.TypeExec || port.ParentOutput != "" || dynamicPortNameReason(port.Name, nil, dynamicNames) != "" {
					return fmt.Errorf("%w: invalid dynamic port", ErrInvalidProgramArtifact)
				}
				dynamicNames[port.Name] = true
			}
			nodes[node.ID] = true
			nodeIDs[node.ID] = true
			if node.Kind == CallSubgraphKind {
				if len(node.DynamicPorts) != 0 || node.Call == nil || node.Call.GraphID == "" || !validProgramCallPort(node.Call.Entry) || node.Call.Entry.Type != "Exec" || node.Call.Inputs == nil || node.Call.Outputs == nil || node.Call.FailurePin != callFailurePin {
					return fmt.Errorf("%w: invalid subgraph call plan", ErrInvalidProgramArtifact)
				}
				for _, port := range append(append([]programCallPort(nil), node.Call.Inputs...), node.Call.Outputs...) {
					if !validProgramCallPort(port) {
						return fmt.Errorf("%w: invalid subgraph call port", ErrInvalidProgramArtifact)
					}
				}
			} else if node.Call != nil {
				return fmt.Errorf("%w: unexpected call plan", ErrInvalidProgramArtifact)
			} else {
				usedKinds[node.Kind] = true
			}
			if node.Kind != CallSubgraphKind && !lockedKinds[node.Kind] {
				return fmt.Errorf("%w: missing node lock", ErrInvalidProgramArtifact)
			}
		}
		inputBoundaries, outputBoundaries := map[string]bool{}, map[string]bool{}
		allPortIDs, boundaryNodes := map[string]bool{}, map[string]bool{}
		for groupIndex, ports := range [][]schema.GraphPort{graph.Inputs, graph.Outputs} {
			for _, port := range ports {
				if port.ID == "" || port.Name == "" || port.Type == "" || port.NodeID == "" || allPortIDs[port.ID] || boundaryNodes[port.NodeID] || nodes[port.NodeID] {
					return fmt.Errorf("%w: invalid graph port", ErrInvalidProgramArtifact)
				}
				allPortIDs[port.ID], boundaryNodes[port.NodeID] = true, true
				endpoint := port.NodeID + "." + port.ID
				if groupIndex == 0 {
					inputBoundaries[endpoint] = true
				} else {
					outputBoundaries[endpoint] = true
				}
			}
		}
		for _, edge := range graph.Edges {
			if edge.From == "" || edge.To == "" {
				return fmt.Errorf("%w: invalid edge", ErrInvalidProgramArtifact)
			}
			if _, _, ok := splitEndpoint(edge.From, nodeIDs); !ok && !inputBoundaries[edge.From] {
				return fmt.Errorf("%w: dangling edge source", ErrInvalidProgramArtifact)
			}
			if _, _, ok := splitEndpoint(edge.To, nodeIDs); !ok && !outputBoundaries[edge.To] {
				return fmt.Errorf("%w: dangling edge target", ErrInvalidProgramArtifact)
			}
		}
	}
	if totalNodes > MaxTotalNodes || totalEdges > MaxTotalEdges || totalPorts > MaxTotalPorts {
		return fmt.Errorf("%w: total collection budget", ErrInvalidProgramArtifact)
	}
	if len(usedKinds) != len(lockedKinds) {
		return fmt.Errorf("%w: unused node lock", ErrInvalidProgramArtifact)
	}
	if mainGraphs != 1 || graphs[body.EntryGraph] != schema.GraphKindMain {
		return fmt.Errorf("%w: invalid entry graph", ErrInvalidProgramArtifact)
	}
	return nil
}

func validProgramCallPort(port programCallPort) bool {
	return port.ID != "" && port.Type != "" && port.NodeID != ""
}

func strictlySortedLocks(locks []NodeLock) bool {
	for index, lock := range locks {
		if lock.Kind == "" || !lock.ContractHash.Valid() || index > 0 && locks[index-1].Kind >= lock.Kind {
			return false
		}
	}
	return true
}

func strictlySortedStrings(values []string) bool {
	for index, value := range values {
		if value == "" || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}

func sortedSet(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
