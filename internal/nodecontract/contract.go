// Package nodecontract defines the durable Node Contract 3.1 machine contract.
package nodecontract

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/contractschema"
	"github.com/yottaapp/yotta/internal/datatype"
)

const (
	Format  = "yotta.node-contract"
	Version = "3.1"

	semanticDigestDomain = "yotta/node-contract-semantic/v1"
	MaxContractBytes     = 1 << 20
	MaxContractDepth     = 96
	MaxContractNodes     = 262_144
	MaxPorts             = 4_096
	MaxIdentifierBytes   = 256
	MaxAuthoringBytes    = 4 << 10
)

var versionSegmentPattern = regexp.MustCompile(`^v[1-9][0-9]*$`)
var portIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`)

type NodeRef struct {
	NodeTypeID     string          `json:"nodeTypeId" jsonschema:"required,format=uri,pattern=/v[1-9][0-9]*$"`
	SemanticDigest artifact.Digest `json:"semanticDigest" jsonschema:"required,pattern=^sha256:[a-f0-9]{64}$"`
}

type DataInputPort struct {
	ID       string                  `json:"id" jsonschema:"required,pattern=^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$"`
	Type     datatype.TypeExpression `json:"type" jsonschema:"required"`
	Required bool                    `json:"required" jsonschema:"required"`
	Default  *json.RawMessage        `json:"default,omitempty"`
}

type DataOutputPort struct {
	ID   string                  `json:"id" jsonschema:"required,pattern=^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$"`
	Type datatype.TypeExpression `json:"type" jsonschema:"required"`
}

type SignalPort struct {
	ID string `json:"id" jsonschema:"required,pattern=^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$"`
}

type PortSet struct {
	DataInputs    []DataInputPort  `json:"dataInputs" jsonschema:"required,maxItems=4096"`
	DataOutputs   []DataOutputPort `json:"dataOutputs" jsonschema:"required,maxItems=4096"`
	ExecInputs    []SignalPort     `json:"execInputs" jsonschema:"required,maxItems=4096"`
	ExecOutputs   []SignalPort     `json:"execOutputs" jsonschema:"required,maxItems=4096"`
	ErrorOutputs  []SignalPort     `json:"errorOutputs" jsonschema:"required,maxItems=4096"`
	StatusOutputs []SignalPort     `json:"statusOutputs" jsonschema:"required,maxItems=4096"`
}

type ExecutionClass string

const (
	ExecutionPureData ExecutionClass = "pure-data"
	ExecutionEffect   ExecutionClass = "effect"
	ExecutionControl  ExecutionClass = "control"
	ExecutionEvent    ExecutionClass = "event"
	ExecutionRegion   ExecutionClass = "region"
	ExecutionMarker   ExecutionClass = "marker"
	ExecutionVisual   ExecutionClass = "visual"
)

type Determinism string

const (
	Deterministic    Determinism = "deterministic"
	Recorded         Determinism = "recorded"
	Nondeterministic Determinism = "nondeterministic"
)

type EvaluationMode string

const (
	EvaluationPull EvaluationMode = "pull"
	EvaluationPush EvaluationMode = "push"
)

type CachePolicy string

const (
	CacheNone   CachePolicy = "none"
	CachePerRun CachePolicy = "per-run"
)

type RetrySafety string

const (
	RetryNever       RetrySafety = "never"
	RetryIdempotent  RetrySafety = "idempotent"
	RetryOperationID RetrySafety = "operation-id"
)

type CancellationMode string

const (
	CancellationCooperative CancellationMode = "cooperative"
	CancellationImmediate   CancellationMode = "immediate"
	CancellationUnsupported CancellationMode = "unsupported"
)

type TimeoutContract string

const (
	TimeoutNone     TimeoutContract = "none"
	TimeoutRequired TimeoutContract = "required"
	TimeoutOptional TimeoutContract = "optional"
)

type EffectID string
type CapabilityID string

type ExecutionSpec struct {
	Class        ExecutionClass   `json:"class" jsonschema:"required,enum=pure-data,enum=effect,enum=control,enum=event,enum=region,enum=marker,enum=visual"`
	Effects      []EffectID       `json:"effects" jsonschema:"required,maxItems=4096"`
	Determinism  Determinism      `json:"determinism" jsonschema:"required,enum=deterministic,enum=recorded,enum=nondeterministic"`
	Evaluation   EvaluationMode   `json:"evaluation" jsonschema:"required,enum=pull,enum=push"`
	Cache        CachePolicy      `json:"cache" jsonschema:"required,enum=none,enum=per-run"`
	Retry        RetrySafety      `json:"retry" jsonschema:"required,enum=never,enum=idempotent,enum=operation-id"`
	Cancellation CancellationMode `json:"cancellation" jsonschema:"required,enum=cooperative,enum=immediate,enum=unsupported"`
	Timeout      TimeoutContract  `json:"timeout" jsonschema:"required,enum=none,enum=required,enum=optional"`
}

type ErrorSpec struct {
	Code      string `json:"code" jsonschema:"required,minLength=1,maxLength=256"`
	Category  string `json:"category" jsonschema:"required,minLength=1,maxLength=256"`
	RetryHint bool   `json:"retryHint" jsonschema:"required"`
}

type InstanceResolver struct {
	ResolverID     string          `json:"resolverId" jsonschema:"required,format=uri,pattern=/v[1-9][0-9]*$"`
	SemanticDigest artifact.Digest `json:"semanticDigest" jsonschema:"required,pattern=^sha256:[a-f0-9]{64}$"`
	MaxPorts       int             `json:"maxPorts" jsonschema:"required,minimum=1,maximum=4096"`
}

type ABIKind string

const (
	ABIBuiltin ABIKind = "builtin"
	ABIWIT     ABIKind = "wit"
	ABIProcess ABIKind = "process"
)

type ABIRequirement struct {
	Kind    ABIKind `json:"kind" jsonschema:"required,enum=builtin,enum=wit,enum=process"`
	Version string  `json:"version" jsonschema:"required,pattern=^v[1-9][0-9]*$"`
}

type Authoring struct {
	TitleKey       string   `json:"titleKey,omitempty"`
	DescriptionKey string   `json:"descriptionKey,omitempty"`
	Category       string   `json:"category,omitempty"`
	Tags           []string `json:"tags" jsonschema:"required,maxItems=4096"`
	Icon           string   `json:"icon,omitempty"`
	EditorAdapter  string   `json:"editorAdapter,omitempty"`
}

type Draft struct {
	NodeTypeID         string
	ConfigSchemaRoot   string
	ConfigSchemaBundle []datatype.SchemaResource
	Ports              PortSet
	Execution          ExecutionSpec
	Capabilities       []CapabilityID
	Errors             []ErrorSpec
	InstanceResolver   *InstanceResolver
	ImplementationABI  []ABIRequirement
	Authoring          Authoring
}

type semanticDocument struct {
	NodeTypeID         string                    `json:"nodeTypeId" jsonschema:"required,format=uri,pattern=/v[1-9][0-9]*$"`
	ConfigSchemaRoot   string                    `json:"configSchemaRoot" jsonschema:"required,format=uri"`
	ConfigSchemaBundle []datatype.SchemaResource `json:"configSchemaBundle" jsonschema:"required,minItems=1,maxItems=256"`
	Ports              PortSet                   `json:"ports" jsonschema:"required"`
	Execution          ExecutionSpec             `json:"execution" jsonschema:"required"`
	Capabilities       []CapabilityID            `json:"capabilities" jsonschema:"required,maxItems=4096"`
	Errors             []ErrorSpec               `json:"errors" jsonschema:"required,maxItems=4096"`
	InstanceResolver   *InstanceResolver         `json:"instanceResolver,omitempty"`
	ImplementationABI  []ABIRequirement          `json:"implementationABI" jsonschema:"required,minItems=1"`
}

type document struct {
	Format    string           `json:"format" jsonschema:"required,enum=yotta.node-contract"`
	Version   string           `json:"version" jsonschema:"required,enum=3.1"`
	NodeRef   NodeRef          `json:"nodeRef" jsonschema:"required"`
	Semantic  semanticDocument `json:"semantic" jsonschema:"required"`
	Authoring Authoring        `json:"authoring" jsonschema:"required"`
}

type state struct {
	nodeRef NodeRef
	bytes   []byte
}

type Contract struct{ state *state }

func Seal(draft Draft) (Contract, error) {
	semantic, err := normalizeSemantic(draft)
	if err != nil {
		return Contract{}, err
	}
	authoring, err := normalizeAuthoring(draft.Authoring)
	if err != nil {
		return Contract{}, err
	}
	return sealNormalized(semantic, authoring)
}

func Open(raw []byte) (Contract, error) {
	if len(raw) == 0 || len(raw) > MaxContractBytes {
		return Contract{}, errors.New("node contract exceeds byte budget")
	}
	if err := inspectJSON(raw); err != nil {
		return Contract{}, fmt.Errorf("node contract exceeds structural budget: %w", err)
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil {
		return Contract{}, fmt.Errorf("canonicalize node contract: %w", err)
	}
	if !bytes.Equal(raw, canonical) {
		return Contract{}, errors.New("node contract is not canonical")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var decoded document
	if err := decoder.Decode(&decoded); err != nil {
		return Contract{}, fmt.Errorf("decode node contract: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Contract{}, errors.New("node contract contains trailing JSON values")
	}
	if decoded.Format != Format || decoded.Version != Version {
		return Contract{}, errors.New("unsupported node contract format")
	}
	normalized, err := normalizeSemantic(Draft{
		NodeTypeID:         decoded.Semantic.NodeTypeID,
		ConfigSchemaRoot:   decoded.Semantic.ConfigSchemaRoot,
		ConfigSchemaBundle: decoded.Semantic.ConfigSchemaBundle,
		Ports:              decoded.Semantic.Ports,
		Execution:          decoded.Semantic.Execution,
		Capabilities:       decoded.Semantic.Capabilities,
		Errors:             decoded.Semantic.Errors,
		InstanceResolver:   decoded.Semantic.InstanceResolver,
		ImplementationABI:  decoded.Semantic.ImplementationABI,
	})
	if err != nil {
		return Contract{}, err
	}
	authoring, err := normalizeAuthoring(decoded.Authoring)
	if err != nil {
		return Contract{}, err
	}
	sealed, err := sealNormalized(normalized, authoring)
	if err != nil {
		return Contract{}, err
	}
	if decoded.NodeRef.NodeTypeID != normalized.NodeTypeID || decoded.NodeRef != sealed.NodeRef() {
		return Contract{}, errors.New("node contract semantic digest mismatch")
	}
	if !bytes.Equal(sealed.Bytes(), raw) {
		return Contract{}, errors.New("node contract is not normalized")
	}
	return sealed, nil
}

func (c Contract) Valid() bool {
	return c.state != nil && c.state.nodeRef.SemanticDigest.Valid()
}

func (c Contract) NodeRef() NodeRef {
	if !c.Valid() {
		return NodeRef{}
	}
	return c.state.nodeRef
}

func (c Contract) Bytes() []byte {
	if !c.Valid() {
		return nil
	}
	return append([]byte(nil), c.state.bytes...)
}

func sealNormalized(semantic semanticDocument, authoring Authoring) (Contract, error) {
	semanticBytes, err := artifact.Marshal(semantic)
	if err != nil {
		return Contract{}, err
	}
	digest, err := artifact.Sum(semanticDigestDomain, semanticBytes)
	if err != nil {
		return Contract{}, err
	}
	ref := NodeRef{NodeTypeID: semantic.NodeTypeID, SemanticDigest: digest}
	canonical, err := artifact.Marshal(document{Format: Format, Version: Version, NodeRef: ref, Semantic: semantic, Authoring: authoring})
	if err != nil {
		return Contract{}, err
	}
	if len(canonical) > MaxContractBytes {
		return Contract{}, errors.New("node contract exceeds byte budget")
	}
	return Contract{state: &state{nodeRef: ref, bytes: canonical}}, nil
}

func normalizeSemantic(draft Draft) (semanticDocument, error) {
	if err := validateVersionedURI(draft.NodeTypeID); err != nil {
		return semanticDocument{}, fmt.Errorf("invalid node type id: %w", err)
	}
	bundle, err := normalizeSchemaBundle(draft.ConfigSchemaRoot, draft.ConfigSchemaBundle)
	if err != nil {
		return semanticDocument{}, err
	}
	ports, err := normalizePorts(draft.Ports)
	if err != nil {
		return semanticDocument{}, err
	}
	execution, err := normalizeExecution(draft.Execution, ports)
	if err != nil {
		return semanticDocument{}, err
	}
	capabilities, err := normalizeStringSet(draft.Capabilities, "capability")
	if err != nil {
		return semanticDocument{}, err
	}
	for _, capability := range capabilities {
		if err := validateVersionedURI(string(capability)); err != nil {
			return semanticDocument{}, fmt.Errorf("invalid capability requirement %q: %w", capability, err)
		}
	}
	if execution.Class == ExecutionPureData && len(capabilities) != 0 {
		return semanticDocument{}, errors.New("pure-data node must not require capabilities")
	}
	if execution.Class == ExecutionEffect && len(execution.Effects)+len(capabilities) == 0 {
		return semanticDocument{}, errors.New("effect node must declare an effect or capability")
	}
	errorsList, err := normalizeErrors(draft.Errors)
	if err != nil {
		return semanticDocument{}, err
	}
	resolver, err := normalizeResolver(draft.InstanceResolver)
	if err != nil {
		return semanticDocument{}, err
	}
	abis, err := normalizeABIs(draft.ImplementationABI)
	if err != nil {
		return semanticDocument{}, err
	}
	return semanticDocument{
		NodeTypeID: draft.NodeTypeID, ConfigSchemaRoot: draft.ConfigSchemaRoot,
		ConfigSchemaBundle: bundle, Ports: ports, Execution: execution,
		Capabilities: capabilities, Errors: errorsList, InstanceResolver: resolver,
		ImplementationABI: abis,
	}, nil
}

func normalizeSchemaBundle(root string, source []datatype.SchemaResource) ([]datatype.SchemaResource, error) {
	normalized, err := contractschema.Normalize(datatype.JSONSchemaDialect, root, source)
	if err != nil {
		return nil, fmt.Errorf("normalize config schema bundle: %w", err)
	}
	return normalized, nil
}

func normalizePorts(source PortSet) (PortSet, error) {
	total := len(source.DataInputs) + len(source.DataOutputs) + len(source.ExecInputs) + len(source.ExecOutputs) + len(source.ErrorOutputs) + len(source.StatusOutputs)
	if total > MaxPorts {
		return PortSet{}, errors.New("node contract exceeds port budget")
	}
	result := PortSet{
		DataInputs: append([]DataInputPort(nil), source.DataInputs...), DataOutputs: append([]DataOutputPort(nil), source.DataOutputs...),
		ExecInputs: append([]SignalPort(nil), source.ExecInputs...), ExecOutputs: append([]SignalPort(nil), source.ExecOutputs...),
		ErrorOutputs: append([]SignalPort(nil), source.ErrorOutputs...), StatusOutputs: append([]SignalPort(nil), source.StatusOutputs...),
	}
	if result.DataInputs == nil {
		result.DataInputs = []DataInputPort{}
	}
	if result.DataOutputs == nil {
		result.DataOutputs = []DataOutputPort{}
	}
	if result.ExecInputs == nil {
		result.ExecInputs = []SignalPort{}
	}
	if result.ExecOutputs == nil {
		result.ExecOutputs = []SignalPort{}
	}
	if result.ErrorOutputs == nil {
		result.ErrorOutputs = []SignalPort{}
	}
	if result.StatusOutputs == nil {
		result.StatusOutputs = []SignalPort{}
	}
	inputIDs, outputIDs := map[string]bool{}, map[string]bool{}
	for i := range result.DataInputs {
		port := &result.DataInputs[i]
		if err := validatePortID(port.ID, inputIDs); err != nil || port.Type.Validate() != nil {
			return PortSet{}, fmt.Errorf("invalid data input port %q", port.ID)
		}
		if port.Default != nil {
			canonical, err := artifact.Canonicalize(*port.Default)
			if err != nil {
				return PortSet{}, fmt.Errorf("canonicalize default for %q: %w", port.ID, err)
			}
			copy := json.RawMessage(append([]byte(nil), canonical...))
			port.Default = &copy
		}
	}
	for _, port := range result.DataOutputs {
		if err := validatePortID(port.ID, outputIDs); err != nil || port.Type.Validate() != nil {
			return PortSet{}, fmt.Errorf("invalid data output port %q", port.ID)
		}
	}
	for _, port := range result.ExecInputs {
		if err := validatePortID(port.ID, inputIDs); err != nil {
			return PortSet{}, err
		}
	}
	for _, group := range [][]SignalPort{result.ExecOutputs, result.ErrorOutputs, result.StatusOutputs} {
		for _, port := range group {
			if err := validatePortID(port.ID, outputIDs); err != nil {
				return PortSet{}, err
			}
		}
	}
	return result, nil
}

func normalizeExecution(source ExecutionSpec, ports PortSet) (ExecutionSpec, error) {
	if !validExecutionClass(source.Class) || !validDeterminism(source.Determinism) || !validEvaluation(source.Evaluation) ||
		!validCache(source.Cache) || !validRetry(source.Retry) || !validCancellation(source.Cancellation) || !validTimeout(source.Timeout) {
		return ExecutionSpec{}, errors.New("node contract contains invalid execution semantics")
	}
	effects, err := normalizeStringSet(source.Effects, "effect")
	if err != nil {
		return ExecutionSpec{}, err
	}
	for _, effect := range effects {
		if err := validateVersionedURI(string(effect)); err != nil {
			return ExecutionSpec{}, fmt.Errorf("invalid effect %q: %w", effect, err)
		}
	}
	source.Effects = effects
	if source.Class == ExecutionPureData {
		if len(effects) != 0 || source.Evaluation != EvaluationPull || source.Retry != RetryNever ||
			len(ports.ExecInputs)+len(ports.ExecOutputs)+len(ports.ErrorOutputs)+len(ports.StatusOutputs) != 0 {
			return ExecutionSpec{}, errors.New("pure-data node contains effect or control semantics")
		}
	}
	if source.Cache != CacheNone && (source.Determinism != Deterministic || len(effects) != 0) {
		return ExecutionSpec{}, errors.New("only deterministic effect-free nodes may cache")
	}
	return source, nil
}

func normalizeStringSet[T ~string](source []T, label string) ([]T, error) {
	result := append([]T(nil), source...)
	if result == nil {
		result = []T{}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	previous := ""
	for _, item := range result {
		value := string(item)
		if value == "" || len(value) > MaxIdentifierBytes || value <= previous {
			return nil, fmt.Errorf("invalid or duplicate %s %q", label, value)
		}
		previous = value
	}
	return result, nil
}

func normalizeErrors(source []ErrorSpec) ([]ErrorSpec, error) {
	result := append([]ErrorSpec(nil), source...)
	if result == nil {
		result = []ErrorSpec{}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	previous := ""
	for _, item := range result {
		if item.Code == "" || item.Category == "" || len(item.Code) > MaxIdentifierBytes || item.Code <= previous {
			return nil, errors.New("invalid or duplicate node error declaration")
		}
		previous = item.Code
	}
	return result, nil
}

func normalizeResolver(source *InstanceResolver) (*InstanceResolver, error) {
	if source == nil {
		return nil, nil
	}
	if err := validateVersionedURI(source.ResolverID); err != nil || !source.SemanticDigest.Valid() || source.MaxPorts <= 0 || source.MaxPorts > MaxPorts {
		return nil, errors.New("invalid instance resolver declaration")
	}
	copy := *source
	return &copy, nil
}

func normalizeABIs(source []ABIRequirement) ([]ABIRequirement, error) {
	result := append([]ABIRequirement(nil), source...)
	if len(result) == 0 {
		return nil, errors.New("node contract must declare an implementation ABI")
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Kind == result[j].Kind {
			return result[i].Version < result[j].Version
		}
		return result[i].Kind < result[j].Kind
	})
	previous := ""
	for _, item := range result {
		key := string(item.Kind) + "/" + item.Version
		if !validABIKind(item.Kind) || !versionSegmentPattern.MatchString(item.Version) || key <= previous {
			return nil, errors.New("invalid or duplicate implementation ABI")
		}
		previous = key
	}
	return result, nil
}

func normalizeAuthoring(source Authoring) (Authoring, error) {
	for _, value := range []string{source.TitleKey, source.DescriptionKey, source.Category, source.Icon, source.EditorAdapter} {
		if len(value) > MaxAuthoringBytes {
			return Authoring{}, errors.New("node authoring annotation exceeds byte budget")
		}
	}
	tags, err := normalizeStringSet(source.Tags, "authoring tag")
	if err != nil {
		return Authoring{}, err
	}
	source.Tags = tags
	if source.EditorAdapter != "" {
		return Authoring{}, errors.New("node editor adapter is not in the built-in allowlist")
	}
	return source, nil
}

func validatePortID(id string, seen map[string]bool) error {
	if len(id) > MaxIdentifierBytes || !portIDPattern.MatchString(id) || seen[id] {
		return fmt.Errorf("invalid or duplicate port id %q", id)
	}
	seen[id] = true
	return nil
}

func validateVersionedURI(value string) error {
	if err := validateAbsoluteURI(value); err != nil {
		return err
	}
	parsed, _ := url.Parse(value)
	segments := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	if len(segments) == 0 || !versionSegmentPattern.MatchString(segments[len(segments)-1]) {
		return fmt.Errorf("%q must end in /vN", value)
	}
	return nil
}

func validateAbsoluteURI(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" || parsed.Fragment != "" {
		return fmt.Errorf("%q must be an absolute URI without a fragment", value)
	}
	return nil
}

func inspectJSON(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	depth, nodes := 0, 0
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			if depth != 0 {
				return errors.New("unbalanced JSON")
			}
			return nil
		}
		if err != nil {
			return err
		}
		nodes++
		if nodes > MaxContractNodes {
			return errors.New("JSON node budget exceeded")
		}
		if delimiter, ok := token.(json.Delim); ok {
			switch delimiter {
			case '{', '[':
				depth++
				if depth > MaxContractDepth {
					return errors.New("JSON depth budget exceeded")
				}
			case '}', ']':
				depth--
			}
		}
	}
}

func validExecutionClass(value ExecutionClass) bool {
	switch value {
	case ExecutionPureData, ExecutionEffect, ExecutionControl, ExecutionEvent, ExecutionRegion, ExecutionMarker, ExecutionVisual:
		return true
	}
	return false
}
func validDeterminism(value Determinism) bool {
	switch value {
	case Deterministic, Recorded, Nondeterministic:
		return true
	}
	return false
}
func validEvaluation(value EvaluationMode) bool {
	return value == EvaluationPull || value == EvaluationPush
}
func validCache(value CachePolicy) bool { return value == CacheNone || value == CachePerRun }
func validRetry(value RetrySafety) bool {
	return value == RetryNever || value == RetryIdempotent || value == RetryOperationID
}
func validCancellation(value CancellationMode) bool {
	return value == CancellationCooperative || value == CancellationImmediate || value == CancellationUnsupported
}
func validTimeout(value TimeoutContract) bool {
	return value == TimeoutNone || value == TimeoutRequired || value == TimeoutOptional
}
func validABIKind(value ABIKind) bool {
	return value == ABIBuiltin || value == ABIWIT || value == ABIProcess
}
