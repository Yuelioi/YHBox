package nodes31

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	DivideNodeID            = "https://schemas.yotta.dev/nodes/math/divide/v1"
	ModuloNodeID            = "https://schemas.yotta.dev/nodes/math/modulo/v1"
	NegateNodeID            = "https://schemas.yotta.dev/nodes/math/negate/v1"
	AbsoluteNodeID          = "https://schemas.yotta.dev/nodes/math/absolute/v1"
	MinimumNodeID           = "https://schemas.yotta.dev/nodes/math/minimum/v1"
	MaximumNodeID           = "https://schemas.yotta.dev/nodes/math/maximum/v1"
	FloorNodeID             = "https://schemas.yotta.dev/nodes/math/floor/v1"
	CeilingNodeID           = "https://schemas.yotta.dev/nodes/math/ceiling/v1"
	RoundNodeID             = "https://schemas.yotta.dev/nodes/math/round/v1"
	ClampNodeID             = "https://schemas.yotta.dev/nodes/math/clamp/v1"
	PowerNodeID             = "https://schemas.yotta.dev/nodes/math/power/v1"
	SquareRootNodeID        = "https://schemas.yotta.dev/nodes/math/square-root/v1"
	EqualNodeID             = "https://schemas.yotta.dev/nodes/comparison/equal/v1"
	NotEqualNodeID          = "https://schemas.yotta.dev/nodes/comparison/not-equal/v1"
	ReplaceNodeID           = "https://schemas.yotta.dev/nodes/text/replace/v1"
	SubstringNodeID         = "https://schemas.yotta.dev/nodes/text/substring/v1"
	TrimNodeID              = "https://schemas.yotta.dev/nodes/text/trim/v1"
	UppercaseNodeID         = "https://schemas.yotta.dev/nodes/text/uppercase/v1"
	LowercaseNodeID         = "https://schemas.yotta.dev/nodes/text/lowercase/v1"
	IndexOfNodeID           = "https://schemas.yotta.dev/nodes/text/index-of/v1"
	StartsWithNodeID        = "https://schemas.yotta.dev/nodes/text/starts-with/v1"
	EndsWithNodeID          = "https://schemas.yotta.dev/nodes/text/ends-with/v1"
	RegexMatchNodeID        = "https://schemas.yotta.dev/nodes/text/regex-match/v1"
	RegexExtractNodeID      = "https://schemas.yotta.dev/nodes/text/regex-extract/v1"
	ToStringNodeID          = "https://schemas.yotta.dev/nodes/conversion/to-string/v1"
	StringToNumberNodeID    = "https://schemas.yotta.dev/nodes/conversion/string-to-number/v1"
	StringToBooleanNodeID   = "https://schemas.yotta.dev/nodes/conversion/string-to-boolean/v1"
	ParseJSONNodeID         = "https://schemas.yotta.dev/nodes/json/parse/v1"
	ToJSONNodeID            = "https://schemas.yotta.dev/nodes/json/stringify/v1"
	JSONPathNodeID          = "https://schemas.yotta.dev/nodes/json/path/v1"
	SelectNodeID            = "https://schemas.yotta.dev/nodes/logic/select/v1"
	MakePointNodeID         = "https://schemas.yotta.dev/nodes/geometry/make-point/v1"
	OffsetPointNodeID       = "https://schemas.yotta.dev/nodes/geometry/offset-point/v1"
	PointDistanceNodeID     = "https://schemas.yotta.dev/nodes/geometry/point-distance/v1"
	RegionAroundPointNodeID = "https://schemas.yotta.dev/nodes/geometry/region-around-point/v1"

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
	key         string
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
		mathBinary(DivideNodeID, "math.divide", "node.Div", "divide", number, result(numberType), append(errorSpec(divisionByZeroCode, "evaluation"), resultErrors...), divideNumbers),
		mathBinary(ModuloNodeID, "math.modulo", "node.Mod", "remainder", number, result(numberType), append(errorSpec(divisionByZeroCode, "evaluation"), resultErrors...), moduloNumbers),
		mathUnary(NegateNodeID, "math.negate", "node.Neg", "plus-minus", number, result(numberType), resultErrors, negateNumber),
		mathUnary(AbsoluteNodeID, "math.absolute", "node.Abs", "math-absolute", number, result(numberType), resultErrors, absoluteNumber),
		mathBinary(MinimumNodeID, "math.minimum", "node.Min", "math-min", number, result(numberType), nil, minimumNumber),
		mathBinary(MaximumNodeID, "math.maximum", "node.Max", "math-max", number, result(numberType), nil, maximumNumber),
		mathUnary(FloorNodeID, "math.floor", "node.Floor", "math-function", number, result(numberType), nil, floorNumber),
		mathUnary(CeilingNodeID, "math.ceiling", "node.Ceil", "math-function", number, result(numberType), nil, ceilingNumber),
		{
			id: RoundNodeID, entrypoint: "math.round", key: "node.Round", category: "math", tags: []string{"math", "number"}, icon: "decimal",
			inputs: []extendedPort{number("value", "0"), integer("digits", "0")}, output: result(numberType), errors: resultErrors,
			conformance: "bounded-decimal-round/value+digits/result", evaluate: roundNumber,
		},
		{
			id: ClampNodeID, entrypoint: "math.clamp", key: "node.Clamp", category: "math", tags: []string{"math", "number"}, icon: "arrows-minimize",
			inputs: []extendedPort{number("value", "0"), number("minimum", "0"), number("maximum", "100")}, output: result(numberType),
			conformance: "ordered-number-clamp/value+minimum+maximum/result", evaluate: clampNumber,
		},
		{
			id: PowerNodeID, entrypoint: "math.power", key: "node.Pow", category: "math", tags: []string{"math", "number"}, icon: "superscript",
			inputs: []extendedPort{number("base", "0"), number("exponent", "1")}, output: result(numberType), errors: append(errorSpec(mathDomainErrorCode, "evaluation"), resultErrors...),
			conformance: "finite-power/base+exponent/result", evaluate: powerNumber,
		},
		mathUnary(SquareRootNodeID, "math.square-root", "node.Sqrt", "square-root", number, result(numberType), append(errorSpec(mathDomainErrorCode, "evaluation"), resultErrors...), squareRoot),
		{
			id: EqualNodeID, entrypoint: "comparison.equal", key: "node.Eq", category: "comparison", tags: []string{"comparison"}, icon: "equal",
			inputs: []extendedPort{port("a", valueType, ""), port("b", valueType, "")}, output: result(booleanType),
			conformance: "canonical-equality/T+T/boolean", evaluate: equalValues,
		},
		{
			id: NotEqualNodeID, entrypoint: "comparison.not-equal", key: "node.NotEq", category: "comparison", tags: []string{"comparison"}, icon: "not-equal",
			inputs: []extendedPort{port("a", valueType, ""), port("b", valueType, "")}, output: result(booleanType),
			conformance: "canonical-inequality/T+T/boolean", evaluate: notEqualValues,
		},
		textNode(ReplaceNodeID, "text.replace", "node.Replace", "replace", []extendedPort{text("text", `""`), text("old", `""`), text("new", `""`), boolean("all", "true")}, result(stringType), nil, "unicode-replace", replaceText),
		textNode(SubstringNodeID, "text.substring", "node.Substring", "section", []extendedPort{text("text", `""`), integer("start", "0"), integer("length", "-1")}, result(stringType), nil, "unicode-rune-substring", substringText),
		textNode(TrimNodeID, "text.trim", "node.Trim", "spacing-vertical", []extendedPort{text("text", `""`)}, result(stringType), nil, "unicode-trim-space", trimText),
		textNode(UppercaseNodeID, "text.uppercase", "node.ToUpper", "letter-case-upper", []extendedPort{text("text", `""`)}, result(stringType), nil, "unicode-uppercase", uppercaseText),
		textNode(LowercaseNodeID, "text.lowercase", "node.ToLower", "letter-case-lower", []extendedPort{text("text", `""`)}, result(stringType), nil, "unicode-lowercase", lowercaseText),
		textNode(IndexOfNodeID, "text.index-of", "node.IndexOf", "text-search", []extendedPort{text("text", `""`), text("search", `""`)}, result(integerType), nil, "unicode-rune-index", indexOfText),
		textNode(StartsWithNodeID, "text.starts-with", "node.StartsWith", "arrow-bar-right", []extendedPort{text("text", `""`), text("prefix", `""`)}, result(booleanType), nil, "unicode-prefix", startsWithText),
		textNode(EndsWithNodeID, "text.ends-with", "node.EndsWith", "arrow-bar-left", []extendedPort{text("text", `""`), text("suffix", `""`)}, result(booleanType), nil, "unicode-suffix", endsWithText),
		textNode(RegexMatchNodeID, "text.regex-match", "node.RegexMatch", "regex", []extendedPort{text("text", `""`), text("pattern", `""`)}, result(booleanType), errorSpec(invalidRegexCode, "evaluation"), "re2-search", regexMatch),
		textNode(RegexExtractNodeID, "text.regex-extract", "node.RegexExtract", "regex", []extendedPort{text("text", `""`), text("pattern", `""`)}, result(stringType), errorSpec(invalidRegexCode, "evaluation"), "re2-first-capture", regexExtract),
		{
			id: ToStringNodeID, entrypoint: "conversion.to-string", key: "node.ToString", category: "conversion", tags: []string{"conversion", "text"}, icon: "text-caption",
			inputs: []extendedPort{port("value", valueType, "")}, output: result(stringType), conformance: "canonical-value-to-string", evaluate: valueToString,
		},
		{
			id: StringToNumberNodeID, entrypoint: "conversion.string-to-number", key: "node.ToNumber", category: "conversion", tags: []string{"conversion", "number"}, icon: "numbers",
			inputs: []extendedPort{text("text", `""`)}, output: result(numberType), errors: errorSpec(invalidNumberCode, "evaluation"), conformance: "strict-decimal-string-to-number", evaluate: stringToNumber,
		},
		{
			id: StringToBooleanNodeID, entrypoint: "conversion.string-to-boolean", key: "node.ToBool", category: "conversion", tags: []string{"conversion", "boolean"}, icon: "toggle-right",
			inputs: []extendedPort{text("text", `"false"`)}, output: result(booleanType), errors: errorSpec(invalidBooleanCode, "evaluation"), conformance: "strict-lowercase-string-to-boolean", evaluate: stringToBoolean,
		},
		{
			id: ParseJSONNodeID, entrypoint: "json.parse", key: "node.ParseJSON", category: "json", tags: []string{"json", "conversion"}, icon: "braces",
			inputs: []extendedPort{text("text", `"null"`)}, output: result(jsonType), errors: errorSpec(invalidJSONCode, "evaluation"), conformance: "strict-single-json-document", evaluate: parseJSONValue,
		},
		{
			id: ToJSONNodeID, entrypoint: "json.stringify", key: "node.ToJSON", category: "json", tags: []string{"json", "conversion"}, icon: "braces",
			inputs: []extendedPort{port("value", valueType, "")}, output: result(stringType), conformance: "canonical-json-stringify", evaluate: stringifyJSONValue,
		},
		{
			id: JSONPathNodeID, entrypoint: "json.path", key: "node.JsonPath", category: "json", tags: []string{"json", "query"}, icon: "route",
			inputs: []extendedPort{port("json", jsonType, ""), text("path", `"$"`)}, output: result(jsonType), errors: errorSpec(invalidJSONPathCode, "evaluation"), conformance: "bounded-json-path", evaluate: queryJSONPath,
		},
		{
			id: SelectNodeID, entrypoint: "logic.select", key: "node.Select", category: "logic", tags: []string{"logic", "generic"}, icon: "selector",
			inputs: []extendedPort{boolean("condition", "true"), port("when_true", valueType, ""), port("when_false", valueType, "")}, output: result(valueType), conformance: "strict-typed-select", evaluate: selectValue,
		},
		{
			id: MakePointNodeID, entrypoint: "geometry.make-point", key: "node.MakePoint", category: "geometry", tags: []string{"geometry", "point"}, icon: "map-pin",
			inputs: []extendedPort{number("x", "0"), number("y", "0"), port("unit", pointUnitType, `"ratio"`)}, output: result(pointType), conformance: "typed-point-construction", evaluate: makePoint,
		},
		{
			id: OffsetPointNodeID, entrypoint: "geometry.offset-point", key: "node.OffsetPoint", category: "geometry", tags: []string{"geometry", "point"}, icon: "arrows-move",
			inputs: []extendedPort{port("point", pointType, ""), number("offset_x", "0"), number("offset_y", "0")}, output: result(pointType), errors: resultErrors, conformance: "unit-preserving-point-offset", evaluate: offsetPoint,
		},
		{
			id: PointDistanceNodeID, entrypoint: "geometry.point-distance", key: "node.PointDistance", category: "geometry", tags: []string{"geometry", "point"}, icon: "ruler-measure",
			inputs: []extendedPort{port("begin", pointType, ""), port("end", pointType, "")}, output: result(numberType), errors: append(errorSpec(geometryUnitMismatchCode, "evaluation"), resultErrors...), conformance: "same-unit-point-distance", evaluate: pointDistance,
		},
		{
			id: RegionAroundPointNodeID, entrypoint: "geometry.region-around-point", key: "node.ROIAroundPoint", category: "geometry", tags: []string{"geometry", "region"}, icon: "crop",
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

func mathBinary(id, entrypoint, key, icon string, number func(string, string) extendedPort, output extendedPort, errors []nodecontract.ErrorSpec, evaluate InlineEvaluator) extendedNode {
	return extendedNode{id: id, entrypoint: entrypoint, key: key, category: "math", tags: []string{"math", "number"}, icon: icon,
		inputs: []extendedPort{number("a", "0"), number("b", "0")}, output: output, errors: errors,
		conformance: "strict-finite-number-binary", evaluate: evaluate}
}

func mathUnary(id, entrypoint, key, icon string, number func(string, string) extendedPort, output extendedPort, errors []nodecontract.ErrorSpec, evaluate InlineEvaluator) extendedNode {
	return extendedNode{id: id, entrypoint: entrypoint, key: key, category: "math", tags: []string{"math", "number"}, icon: icon,
		inputs: []extendedPort{number("value", "0")}, output: output, errors: errors,
		conformance: "strict-finite-number-unary", evaluate: evaluate}
}

func textNode(id, entrypoint, key, icon string, inputs []extendedPort, output extendedPort, errors []nodecontract.ErrorSpec, conformance string, evaluate InlineEvaluator) extendedNode {
	return extendedNode{id: id, entrypoint: entrypoint, key: key, category: "text", tags: []string{"text"}, icon: icon,
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
	return nodecontract.Seal(nodecontract.Draft{
		NodeTypeID: spec.id, ConfigSchemaRoot: configID, ConfigSchemaBundle: emptyConfigSchema(configID),
		Ports: nodecontract.PortSet{
			DataInputs: inputs, DataOutputs: []nodecontract.DataOutputPort{{ID: spec.output.id, Type: spec.output.typeExpr}},
			ExecInputs: []nodecontract.SignalPort{}, ExecOutputs: []nodecontract.SignalPort{}, ErrorOutputs: []nodecontract.SignalPort{},
		},
		Execution: pureDataExecution(), Instruction: nodecontract.Invoke(), CapabilityRequirements: []capability.Requirement{}, Errors: spec.errors,
		StatusEvents:      []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: spec.key + ".label", DescriptionKey: spec.key + ".description", Category: spec.category,
			Tags: spec.tags, Icon: spec.icon,
		},
	})
}
