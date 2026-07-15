package nodes31

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	AddNodeID            = "https://schemas.yotta.dev/nodes/math/add/v1"
	SubtractNodeID       = "https://schemas.yotta.dev/nodes/math/subtract/v1"
	MultiplyNodeID       = "https://schemas.yotta.dev/nodes/math/multiply/v1"
	LessThanNodeID       = "https://schemas.yotta.dev/nodes/comparison/less-than/v1"
	LessOrEqualNodeID    = "https://schemas.yotta.dev/nodes/comparison/less-or-equal/v1"
	GreaterThanNodeID    = "https://schemas.yotta.dev/nodes/comparison/greater-than/v1"
	GreaterOrEqualNodeID = "https://schemas.yotta.dev/nodes/comparison/greater-or-equal/v1"
	AndNodeID            = "https://schemas.yotta.dev/nodes/logic/and/v1"
	OrNodeID             = "https://schemas.yotta.dev/nodes/logic/or/v1"
	NotNodeID            = "https://schemas.yotta.dev/nodes/logic/not/v1"
	ContainsNodeID       = "https://schemas.yotta.dev/nodes/text/contains/v1"
	LengthNodeID         = "https://schemas.yotta.dev/nodes/text/length/v1"
)

const unrepresentableResultCode = "math.result_not_representable"

// InlineFailure is a declared, routable failure from an inline built-in. The
// runtime installer translates it to the Program interpreter's failure type.
type InlineFailure struct {
	Code   string
	Output string
	Cause  error
}

func (f *InlineFailure) Error() string {
	if f == nil {
		return "inline node failure"
	}
	if f.Cause != nil {
		return f.Cause.Error()
	}
	return f.Code
}

func (f *InlineFailure) Unwrap() error {
	if f == nil {
		return nil
	}
	return f.Cause
}

type primitiveTypes struct {
	stringRef  datatype.TypeRef
	numberRef  datatype.TypeRef
	integerRef datatype.TypeRef
	booleanRef datatype.TypeRef
}

type primitivePort struct {
	id           string
	typeRef      datatype.TypeRef
	defaultValue string
}

type primitiveNode struct {
	id          string
	entrypoint  string
	conformance string
	titleKey    string
	description string
	category    string
	tags        []string
	icon        string
	inputs      []primitivePort
	output      primitivePort
	nonFinite   bool
	evaluate    InlineEvaluator
}

func definePrimitiveNodes(types primitiveTypes) ([]BuiltinDefinition, error) {
	number := func(id string) primitivePort {
		return primitivePort{id: id, typeRef: types.numberRef, defaultValue: "0"}
	}
	boolean := func(id string, defaultValue bool) primitivePort {
		return primitivePort{id: id, typeRef: types.booleanRef, defaultValue: fmt.Sprintf("%t", defaultValue)}
	}
	text := func(id string) primitivePort {
		return primitivePort{id: id, typeRef: types.stringRef, defaultValue: `""`}
	}
	result := func(ref datatype.TypeRef) primitivePort { return primitivePort{id: "result", typeRef: ref} }

	specs := []primitiveNode{
		binaryNumberSpec(AddNodeID, "math.add", "node.Add", "plus", number, result(types.numberRef), addNumbers),
		binaryNumberSpec(SubtractNodeID, "math.subtract", "node.Sub", "minus", number, result(types.numberRef), subtractNumbers),
		binaryNumberSpec(MultiplyNodeID, "math.multiply", "node.Mul", "x", number, result(types.numberRef), multiplyNumbers),
		binaryComparisonSpec(LessThanNodeID, "comparison.less-than", "node.Lt", "math-lower", number, result(types.booleanRef), func(a, b float64) bool { return a < b }),
		binaryComparisonSpec(LessOrEqualNodeID, "comparison.less-or-equal", "node.LtEq", "math-equal-lower", number, result(types.booleanRef), func(a, b float64) bool { return a <= b }),
		binaryComparisonSpec(GreaterThanNodeID, "comparison.greater-than", "node.Gt", "math-greater", number, result(types.booleanRef), func(a, b float64) bool { return a > b }),
		binaryComparisonSpec(GreaterOrEqualNodeID, "comparison.greater-or-equal", "node.GtEq", "math-equal-greater", number, result(types.booleanRef), func(a, b float64) bool { return a >= b }),
		{
			id: AndNodeID, entrypoint: "logic.and", conformance: "strict-boolean-and/a+b/result", titleKey: "node.And",
			description: "node.And.description", category: "logic", tags: []string{"boolean", "logic"}, icon: "logic-and",
			inputs: []primitivePort{boolean("a", true), boolean("b", true)}, output: result(types.booleanRef), evaluate: booleanBinary(func(a, b bool) bool { return a && b }),
		},
		{
			id: OrNodeID, entrypoint: "logic.or", conformance: "strict-boolean-or/a+b/result", titleKey: "node.Or",
			description: "node.Or.description", category: "logic", tags: []string{"boolean", "logic"}, icon: "logic-or",
			inputs: []primitivePort{boolean("a", false), boolean("b", false)}, output: result(types.booleanRef), evaluate: booleanBinary(func(a, b bool) bool { return a || b }),
		},
		{
			id: NotNodeID, entrypoint: "logic.not", conformance: "strict-boolean-not/value/result", titleKey: "node.Not",
			description: "node.Not.description", category: "logic", tags: []string{"boolean", "logic"}, icon: "logic-not",
			inputs: []primitivePort{boolean("value", false)}, output: result(types.booleanRef), evaluate: booleanNot,
		},
		{
			id: ContainsNodeID, entrypoint: "text.contains", conformance: "unicode-contains/text+search/result", titleKey: "node.Contains",
			description: "node.Contains.description", category: "text", tags: []string{"text", "search"}, icon: "text-recognition",
			inputs: []primitivePort{text("text"), text("search")}, output: result(types.booleanRef), evaluate: containsText,
		},
		{
			id: LengthNodeID, entrypoint: "text.length", conformance: "unicode-rune-count/text/result", titleKey: "node.Length",
			description: "node.Length.description", category: "text", tags: []string{"text", "length"}, icon: "ruler-measure",
			inputs: []primitivePort{text("text")}, output: result(types.integerRef), evaluate: textLength,
		},
	}

	definitions := make([]BuiltinDefinition, 0, len(specs))
	for _, spec := range specs {
		contract, err := sealPrimitiveNode(spec)
		if err != nil {
			return nil, fmt.Errorf("seal built-in %s: %w", spec.id, err)
		}
		definition, err := defineBuiltin(contract, spec.entrypoint, "v1", spec.conformance, spec.evaluate)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, definition)
	}
	return definitions, nil
}

func binaryNumberSpec(id, entrypoint, key, icon string, input func(string) primitivePort, output primitivePort, evaluate InlineEvaluator) primitiveNode {
	return primitiveNode{
		id: id, entrypoint: entrypoint, conformance: "strict-finite-number/a+b/result", titleKey: key,
		description: key + ".description", category: "math", tags: []string{"math", "number"}, icon: icon,
		inputs: []primitivePort{input("a"), input("b")}, output: output, nonFinite: true, evaluate: evaluate,
	}
}

func binaryComparisonSpec(id, entrypoint, key, icon string, input func(string) primitivePort, output primitivePort, compare func(float64, float64) bool) primitiveNode {
	return primitiveNode{
		id: id, entrypoint: entrypoint, conformance: "strict-number-comparison/a+b/result", titleKey: key,
		description: key + ".description", category: "comparison", tags: []string{"comparison", "number"}, icon: icon,
		inputs: []primitivePort{input("a"), input("b")}, output: output, evaluate: compareNumbers(compare),
	}
}

func sealPrimitiveNode(spec primitiveNode) (nodecontract.Contract, error) {
	inputs := make([]nodecontract.DataInputPort, 0, len(spec.inputs))
	for _, port := range spec.inputs {
		value := json.RawMessage(port.defaultValue)
		inputs = append(inputs, nodecontract.DataInputPort{
			ID: port.id, Type: datatype.RefExpression(port.typeRef), Required: true, Default: &value,
		})
	}
	errorsList := []nodecontract.ErrorSpec{}
	errorOutputs := []nodecontract.SignalPort{}
	if spec.nonFinite {
		errorsList = append(errorsList, nodecontract.ErrorSpec{Code: unrepresentableResultCode, Category: "evaluation", RetryHint: false})
	}
	configID := spec.id + "/config"
	return nodecontract.Seal(nodecontract.Draft{
		NodeTypeID: spec.id, ConfigSchemaRoot: configID, ConfigSchemaBundle: emptyConfigSchema(configID),
		Ports: nodecontract.PortSet{
			DataInputs:  inputs,
			DataOutputs: []nodecontract.DataOutputPort{{ID: spec.output.id, Type: datatype.RefExpression(spec.output.typeRef)}},
			ExecInputs:  []nodecontract.SignalPort{}, ExecOutputs: []nodecontract.SignalPort{}, ErrorOutputs: errorOutputs,
		},
		Execution: pureDataExecution(), Instruction: nodecontract.Invoke(), CapabilityRequirements: []capability.Requirement{}, Errors: errorsList,
		StatusEvents:      []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: spec.titleKey + ".label", DescriptionKey: spec.description, Category: spec.category,
			Tags: spec.tags, Icon: spec.icon,
		},
	})
}

func sealPrimitiveType(typeID, jsonType, key, color, icon string) (datatype.Definition, error) {
	schemaID := typeID + "/schema"
	return datatype.SealDefinition(datatype.DefinitionDraft{
		TypeID: typeID, SchemaDialect: datatype.JSONSchemaDialect, SchemaRoot: schemaID,
		SchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":%q
		}`, schemaID, jsonType))}},
		Representations: []datatype.RepresentationSpec{{Kind: datatype.RepresentationInlineJSON, Codec: datatype.CodecJCSV1}},
		Authoring:       datatype.Authoring{TitleKey: key + ".title", DescriptionKey: key + ".description", Color: color, Icon: icon},
	})
}

func sealSafeIntegerType() (datatype.Definition, error) {
	const schemaID = IntegerTypeID + "/schema"
	return datatype.SealDefinition(datatype.DefinitionDraft{
		TypeID: IntegerTypeID, SchemaDialect: datatype.JSONSchemaDialect, SchemaRoot: schemaID,
		SchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema",
			"type":"integer","minimum":-9007199254740991,"maximum":9007199254740991
		}`, schemaID))}},
		Representations: []datatype.RepresentationSpec{{Kind: datatype.RepresentationInlineJSON, Codec: datatype.CodecJCSV1}},
		Authoring: datatype.Authoring{
			TitleKey: "type.core.integer.title", DescriptionKey: "type.core.integer.description", Color: "#06b6d4", Icon: "number-123",
			Examples: []json.RawMessage{json.RawMessage(`0`), json.RawMessage(`42`)},
		},
	})
}

func pureDataExecution() nodecontract.ExecutionSpec {
	return nodecontract.ExecutionSpec{
		Class: nodecontract.ExecutionPureData, Effects: []nodecontract.EffectID{}, Determinism: nodecontract.Deterministic,
		Evaluation: nodecontract.EvaluationPull, Cache: nodecontract.CachePerRun, Retry: nodecontract.RetryNever,
		Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutNone,
	}
}

func addNumbers(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	a, b, err := numberPair(inputs)
	if err != nil {
		return nil, err
	}
	return finiteNumberResult(a + b)
}

func subtractNumbers(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	a, b, err := numberPair(inputs)
	if err != nil {
		return nil, err
	}
	return finiteNumberResult(a - b)
}

func multiplyNumbers(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	a, b, err := numberPair(inputs)
	if err != nil {
		return nil, err
	}
	return finiteNumberResult(a * b)
}

func compareNumbers(compare func(float64, float64) bool) InlineEvaluator {
	return func(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
		a, b, err := numberPair(inputs)
		if err != nil {
			return nil, err
		}
		return jsonResult(compare(a, b))
	}
}

func booleanBinary(evaluate func(bool, bool) bool) InlineEvaluator {
	return func(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
		var a, b bool
		if err := json.Unmarshal(inputs["a"], &a); err != nil {
			return nil, fmt.Errorf("decode boolean input a: %w", err)
		}
		if err := json.Unmarshal(inputs["b"], &b); err != nil {
			return nil, fmt.Errorf("decode boolean input b: %w", err)
		}
		return jsonResult(evaluate(a, b))
	}
}

func booleanNot(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	var value bool
	if err := json.Unmarshal(inputs["value"], &value); err != nil {
		return nil, fmt.Errorf("decode boolean input value: %w", err)
	}
	return jsonResult(!value)
}

func containsText(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	text, search, err := stringPair(inputs, "text", "search")
	if err != nil {
		return nil, err
	}
	return jsonResult(strings.Contains(text, search))
}

func textLength(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	var text string
	if err := json.Unmarshal(inputs["text"], &text); err != nil {
		return nil, fmt.Errorf("decode string input text: %w", err)
	}
	return jsonResult(utf8.RuneCountInString(text))
}

func numberPair(inputs map[string]json.RawMessage) (float64, float64, error) {
	a, err := decodeFiniteNumber(inputs["a"])
	if err != nil {
		return 0, 0, fmt.Errorf("decode number input a: %w", err)
	}
	b, err := decodeFiniteNumber(inputs["b"])
	if err != nil {
		return 0, 0, fmt.Errorf("decode number input b: %w", err)
	}
	return a, b, nil
}

func decodeFiniteNumber(raw json.RawMessage) (float64, error) {
	var value float64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, err
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, errors.New("number is not finite")
	}
	return value, nil
}

func stringPair(inputs map[string]json.RawMessage, first, second string) (string, string, error) {
	var a, b string
	if err := json.Unmarshal(inputs[first], &a); err != nil {
		return "", "", fmt.Errorf("decode string input %s: %w", first, err)
	}
	if err := json.Unmarshal(inputs[second], &b); err != nil {
		return "", "", fmt.Errorf("decode string input %s: %w", second, err)
	}
	return a, b, nil
}

func finiteNumberResult(value float64) (map[string]json.RawMessage, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, &InlineFailure{Code: unrepresentableResultCode, Cause: errors.New("numeric result is not representable")}
	}
	result, err := jsonResult(value)
	if err != nil {
		return nil, err
	}
	if _, err := artifact.Canonicalize(result["result"]); err != nil {
		return nil, &InlineFailure{Code: unrepresentableResultCode, Cause: errors.New("numeric result is not representable")}
	}
	return result, nil
}

func jsonResult(value any) (map[string]json.RawMessage, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return map[string]json.RawMessage{"result": raw}, nil
}
