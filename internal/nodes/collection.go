package nodes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	SplitNodeID        = "https://schemas.yotta.dev/nodes/collection/split"
	JoinNodeID         = "https://schemas.yotta.dev/nodes/collection/join"
	ListLengthNodeID   = "https://schemas.yotta.dev/nodes/collection/length"
	ListGetNodeID      = "https://schemas.yotta.dev/nodes/collection/get"
	ListContainsNodeID = "https://schemas.yotta.dev/nodes/collection/contains"
	ListAppendNodeID   = "https://schemas.yotta.dev/nodes/collection/append"
	ListSliceNodeID    = "https://schemas.yotta.dev/nodes/collection/slice"

	collectionIndexOutOfRangeCode = "collection.index_out_of_range"
)

type collectionPort struct {
	id           string
	typeExpr     datatype.TypeExpression
	defaultValue string
}

type collectionNode struct {
	id          string
	entrypoint  string
	icon        string
	inputs      []collectionPort
	output      collectionPort
	errors      []nodecontract.ErrorSpec
	conformance string
	evaluate    InlineEvaluator
}

func defineCollectionNodes(types primitiveTypes) ([]BuiltinDefinition, error) {
	stringType := datatype.RefExpression(types.stringRef)
	integerType := datatype.RefExpression(types.integerRef)
	booleanType := datatype.RefExpression(types.booleanRef)
	element := datatype.VariableExpression("T")
	equatableElement := datatype.VariableExpression("E", string(datatype.TraitEquatable))
	list := datatype.ListExpression(element)
	equatableList := datatype.ListExpression(equatableElement)
	stringList := datatype.ListExpression(stringType)
	port := func(id string, typeExpr datatype.TypeExpression, defaultValue string) collectionPort {
		return collectionPort{id: id, typeExpr: typeExpr, defaultValue: defaultValue}
	}
	result := func(typeExpr datatype.TypeExpression) collectionPort {
		return collectionPort{id: "result", typeExpr: typeExpr}
	}
	specs := []collectionNode{
		{
			id: SplitNodeID, entrypoint: "collection.split", icon: "separator",
			inputs: []collectionPort{port("text", stringType, `""`), port("separator", stringType, `","`)}, output: result(stringList),
			conformance: "unicode-string-split/text+separator/list-string", evaluate: splitCollection,
		},
		{
			id: JoinNodeID, entrypoint: "collection.join", icon: "join-straight",
			inputs: []collectionPort{port("list", stringList, ""), port("separator", stringType, `","`)}, output: result(stringType),
			conformance: "strict-string-list-join/list+separator/string", evaluate: joinCollection,
		},
		{
			id: ListLengthNodeID, entrypoint: "collection.length", icon: "list-numbers",
			inputs: []collectionPort{port("list", list, "")}, output: result(integerType),
			conformance: "typed-list-length/list-T/integer", evaluate: listLength,
		},
		{
			id: ListGetNodeID, entrypoint: "collection.get", icon: "list-details",
			inputs: []collectionPort{port("list", list, ""), port("index", integerType, "0")}, output: result(element),
			errors:      []nodecontract.ErrorSpec{{Code: collectionIndexOutOfRangeCode, Category: "evaluation", RetryHint: false}},
			conformance: "typed-list-index/list-T+integer/T", evaluate: listGet,
		},
		{
			id: ListContainsNodeID, entrypoint: "collection.contains", icon: "list-search",
			inputs: []collectionPort{port("list", equatableList, ""), port("value", equatableElement, "")}, output: result(booleanType),
			conformance: "canonical-list-membership/list-E-equatable+E-equatable/boolean", evaluate: listContains,
		},
		{
			id: ListAppendNodeID, entrypoint: "collection.append", icon: "playlist-add",
			inputs: []collectionPort{port("list", list, ""), port("item", element, "")}, output: result(list),
			conformance: "immutable-list-append/list-T+T/list-T", evaluate: listAppend,
		},
		{
			id: ListSliceNodeID, entrypoint: "collection.slice", icon: "cut",
			inputs: []collectionPort{port("list", list, ""), port("start", integerType, "0"), port("count", integerType, "-1")}, output: result(list),
			conformance: "bounded-list-slice/list-T+start+count/list-T", evaluate: listSlice,
		},
	}

	definitions := make([]BuiltinDefinition, 0, len(specs))
	for _, spec := range specs {
		contract, err := sealCollectionNode(spec)
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

func sealCollectionNode(spec collectionNode) (nodecontract.Contract, error) {
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
			TitleKey: builtinMessageKey(spec.entrypoint) + ".title", DescriptionKey: builtinMessageKey(spec.entrypoint) + ".description", Category: "collection",
			Tags: []string{"collection", "list"}, Icon: spec.icon,
		},
	})
}

func splitCollection(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	text, separator, err := stringPair(inputs, "text", "separator")
	if err != nil {
		return nil, err
	}
	parts := []string{}
	if text != "" {
		parts = strings.Split(text, separator)
	}
	return jsonResult(parts)
}

func joinCollection(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	var items []string
	if err := json.Unmarshal(inputs["list"], &items); err != nil {
		return nil, fmt.Errorf("decode string list: %w", err)
	}
	var separator string
	if err := json.Unmarshal(inputs["separator"], &separator); err != nil {
		return nil, fmt.Errorf("decode separator: %w", err)
	}
	return jsonResult(strings.Join(items, separator))
}

func listLength(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	items, err := decodeRawList(inputs["list"])
	if err != nil {
		return nil, err
	}
	return jsonResult(len(items))
}

func listGet(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	items, err := decodeRawList(inputs["list"])
	if err != nil {
		return nil, err
	}
	index, err := decodeInteger(inputs["index"])
	if err != nil {
		return nil, err
	}
	if index < 0 || index >= int64(len(items)) {
		return nil, &InlineFailure{Code: collectionIndexOutOfRangeCode, Cause: errors.New("list index is out of range")}
	}
	return map[string]json.RawMessage{"result": append(json.RawMessage(nil), items[index]...)}, nil
}

func listContains(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	items, err := decodeRawList(inputs["list"])
	if err != nil {
		return nil, err
	}
	needle, err := artifact.Canonicalize(inputs["value"])
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		canonical, err := artifact.Canonicalize(item)
		if err != nil {
			return nil, err
		}
		if bytes.Equal(canonical, needle) {
			return jsonResult(true)
		}
	}
	return jsonResult(false)
}

func listAppend(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	items, err := decodeRawList(inputs["list"])
	if err != nil {
		return nil, err
	}
	items = append(items, append(json.RawMessage(nil), inputs["item"]...))
	return jsonResult(items)
}

func listSlice(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	items, err := decodeRawList(inputs["list"])
	if err != nil {
		return nil, err
	}
	start, err := decodeInteger(inputs["start"])
	if err != nil {
		return nil, err
	}
	count, err := decodeInteger(inputs["count"])
	if err != nil {
		return nil, err
	}
	if start < 0 {
		start = 0
	}
	if start >= int64(len(items)) || count == 0 {
		return jsonResult([]json.RawMessage{})
	}
	remaining := int64(len(items)) - start
	length := remaining
	if count > 0 && count < remaining {
		length = count
	}
	result := append([]json.RawMessage(nil), items[int(start):int(start+length)]...)
	return jsonResult(result)
}

func decodeRawList(raw json.RawMessage) ([]json.RawMessage, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("decode list: %w", err)
	}
	if items == nil {
		return nil, errors.New("list must be a JSON array")
	}
	return items, nil
}

func decodeInteger(raw json.RawMessage) (int64, error) {
	var value json.Number
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, fmt.Errorf("decode integer: %w", err)
	}
	result, err := strconv.ParseInt(value.String(), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("decode integer: %w", err)
	}
	return result, nil
}
