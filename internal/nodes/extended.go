package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	DivideNodeID            = "https://schemas.yotta.dev/nodes/math/divide"
	ModuloNodeID            = "https://schemas.yotta.dev/nodes/math/modulo"
	NegateNodeID            = "https://schemas.yotta.dev/nodes/math/negate"
	AbsoluteNodeID          = "https://schemas.yotta.dev/nodes/math/absolute"
	MinimumNodeID           = "https://schemas.yotta.dev/nodes/math/minimum"
	MaximumNodeID           = "https://schemas.yotta.dev/nodes/math/maximum"
	FloorNodeID             = "https://schemas.yotta.dev/nodes/math/floor"
	CeilingNodeID           = "https://schemas.yotta.dev/nodes/math/ceiling"
	RoundNodeID             = "https://schemas.yotta.dev/nodes/math/round"
	ClampNodeID             = "https://schemas.yotta.dev/nodes/math/clamp"
	PowerNodeID             = "https://schemas.yotta.dev/nodes/math/power"
	SquareRootNodeID        = "https://schemas.yotta.dev/nodes/math/square-root"
	EqualNodeID             = "https://schemas.yotta.dev/nodes/comparison/equal"
	NotEqualNodeID          = "https://schemas.yotta.dev/nodes/comparison/not-equal"
	ReplaceNodeID           = "https://schemas.yotta.dev/nodes/text/replace"
	SubstringNodeID         = "https://schemas.yotta.dev/nodes/text/substring"
	TrimNodeID              = "https://schemas.yotta.dev/nodes/text/trim"
	UppercaseNodeID         = "https://schemas.yotta.dev/nodes/text/uppercase"
	LowercaseNodeID         = "https://schemas.yotta.dev/nodes/text/lowercase"
	IndexOfNodeID           = "https://schemas.yotta.dev/nodes/text/index-of"
	StartsWithNodeID        = "https://schemas.yotta.dev/nodes/text/starts-with"
	EndsWithNodeID          = "https://schemas.yotta.dev/nodes/text/ends-with"
	RegexMatchNodeID        = "https://schemas.yotta.dev/nodes/text/regex-match"
	RegexExtractNodeID      = "https://schemas.yotta.dev/nodes/text/regex-extract"
	ToStringNodeID          = "https://schemas.yotta.dev/nodes/conversion/to-string"
	StringToNumberNodeID    = "https://schemas.yotta.dev/nodes/conversion/string-to-number"
	StringToBooleanNodeID   = "https://schemas.yotta.dev/nodes/conversion/string-to-boolean"
	ParseJSONNodeID         = "https://schemas.yotta.dev/nodes/json/parse"
	ToJSONNodeID            = "https://schemas.yotta.dev/nodes/json/stringify"
	JSONPathNodeID          = "https://schemas.yotta.dev/nodes/json/path"
	SelectNodeID            = "https://schemas.yotta.dev/nodes/logic/select"
	MakePointNodeID         = "https://schemas.yotta.dev/nodes/geometry/make-point"
	OffsetPointNodeID       = "https://schemas.yotta.dev/nodes/geometry/offset-point"
	PointDistanceNodeID     = "https://schemas.yotta.dev/nodes/geometry/point-distance"
	RegionAroundPointNodeID = "https://schemas.yotta.dev/nodes/geometry/region-around-point"

	divisionByZeroCode       = "math.division_by_zero"
	mathDomainErrorCode      = "math.domain_error"
	invalidNumberCode        = "conversion.invalid_number"
	invalidBooleanCode       = "conversion.invalid_boolean"
	invalidJSONCode          = "json.invalid_document"
	invalidJSONPathCode      = "json.invalid_path"
	invalidRegexCode         = "text.invalid_regex"
	geometryUnitMismatchCode = "geometry.unit_mismatch"
)

type extendedTypes struct {
	stringRef    datatype.TypeRef
	numberRef    datatype.TypeRef
	integerRef   datatype.TypeRef
	booleanRef   datatype.TypeRef
	jsonRef      datatype.TypeRef
	pointUnitRef datatype.TypeRef
	pointRef     datatype.TypeRef
	regionRef    datatype.TypeRef
}

type extendedPort struct {
	id           string
	typeExpr     datatype.TypeExpression
	defaultValue string
}

type extendedNode struct {
	id          string
	entrypoint  string
	category    string
	tags        []string
	icon        string
	inputs      []extendedPort
	output      extendedPort
	errors      []nodecontract.ErrorSpec
	conformance string
	evaluate    InlineEvaluator
}

func defineExtendedPureNodes(types extendedTypes) ([]BuiltinDefinition, error) {
	stringType := datatype.RefExpression(types.stringRef)
	numberType := datatype.RefExpression(types.numberRef)
	integerType := datatype.RefExpression(types.integerRef)
	booleanType := datatype.RefExpression(types.booleanRef)
	jsonType := datatype.RefExpression(types.jsonRef)
	pointUnitType := datatype.RefExpression(types.pointUnitRef)
	pointType := datatype.RefExpression(types.pointRef)
	regionType := datatype.RefExpression(types.regionRef)
	valueType := datatype.VariableExpression("T")
	port := func(id string, typeExpr datatype.TypeExpression, defaultValue string) extendedPort {
		return extendedPort{id: id, typeExpr: typeExpr, defaultValue: defaultValue}
	}
	result := func(typeExpr datatype.TypeExpression) extendedPort {
		return extendedPort{id: "result", typeExpr: typeExpr}
	}
	number := func(id string, value string) extendedPort { return port(id, numberType, value) }
	integer := func(id string, value string) extendedPort { return port(id, integerType, value) }
	text := func(id string, value string) extendedPort { return port(id, stringType, value) }
	boolean := func(id string, value string) extendedPort { return port(id, booleanType, value) }
	errorSpec := func(code, category string) []nodecontract.ErrorSpec {
		return []nodecontract.ErrorSpec{{Code: code, Category: category, RetryHint: false}}
	}
	resultErrors := errorSpec(unrepresentableResultCode, "evaluation")
	specs := []extendedNode{
		mathBinary(DivideNodeID, "math.divide", "divide", number, result(numberType), append(errorSpec(divisionByZeroCode, "evaluation"), resultErrors...), divideNumbers),
		mathBinary(ModuloNodeID, "math.modulo", "remainder", number, result(numberType), append(errorSpec(divisionByZeroCode, "evaluation"), resultErrors...), moduloNumbers),
		mathUnary(NegateNodeID, "math.negate", "plus-minus", number, result(numberType), resultErrors, negateNumber),
		mathUnary(AbsoluteNodeID, "math.absolute", "math-absolute", number, result(numberType), resultErrors, absoluteNumber),
		mathBinary(MinimumNodeID, "math.minimum", "math-min", number, result(numberType), nil, minimumNumber),
		mathBinary(MaximumNodeID, "math.maximum", "math-max", number, result(numberType), nil, maximumNumber),
		mathUnary(FloorNodeID, "math.floor", "math-function", number, result(numberType), nil, floorNumber),
		mathUnary(CeilingNodeID, "math.ceiling", "math-function", number, result(numberType), nil, ceilingNumber),
		{
			id: RoundNodeID, entrypoint: "math.round", category: "math", tags: []string{"math", "number"}, icon: "decimal",
			inputs: []extendedPort{number("value", "0"), integer("digits", "0")}, output: result(numberType), errors: resultErrors,
			conformance: "bounded-decimal-round/value+digits/result", evaluate: roundNumber,
		},
		{
			id: ClampNodeID, entrypoint: "math.clamp", category: "math", tags: []string{"math", "number"}, icon: "arrows-minimize",
			inputs: []extendedPort{number("value", "0"), number("minimum", "0"), number("maximum", "100")}, output: result(numberType),
			conformance: "ordered-number-clamp/value+minimum+maximum/result", evaluate: clampNumber,
		},
		{
			id: PowerNodeID, entrypoint: "math.power", category: "math", tags: []string{"math", "number"}, icon: "superscript",
			inputs: []extendedPort{number("base", "0"), number("exponent", "1")}, output: result(numberType), errors: append(errorSpec(mathDomainErrorCode, "evaluation"), resultErrors...),
			conformance: "finite-power/base+exponent/result", evaluate: powerNumber,
		},
		mathUnary(SquareRootNodeID, "math.square-root", "square-root", number, result(numberType), append(errorSpec(mathDomainErrorCode, "evaluation"), resultErrors...), squareRoot),
		{
			id: EqualNodeID, entrypoint: "comparison.equal", category: "comparison", tags: []string{"comparison"}, icon: "equal",
			inputs: []extendedPort{port("a", valueType, ""), port("b", valueType, "")}, output: result(booleanType),
			conformance: "canonical-equality/T+T/boolean", evaluate: equalValues,
		},
		{
			id: NotEqualNodeID, entrypoint: "comparison.not-equal", category: "comparison", tags: []string{"comparison"}, icon: "not-equal",
			inputs: []extendedPort{port("a", valueType, ""), port("b", valueType, "")}, output: result(booleanType),
			conformance: "canonical-inequality/T+T/boolean", evaluate: notEqualValues,
		},
		textNode(ReplaceNodeID, "text.replace", "replace", []extendedPort{text("text", `""`), text("old", `""`), text("new", `""`), boolean("all", "true")}, result(stringType), nil, "unicode-replace", replaceText),
		textNode(SubstringNodeID, "text.substring", "section", []extendedPort{text("text", `""`), integer("start", "0"), integer("length", "-1")}, result(stringType), nil, "unicode-rune-substring", substringText),
		textNode(TrimNodeID, "text.trim", "spacing-vertical", []extendedPort{text("text", `""`)}, result(stringType), nil, "unicode-trim-space", trimText),
		textNode(UppercaseNodeID, "text.uppercase", "letter-case-upper", []extendedPort{text("text", `""`)}, result(stringType), nil, "unicode-uppercase", uppercaseText),
		textNode(LowercaseNodeID, "text.lowercase", "letter-case-lower", []extendedPort{text("text", `""`)}, result(stringType), nil, "unicode-lowercase", lowercaseText),
		textNode(IndexOfNodeID, "text.index-of", "text-search", []extendedPort{text("text", `""`), text("search", `""`)}, result(integerType), nil, "unicode-rune-index", indexOfText),
		textNode(StartsWithNodeID, "text.starts-with", "arrow-bar-right", []extendedPort{text("text", `""`), text("prefix", `""`)}, result(booleanType), nil, "unicode-prefix", startsWithText),
		textNode(EndsWithNodeID, "text.ends-with", "arrow-bar-left", []extendedPort{text("text", `""`), text("suffix", `""`)}, result(booleanType), nil, "unicode-suffix", endsWithText),
		textNode(RegexMatchNodeID, "text.regex-match", "regex", []extendedPort{text("text", `""`), text("pattern", `""`)}, result(booleanType), errorSpec(invalidRegexCode, "evaluation"), "re2-search", regexMatch),
		textNode(RegexExtractNodeID, "text.regex-extract", "regex", []extendedPort{text("text", `""`), text("pattern", `""`)}, result(stringType), errorSpec(invalidRegexCode, "evaluation"), "re2-first-capture", regexExtract),
		{
			id: ToStringNodeID, entrypoint: "conversion.to-string", category: "conversion", tags: []string{"conversion", "text"}, icon: "text-caption",
			inputs: []extendedPort{port("value", valueType, "")}, output: result(stringType), conformance: "canonical-value-to-string", evaluate: valueToString,
		},
		{
			id: StringToNumberNodeID, entrypoint: "conversion.string-to-number", category: "conversion", tags: []string{"conversion", "number"}, icon: "numbers",
			inputs: []extendedPort{text("text", `""`)}, output: result(numberType), errors: errorSpec(invalidNumberCode, "evaluation"), conformance: "strict-decimal-string-to-number", evaluate: stringToNumber,
		},
		{
			id: StringToBooleanNodeID, entrypoint: "conversion.string-to-boolean", category: "conversion", tags: []string{"conversion", "boolean"}, icon: "toggle-right",
			inputs: []extendedPort{text("text", `"false"`)}, output: result(booleanType), errors: errorSpec(invalidBooleanCode, "evaluation"), conformance: "strict-lowercase-string-to-boolean", evaluate: stringToBoolean,
		},
		{
			id: ParseJSONNodeID, entrypoint: "json.parse", category: "json", tags: []string{"json", "conversion"}, icon: "braces",
			inputs: []extendedPort{text("text", `"null"`)}, output: result(jsonType), errors: errorSpec(invalidJSONCode, "evaluation"), conformance: "strict-single-json-document", evaluate: parseJSONValue,
		},
		{
			id: ToJSONNodeID, entrypoint: "json.stringify", category: "json", tags: []string{"json", "conversion"}, icon: "braces",
			inputs: []extendedPort{port("value", valueType, "")}, output: result(stringType), conformance: "canonical-json-stringify", evaluate: stringifyJSONValue,
		},
		{
			id: JSONPathNodeID, entrypoint: "json.path", category: "json", tags: []string{"json", "query"}, icon: "route",
			inputs: []extendedPort{port("json", jsonType, ""), text("path", `"$"`)}, output: result(jsonType), errors: errorSpec(invalidJSONPathCode, "evaluation"), conformance: "bounded-json-path", evaluate: queryJSONPath,
		},
		{
			id: SelectNodeID, entrypoint: "logic.select", category: "logic", tags: []string{"logic", "generic"}, icon: "selector",
			inputs: []extendedPort{boolean("condition", "true"), port("when_true", valueType, ""), port("when_false", valueType, "")}, output: result(valueType), conformance: "strict-typed-select", evaluate: selectValue,
		},
		{
			id: MakePointNodeID, entrypoint: "geometry.make-point", category: "geometry", tags: []string{"geometry", "point"}, icon: "map-pin",
			inputs: []extendedPort{number("x", "0"), number("y", "0"), port("unit", pointUnitType, `"ratio"`)}, output: result(pointType), conformance: "typed-point-construction", evaluate: makePoint,
		},
		{
			id: OffsetPointNodeID, entrypoint: "geometry.offset-point", category: "geometry", tags: []string{"geometry", "point"}, icon: "arrows-move",
			inputs: []extendedPort{port("point", pointType, ""), number("offset_x", "0"), number("offset_y", "0")}, output: result(pointType), errors: resultErrors, conformance: "unit-preserving-point-offset", evaluate: offsetPoint,
		},
		{
			id: PointDistanceNodeID, entrypoint: "geometry.point-distance", category: "geometry", tags: []string{"geometry", "point"}, icon: "ruler-measure",
			inputs: []extendedPort{port("begin", pointType, ""), port("end", pointType, "")}, output: result(numberType), errors: append(errorSpec(geometryUnitMismatchCode, "evaluation"), resultErrors...), conformance: "same-unit-point-distance", evaluate: pointDistance,
		},
		{
			id: RegionAroundPointNodeID, entrypoint: "geometry.region-around-point", category: "geometry", tags: []string{"geometry", "region"}, icon: "crop",
			inputs: []extendedPort{port("center", pointType, ""), number("width", "0.2"), number("height", "0.2")}, output: result(regionType), errors: resultErrors, conformance: "same-unit-centered-region", evaluate: regionAroundPoint,
		},
	}

	definitions := make([]BuiltinDefinition, 0, len(specs))
	for _, spec := range specs {
		contract, err := sealExtendedNode(spec)
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

func mathBinary(id, entrypoint, icon string, number func(string, string) extendedPort, output extendedPort, errors []nodecontract.ErrorSpec, evaluate InlineEvaluator) extendedNode {
	return extendedNode{id: id, entrypoint: entrypoint, category: "math", tags: []string{"math", "number"}, icon: icon,
		inputs: []extendedPort{number("a", "0"), number("b", "0")}, output: output, errors: errors,
		conformance: "strict-finite-number-binary", evaluate: evaluate}
}

func mathUnary(id, entrypoint, icon string, number func(string, string) extendedPort, output extendedPort, errors []nodecontract.ErrorSpec, evaluate InlineEvaluator) extendedNode {
	return extendedNode{id: id, entrypoint: entrypoint, category: "math", tags: []string{"math", "number"}, icon: icon,
		inputs: []extendedPort{number("value", "0")}, output: output, errors: errors,
		conformance: "strict-finite-number-unary", evaluate: evaluate}
}

func textNode(id, entrypoint, icon string, inputs []extendedPort, output extendedPort, errors []nodecontract.ErrorSpec, conformance string, evaluate InlineEvaluator) extendedNode {
	return extendedNode{id: id, entrypoint: entrypoint, category: "text", tags: []string{"text"}, icon: icon,
		inputs: inputs, output: output, errors: errors, conformance: conformance, evaluate: evaluate}
}

func sealExtendedNode(spec extendedNode) (nodecontract.Contract, error) {
	inputs := make([]nodecontract.DataInputPort, 0, len(spec.inputs))
	for _, port := range spec.inputs {
		input := nodecontract.DataInputPort{ID: port.id, Type: port.typeExpr, Required: true}
		if port.defaultValue != "" {
			value := json.RawMessage(port.defaultValue)
			input.Default = &value
		}
		inputs = append(inputs, input)
	}
	configID := spec.id + "/config"
	return nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
		NodeTypeID: spec.id, ConfigSchemaRoot: configID, ConfigSchemaBundle: emptyConfigSchema(configID),
		Ports: nodecontract.PortSet{
			DataInputs: inputs, DataOutputs: []nodecontract.DataOutputPort{{ID: spec.output.id, Type: spec.output.typeExpr}},
			ExecInputs: []nodecontract.SignalPort{}, ExecOutputs: []nodecontract.SignalPort{}, ErrorOutputs: []nodecontract.SignalPort{},
		},
		Execution: pureDataExecution(), Instruction: nodecontract.Invoke(), CapabilityRequirements: []capability.Requirement{}, Errors: spec.errors,
		StatusEvents:      []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: builtinMessageKey(spec.entrypoint) + ".title", DescriptionKey: builtinMessageKey(spec.entrypoint) + ".description", Category: spec.category,
			Tags: spec.tags, Icon: spec.icon,
		},
	})
}
