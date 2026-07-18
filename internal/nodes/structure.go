package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func defineStructureNodes(types []datatype.Definition) ([]BuiltinDefinition, error) {
	definitions := make([]BuiltinDefinition, 0)
	for _, definition := range types {
		machine := definition.Machine()
		if machine.Structure == nil {
			continue
		}
		structure := machine.Structure
		outputs := make([]nodecontract.DataOutputPort, 0, len(structure.Fields))
		fields := make([]datatype.StructureField, 0, len(structure.Fields))
		for _, field := range structure.Fields {
			outputs = append(outputs, nodecontract.DataOutputPort{ID: field.ID, Type: field.Type})
			fields = append(fields, field)
		}
		configID := structure.BreakNodeTypeID + "/config"
		authoring := definition.Authoring()
		contract, err := nodecontract.Seal(nodecontract.Draft{
			NodeTypeID: structure.BreakNodeTypeID, Version: BuiltinNodeVersion,
			ConfigSchemaRoot: configID, ConfigSchemaBundle: emptyConfigSchema(configID),
			Ports: nodecontract.PortSet{
				DataInputs:  []nodecontract.DataInputPort{{ID: "value", Type: datatype.RefExpression(definition.TypeRef()), Required: true}},
				DataOutputs: outputs, ExecInputs: []nodecontract.SignalPort{}, ExecOutputs: []nodecontract.SignalPort{}, ErrorOutputs: []nodecontract.SignalPort{},
			},
			Execution: pureDataExecution(), Instruction: nodecontract.Invoke(),
			CapabilityRequirements: []capability.Requirement{}, Errors: []nodecontract.ErrorSpec{}, StatusEvents: []nodecontract.StatusEventSpec{},
			ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
			Authoring: nodecontract.Authoring{
				TitleKey: authoring.BreakTitleKey, DescriptionKey: authoring.BreakDescriptionKey, Category: "data",
				Tags: structureTags(definition.TypeRef().TypeID, structureFieldIDs(fields)), Icon: authoring.Icon,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("seal structure node for %q: %w", definition.TypeRef().TypeID, err)
		}
		evaluate := structureEvaluator(fields)
		builtin, err := defineBuiltin(contract, structureEntrypoint(structure.BreakNodeTypeID), "v1", "typed-structure-break/value/fields", evaluate)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, builtin)
	}
	return definitions, nil
}

func structureEvaluator(source []datatype.StructureField) InlineEvaluator {
	fields := append([]datatype.StructureField(nil), source...)
	return func(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
		var value map[string]json.RawMessage
		if err := json.Unmarshal(inputs["value"], &value); err != nil {
			return nil, fmt.Errorf("decode structured value: %w", err)
		}
		outputs := make(map[string]json.RawMessage, len(fields))
		for _, field := range fields {
			raw, ok := value[field.JSONKey]
			if !ok {
				return nil, fmt.Errorf("structured value is missing required field %q", field.JSONKey)
			}
			outputs[field.ID] = append(json.RawMessage(nil), raw...)
		}
		return outputs, nil
	}
}

func structureFieldIDs(fields []datatype.StructureField) []string {
	result := make([]string, len(fields))
	for index, field := range fields {
		result[index] = field.ID
	}
	return result
}

func structureEntrypoint(nodeTypeID string) string {
	segments := strings.Split(strings.Trim(nodeTypeID, "/"), "/")
	return "structure." + strings.ReplaceAll(segments[len(segments)-1], "-", "_")
}

func structureTags(typeID string, fields []string) []string {
	segments := strings.FieldsFunc(typeID, func(char rune) bool { return char == '/' || char == '-' || char == '_' })
	result := []string{"break", "structure", "field"}
	result = append(result, segments...)
	result = append(result, fields...)
	return result
}
