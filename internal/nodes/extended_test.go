package nodes

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestExtendedPureDefinitionsAreStrictAndGeneratedFromTheCatalog(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	nodeIDs := []string{
		DivideNodeID, ModuloNodeID, NegateNodeID, AbsoluteNodeID, MinimumNodeID, MaximumNodeID,
		FloorNodeID, CeilingNodeID, RoundNodeID, ClampNodeID, PowerNodeID, SquareRootNodeID,
		IntegerAddNodeID, IntegerSubtractNodeID, IntegerMultiplyNodeID, IntegerModuloNodeID,
		IntegerNegateNodeID, IntegerAbsoluteNodeID, IntegerMinimumNodeID, IntegerMaximumNodeID, IntegerClampNodeID,
		EqualNodeID, NotEqualNodeID, ReplaceNodeID, SubstringNodeID, TrimNodeID, UppercaseNodeID,
		LowercaseNodeID, IndexOfNodeID, StartsWithNodeID, EndsWithNodeID, RegexMatchNodeID,
		RegexExtractNodeID, ToStringNodeID, StringToNumberNodeID, StringToIntegerNodeID,
		TruncateToIntegerNodeID, FloorToIntegerNodeID, CeilingToIntegerNodeID, RoundToIntegerNodeID, StringToBooleanNodeID,
		ParseJSONNodeID, ToJSONNodeID, JSONPathNodeID, SelectNodeID, MakePointNodeID,
		OffsetPointNodeID, PointDistanceNodeID, RegionAroundPointNodeID,
	}
	if len(nodeIDs) != 49 || len(builtins.Definitions()) != 148 {
		t.Fatalf("extended=%d total=%d", len(nodeIDs), len(builtins.Definitions()))
	}
	for _, nodeID := range nodeIDs {
		definition, ok := builtins.Definition(nodeID)
		if !ok || definition.EvaluateInline == nil {
			t.Fatalf("missing extended definition %q", nodeID)
		}
		machine := definition.Contract.Machine()
		if machine.Execution.Class != nodecontract.ExecutionPureData || machine.Execution.Determinism != nodecontract.Deterministic ||
			len(machine.Execution.Effects)+len(machine.CapabilityRequirements)+len(machine.Ports.ExecInputs)+len(machine.Ports.ExecOutputs)+len(machine.Ports.ErrorOutputs) != 0 {
			t.Fatalf("extended node %q contains non-pure semantics", nodeID)
		}
	}
	projection, err := nodeauthoring.Project(nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: GeneratorVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	unit, _ := projection.Type(PointUnitTypeID)
	point, _ := projection.Type(PointTypeID)
	region, _ := projection.Type(RegionTypeID)
	jsonType, _ := projection.Type(JSONTypeID)
	if unit.Control != nodeauthoring.ControlSelect || len(unit.Constraints.Enum) != 2 {
		t.Fatalf("point unit authoring = %#v", unit)
	}
	if point.EditorAdapter != "point" || point.Importance != "primary" || point.InlinePriority != 100 ||
		region.EditorAdapter != "region" || region.Importance != "primary" || region.InlinePriority != 80 ||
		jsonType.Control != nodeauthoring.ControlJSON {
		t.Fatalf("point/region/json authoring = %#v / %#v / %#v", point, region, jsonType)
	}
	truncate, ok := projection.Node(TruncateToIntegerNodeID)
	if !ok || truncate.Conversion == nil || truncate.Conversion.Kind != nodecontract.ConversionLossy ||
		truncate.Conversion.Total || truncate.Conversion.InputPort != "value" || truncate.Conversion.OutputPort != "result" {
		t.Fatalf("truncate conversion projection = %#v", truncate.Conversion)
	}
	parser, ok := projection.Node(StringToIntegerNodeID)
	if !ok || parser.Conversion == nil || parser.Conversion.Kind != nodecontract.ConversionParser || parser.Conversion.Total {
		t.Fatalf("integer parser projection = %#v", parser.Conversion)
	}
}

func TestMathContractsDeclareOnlyReachableFailures(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string][]string{
		DivideNodeID:          {divisionByZeroCode, unrepresentableResultCode},
		ModuloNodeID:          {divisionByZeroCode},
		NegateNodeID:          {},
		AbsoluteNodeID:        {},
		MinimumNodeID:         {},
		MaximumNodeID:         {},
		FloorNodeID:           {},
		CeilingNodeID:         {},
		RoundNodeID:           {unrepresentableResultCode},
		ClampNodeID:           {},
		PowerNodeID:           {mathDomainErrorCode, unrepresentableResultCode},
		SquareRootNodeID:      {mathDomainErrorCode},
		IntegerAddNodeID:      {unrepresentableResultCode},
		IntegerSubtractNodeID: {unrepresentableResultCode},
		IntegerMultiplyNodeID: {unrepresentableResultCode},
		IntegerModuloNodeID:   {divisionByZeroCode},
		IntegerNegateNodeID:   {},
		IntegerAbsoluteNodeID: {},
		IntegerMinimumNodeID:  {},
		IntegerMaximumNodeID:  {},
		IntegerClampNodeID:    {},
	}
	for nodeID, want := range expected {
		definition, ok := builtins.Definition(nodeID)
		if !ok {
			t.Fatalf("math definition %q is missing", nodeID)
		}
		declared := definition.Contract.Machine().Errors
		got := make([]string, len(declared))
		for index, item := range declared {
			got[index] = item.Code
		}
		if !slices.Equal(got, want) {
			t.Fatalf("%s errors = %v, want %v", nodeID, got, want)
		}
	}
}

func TestTextContractsDeclareOnlyReachableFailures(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string][]string{
		ConcatNodeID:       {},
		ContainsNodeID:     {},
		LengthNodeID:       {},
		ReplaceNodeID:      {},
		SubstringNodeID:    {},
		TrimNodeID:         {},
		UppercaseNodeID:    {},
		LowercaseNodeID:    {},
		IndexOfNodeID:      {},
		StartsWithNodeID:   {},
		EndsWithNodeID:     {},
		RegexMatchNodeID:   {invalidRegexCode},
		RegexExtractNodeID: {invalidRegexCode},
	}
	for nodeID, want := range expected {
		definition, ok := builtins.Definition(nodeID)
		if !ok {
			t.Fatalf("text definition %q is missing", nodeID)
		}
		declared := definition.Contract.Machine().Errors
		got := make([]string, len(declared))
		for index, item := range declared {
			got[index] = item.Code
		}
		if !slices.Equal(got, want) {
			t.Fatalf("%s errors = %v, want %v", nodeID, got, want)
		}
	}
}

func TestConversionContractsDeclareOnlyReachableFailures(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string][]string{
		BlobToStreamNodeID:      {"conversion.blob_to_stream_failed"},
		StreamToBlobNodeID:      {"conversion.stream_to_blob_failed"},
		ToStringNodeID:          {},
		StringToNumberNodeID:    {invalidNumberCode},
		StringToIntegerNodeID:   {invalidIntegerCode},
		TruncateToIntegerNodeID: {invalidIntegerCode},
		FloorToIntegerNodeID:    {invalidIntegerCode},
		CeilingToIntegerNodeID:  {invalidIntegerCode},
		RoundToIntegerNodeID:    {invalidIntegerCode},
		StringToBooleanNodeID:   {invalidBooleanCode},
	}
	for nodeID, want := range expected {
		definition, ok := builtins.Definition(nodeID)
		if !ok {
			t.Fatalf("conversion definition %q is missing", nodeID)
		}
		declared := definition.Contract.Machine().Errors
		got := make([]string, len(declared))
		for index, item := range declared {
			got[index] = item.Code
		}
		if !slices.Equal(got, want) {
			t.Fatalf("%s errors = %v, want %v", nodeID, got, want)
		}
	}
}

func TestJSONContractsAndInlineGenericsDeclareExactBoundaries(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string][]string{
		ParseJSONNodeID: {invalidJSONCode},
		JSONPathNodeID:  {invalidJSONPathCode},
		ToJSONNodeID:    {},
	}
	for nodeID, want := range expected {
		definition, ok := builtins.Definition(nodeID)
		if !ok {
			t.Fatalf("JSON definition %q is missing", nodeID)
		}
		declared := definition.Contract.Machine().Errors
		got := make([]string, len(declared))
		for index, item := range declared {
			got[index] = item.Code
		}
		if !slices.Equal(got, want) {
			t.Fatalf("%s errors = %v, want %v", nodeID, got, want)
		}
	}
	for nodeID, portIDs := range map[string][]string{
		ToStringNodeID: {"value"},
		ToJSONNodeID:   {"value"},
		SelectNodeID:   {"when_true", "when_false"},
	} {
		definition, _ := builtins.Definition(nodeID)
		ports := definition.Contract.Machine().Ports.DataInputs
		for _, portID := range portIDs {
			found := false
			for _, port := range ports {
				if port.ID != portID {
					continue
				}
				found = true
				if port.Type.Kind != datatype.TypeExpressionVariable ||
					!slices.Equal(port.Type.Constraints, []string{string(datatype.TraitObservable)}) {
					t.Fatalf("%s.%s type = %#v", nodeID, portID, port.Type)
				}
			}
			if !found {
				t.Fatalf("%s input %s is missing", nodeID, portID)
			}
		}
	}
}

func TestExtendedEvaluatorsReportDeclaredTerminalFailures(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	evaluate := func(nodeID string, inputs map[string]json.RawMessage) (map[string]json.RawMessage, error) {
		t.Helper()
		definition, _ := builtins.Definition(nodeID)
		return definition.EvaluateInline(context.Background(), inputs, nil)
	}
	if result, err := evaluate(StringToNumberNodeID, map[string]json.RawMessage{"text": json.RawMessage(`"12.5"`)}); err != nil || string(result["result"]) != "12.5" {
		t.Fatalf("string-to-number = %s, %v", result["result"], err)
	}
	if result, err := evaluate(ParseJSONNodeID, map[string]json.RawMessage{"text": json.RawMessage(`"{\"b\":2,\"a\":1}"`)}); err != nil || string(result["result"]) != `{"a":1,"b":2}` {
		t.Fatalf("parse JSON = %s, %v", result["result"], err)
	}
	if result, err := evaluate(JSONPathNodeID, map[string]json.RawMessage{"json": json.RawMessage(`{"a":["节点"]}`), "path": json.RawMessage(`"$.a[0]"`)}); err != nil || string(result["result"]) != `"节点"` {
		t.Fatalf("JSON path = %s, %v", result["result"], err)
	}
	for _, failureCase := range []struct {
		nodeID string
		inputs map[string]json.RawMessage
		code   string
	}{
		{DivideNodeID, map[string]json.RawMessage{"a": json.RawMessage(`1`), "b": json.RawMessage(`0`)}, divisionByZeroCode},
		{ModuloNodeID, map[string]json.RawMessage{"a": json.RawMessage(`1`), "b": json.RawMessage(`0`)}, divisionByZeroCode},
		{PowerNodeID, map[string]json.RawMessage{"base": json.RawMessage(`-1`), "exponent": json.RawMessage(`0.5`)}, mathDomainErrorCode},
		{PowerNodeID, map[string]json.RawMessage{"base": json.RawMessage(`0`), "exponent": json.RawMessage(`-1`)}, unrepresentableResultCode},
		{SquareRootNodeID, map[string]json.RawMessage{"value": json.RawMessage(`-1`)}, mathDomainErrorCode},
		{IntegerModuloNodeID, map[string]json.RawMessage{"a": json.RawMessage(`1`), "b": json.RawMessage(`0`)}, divisionByZeroCode},
		{IntegerAddNodeID, map[string]json.RawMessage{"a": json.RawMessage(`9007199254740991`), "b": json.RawMessage(`1`)}, unrepresentableResultCode},
		{RegexMatchNodeID, map[string]json.RawMessage{"text": json.RawMessage(`"x"`), "pattern": json.RawMessage(`"["`)}, invalidRegexCode},
		{StringToBooleanNodeID, map[string]json.RawMessage{"text": json.RawMessage(`"yes"`)}, invalidBooleanCode},
	} {
		_, err := evaluate(failureCase.nodeID, failureCase.inputs)
		var failure *InlineFailure
		if !errors.As(err, &failure) || failure.Code != failureCase.code || failure.Output != "" {
			t.Fatalf("%s failure = %#v, %v", failureCase.nodeID, failure, err)
		}
	}
}

func TestEveryExtendedEvaluatorExecutesItsConformanceExample(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		nodeID string
		inputs map[string]json.RawMessage
		want   string
	}{
		{DivideNodeID, rawInputs("a", `8`, "b", `2`), `4`},
		{ModuloNodeID, rawInputs("a", `5`, "b", `2`), `1`},
		{NegateNodeID, rawInputs("value", `3`), `-3`},
		{AbsoluteNodeID, rawInputs("value", `-3`), `3`},
		{MinimumNodeID, rawInputs("a", `2`, "b", `3`), `2`},
		{MaximumNodeID, rawInputs("a", `2`, "b", `3`), `3`},
		{FloorNodeID, rawInputs("value", `2.9`), `2`},
		{CeilingNodeID, rawInputs("value", `2.1`), `3`},
		{RoundNodeID, rawInputs("value", `1.235`, "digits", `2`), `1.24`},
		{ClampNodeID, rawInputs("value", `5`, "minimum", `10`, "maximum", `0`), `5`},
		{PowerNodeID, rawInputs("base", `2`, "exponent", `3`), `8`},
		{SquareRootNodeID, rawInputs("value", `9`), `3`},
		{IntegerAddNodeID, rawInputs("a", `2`, "b", `3`), `5`},
		{IntegerSubtractNodeID, rawInputs("a", `2`, "b", `3`), `-1`},
		{IntegerMultiplyNodeID, rawInputs("a", `-2`, "b", `3`), `-6`},
		{IntegerModuloNodeID, rawInputs("a", `7`, "b", `3`), `1`},
		{IntegerNegateNodeID, rawInputs("value", `-3`), `3`},
		{IntegerAbsoluteNodeID, rawInputs("value", `-3`), `3`},
		{IntegerMinimumNodeID, rawInputs("a", `2`, "b", `3`), `2`},
		{IntegerMaximumNodeID, rawInputs("a", `2`, "b", `3`), `3`},
		{IntegerClampNodeID, rawInputs("value", `12`, "minimum", `10`, "maximum", `0`), `10`},
		{EqualNodeID, rawInputs("a", `{"b":2,"a":1}`, "b", `{"a":1,"b":2}`), `true`},
		{NotEqualNodeID, rawInputs("a", `1`, "b", `2`), `true`},
		{ReplaceNodeID, rawInputs("text", `"a-a"`, "old", `"a"`, "new", `"x"`, "all", `false`), `"x-a"`},
		{SubstringNodeID, rawInputs("text", `"a节点b"`, "start", `1`, "length", `2`), `"节点"`},
		{TrimNodeID, rawInputs("text", `" 节点 "`), `"节点"`},
		{UppercaseNodeID, rawInputs("text", `"Go节点"`), `"GO节点"`},
		{LowercaseNodeID, rawInputs("text", `"Go"`), `"go"`},
		{IndexOfNodeID, rawInputs("text", `"a节点b"`, "search", `"点"`), `2`},
		{StartsWithNodeID, rawInputs("text", `"Yotta节点"`, "prefix", `"Yotta"`), `true`},
		{EndsWithNodeID, rawInputs("text", `"Yotta节点"`, "suffix", `"节点"`), `true`},
		{RegexMatchNodeID, rawInputs("text", `"abc12"`, "pattern", `"\\d+"`), `true`},
		{RegexExtractNodeID, rawInputs("text", `"abc12"`, "pattern", `"([0-9]+)"`), `"12"`},
		{ToStringNodeID, rawInputs("value", `"hello"`), `"hello"`},
		{StringToIntegerNodeID, rawInputs("text", `"42"`), `42`},
		{TruncateToIntegerNodeID, rawInputs("value", `-1.9`), `-1`},
		{FloorToIntegerNodeID, rawInputs("value", `-1.1`), `-2`},
		{CeilingToIntegerNodeID, rawInputs("value", `1.1`), `2`},
		{RoundToIntegerNodeID, rawInputs("value", `1.5`), `2`},
		{StringToNumberNodeID, rawInputs("text", `"12.5"`), `12.5`},
		{StringToBooleanNodeID, rawInputs("text", `"false"`), `false`},
		{ParseJSONNodeID, rawInputs("text", `"{\"b\":2,\"a\":1}"`), `{"a":1,"b":2}`},
		{ToJSONNodeID, rawInputs("value", `{"b":2,"a":1}`), `"{\"a\":1,\"b\":2}"`},
		{JSONPathNodeID, rawInputs("json", `{"items":[{"x":1},{"x":2}]}`, "path", `"$.items[*].x"`), `[1,2]`},
		{SelectNodeID, rawInputs("condition", `false`, "when_true", `1`, "when_false", `2`), `2`},
		{MakePointNodeID, rawInputs("x", `0.25`, "y", `0.75`, "unit", `"ratio"`), `{"unit":"ratio","x":0.25,"y":0.75}`},
		{OffsetPointNodeID, rawInputs("point", `{"x":0.9,"y":0.1,"unit":"ratio"}`, "offset_x", `0.5`, "offset_y", `-0.5`), `{"unit":"ratio","x":1,"y":0}`},
		{PointDistanceNodeID, rawInputs("begin", `{"x":0,"y":0,"unit":"px"}`, "end", `{"x":3,"y":4,"unit":"px"}`), `5`},
		{RegionAroundPointNodeID, rawInputs("center", `{"x":0.95,"y":0.05,"unit":"ratio"}`, "width", `0.4`, "height", `0.2`), `{"height":0.2,"unit":"ratio","width":0.4,"x":0.6,"y":0}`},
	}
	for _, test := range cases {
		t.Run(test.nodeID, func(t *testing.T) {
			definition, ok := builtins.Definition(test.nodeID)
			if !ok {
				t.Fatal("definition is missing")
			}
			outputs, err := definition.EvaluateInline(context.Background(), test.inputs, nil)
			if err != nil {
				t.Fatal(err)
			}
			got, err := artifact.Canonicalize(outputs["result"])
			if err != nil {
				t.Fatal(err)
			}
			want, err := artifact.Canonicalize([]byte(test.want))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(want) {
				t.Fatalf("result = %s, want %s", got, want)
			}
		})
	}
}

func rawInputs(values ...string) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(values)/2)
	for index := 0; index < len(values); index += 2 {
		result[values[index]] = json.RawMessage(values[index+1])
	}
	return result
}

func TestExtendedEvaluatorsEnforceBoundarySemantics(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	evaluate := func(nodeID string, inputs map[string]json.RawMessage) (json.RawMessage, error) {
		t.Helper()
		definition, _ := builtins.Definition(nodeID)
		outputs, err := definition.EvaluateInline(context.Background(), inputs, nil)
		return outputs["result"], err
	}
	for _, test := range []struct {
		nodeID string
		inputs map[string]json.RawMessage
		want   string
	}{
		{ReplaceNodeID, rawInputs("text", `"a-a"`, "old", `"a"`, "new", `"x"`, "all", `true`), `"x-x"`},
		{ReplaceNodeID, rawInputs("text", `"abc"`, "old", `""`, "new", `"x"`, "all", `true`), `"abc"`},
		{SubstringNodeID, rawInputs("text", `"abc"`, "start", `-2`, "length", `0`), `""`},
		{SubstringNodeID, rawInputs("text", `"abc"`, "start", `99`, "length", `-1`), `""`},
		{IndexOfNodeID, rawInputs("text", `"abc"`, "search", `"z"`), `-1`},
		{RegexExtractNodeID, rawInputs("text", `"abc"`, "pattern", `"z+"`), `""`},
		{RegexExtractNodeID, rawInputs("text", `"abc"`, "pattern", `"b"`), `"b"`},
		{ToStringNodeID, rawInputs("value", `{"b":2,"a":1}`), `"{\"a\":1,\"b\":2}"`},
		{StringToBooleanNodeID, rawInputs("text", `"true"`), `true`},
		{JSONPathNodeID, rawInputs("json", `{"a":1}`, "path", `"$.missing"`), `null`},
		{JSONPathNodeID, rawInputs("json", `{"items":[{"x":1},{"y":2},{"x":3}]}`, "path", `"$.items[*].x"`), `[1,null,3]`},
		{SelectNodeID, rawInputs("condition", `true`, "when_true", `1`, "when_false", `2`), `1`},
		{RegionAroundPointNodeID, rawInputs("center", `{"x":5,"y":6,"unit":"px"}`, "width", `-2`, "height", `4`), `{"height":4,"unit":"px","width":0,"x":5,"y":4}`},
	} {
		got, err := evaluate(test.nodeID, test.inputs)
		if err != nil {
			t.Fatalf("%s: %v", test.nodeID, err)
		}
		canonical, _ := artifact.Canonicalize(got)
		want, _ := artifact.Canonicalize([]byte(test.want))
		if string(canonical) != string(want) {
			t.Fatalf("%s result = %s, want %s", test.nodeID, canonical, want)
		}
	}
	for _, test := range []struct {
		nodeID string
		inputs map[string]json.RawMessage
		code   string
	}{
		{ModuloNodeID, rawInputs("a", `1`, "b", `0`), divisionByZeroCode},
		{PowerNodeID, rawInputs("base", `-1`, "exponent", `0.5`), mathDomainErrorCode},
		{RegexExtractNodeID, rawInputs("text", `"x"`, "pattern", `"["`), invalidRegexCode},
		{StringToNumberNodeID, rawInputs("text", `" 1"`), invalidNumberCode},
		{StringToNumberNodeID, rawInputs("text", `"+1"`), invalidNumberCode},
		{StringToNumberNodeID, rawInputs("text", `"0x1p2"`), invalidNumberCode},
		{StringToNumberNodeID, rawInputs("text", `"01"`), invalidNumberCode},
		{StringToNumberNodeID, rawInputs("text", `"1e999"`), invalidNumberCode},
		{StringToIntegerNodeID, rawInputs("text", `"+1"`), invalidIntegerCode},
		{StringToIntegerNodeID, rawInputs("text", `"01"`), invalidIntegerCode},
		{StringToIntegerNodeID, rawInputs("text", `"9007199254740992"`), invalidIntegerCode},
		{ParseJSONNodeID, rawInputs("text", `"{"`), invalidJSONCode},
		{ParseJSONNodeID, rawInputs("text", `"1 2"`), invalidJSONCode},
		{ParseJSONNodeID, rawInputs("text", `"-0"`), invalidJSONCode},
		{JSONPathNodeID, rawInputs("json", `{}`, "path", `"a"`), invalidJSONPathCode},
		{JSONPathNodeID, rawInputs("json", `{}`, "path", `"$.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a.a"`), invalidJSONPathCode},
		{PointDistanceNodeID, rawInputs("begin", `{"x":0,"y":0,"unit":"px"}`, "end", `{"x":0,"y":0,"unit":"ratio"}`), geometryUnitMismatchCode},
	} {
		_, err := evaluate(test.nodeID, test.inputs)
		var failure *InlineFailure
		if !errors.As(err, &failure) || failure.Code != test.code {
			t.Fatalf("%s failure = %#v, %v", test.nodeID, failure, err)
		}
	}
}
