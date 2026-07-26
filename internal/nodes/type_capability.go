package nodes

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

type typeCapabilityRow struct {
	TypeID      string
	Traits      []datatype.Trait
	Produced    bool
	Consumed    bool
	Structured  bool
	Conversions int
	Waiver      string
}

var typeCapabilityWaivers = map[string]string{
	InputClipTypeID: "created and selected through the recording asset library",
	MacroTypeID:     "created and selected through the macro asset library",
}

func validateTypeCapabilityClosure(types []datatype.Definition, contracts []nodecontract.Contract) ([]typeCapabilityRow, error) {
	system, err := datatype.NewSystem(types)
	if err != nil {
		return nil, err
	}
	constraints := map[datatype.Trait]bool{}
	for _, contract := range contracts {
		for _, input := range contract.Machine().Ports.DataInputs {
			collectConstraints(input.Type, constraints)
		}
	}
	for _, trait := range []datatype.Trait{datatype.TraitDurable, datatype.TraitEquatable, datatype.TraitObservable} {
		if !constraints[trait] {
			return nil, fmt.Errorf("type capability closure has no generic consumer for trait %q", trait)
		}
	}

	rows := make([]typeCapabilityRow, 0, len(types))
	for _, definition := range types {
		machine := definition.Machine()
		ref := definition.TypeRef()
		row := typeCapabilityRow{TypeID: ref.TypeID, Traits: append([]datatype.Trait(nil), machine.Traits...), Structured: machine.Structure != nil}
		if rootSchemaType(definition) == "object" && machine.Structure == nil {
			return nil, fmt.Errorf("object type %q has no typed structure contract", ref.TypeID)
		}
		for _, contract := range contracts {
			node := contract.Machine()
			for _, output := range node.Ports.DataOutputs {
				if expressionContainsRef(output.Type, ref) {
					row.Produced = true
				}
			}
			for _, input := range node.Ports.DataInputs {
				if expressionAcceptsType(input.Type, ref, system) {
					row.Consumed = true
				}
			}
			if node.Conversion != nil && (expressionContainsRef(portInputType(node.Ports, node.Conversion.InputPort), ref) || expressionContainsRef(portOutputType(node.Ports, node.Conversion.OutputPort), ref)) {
				row.Conversions++
			}
		}
		row.Waiver = typeCapabilityWaivers[ref.TypeID]
		if !row.Produced && !authorableLiteral(definition) && row.Waiver == "" {
			return nil, fmt.Errorf("type %q has neither a producer nor a literal example", ref.TypeID)
		}
		if !row.Consumed {
			return nil, fmt.Errorf("type %q has no compatible consumer", ref.TypeID)
		}
		if machine.Structure != nil {
			if err := validateBreakNode(machine.Structure, ref, contracts); err != nil {
				return nil, fmt.Errorf("type %q: %w", ref.TypeID, err)
			}
		}
		if system.HasTrait(ref, datatype.TraitNumeric) && !hasAssignableCategoryConsumer(ref, "math", contracts, system) {
			return nil, fmt.Errorf("numeric type %q has no math consumer", ref.TypeID)
		}
		if system.HasTrait(ref, datatype.TraitOrdered) && !hasAssignableCategoryConsumer(ref, "comparison", contracts, system) {
			return nil, fmt.Errorf("ordered type %q has no comparison consumer", ref.TypeID)
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].TypeID < rows[j].TypeID })
	return rows, nil
}

func rootSchemaType(definition datatype.Definition) string {
	machine := definition.Machine()
	for _, resource := range machine.SchemaBundle {
		if resource.ID != machine.SchemaRoot {
			continue
		}
		var schema map[string]any
		if json.Unmarshal(resource.Schema, &schema) != nil {
			return ""
		}
		typeName, _ := schema["type"].(string)
		return typeName
	}
	return ""
}

func authorableLiteral(definition datatype.Definition) bool {
	authoring := definition.Authoring()
	if len(authoring.Examples) != 0 || authoring.EditorAdapter != "" {
		return true
	}
	machine := definition.Machine()
	inline := false
	for _, representation := range machine.Representations {
		inline = inline || representation.Kind == datatype.RepresentationInlineJSON
	}
	if !inline {
		return false
	}
	for _, resource := range machine.SchemaBundle {
		if resource.ID != machine.SchemaRoot {
			continue
		}
		typeName := rootSchemaType(definition)
		return typeName == "string" || typeName == "number" || typeName == "integer" || typeName == "boolean"
	}
	return false
}

func collectConstraints(expression datatype.TypeExpression, result map[datatype.Trait]bool) {
	switch expression.Kind {
	case datatype.TypeExpressionVariable:
		for _, raw := range expression.Constraints {
			result[datatype.Trait(raw)] = true
		}
	case datatype.TypeExpressionList:
		collectConstraints(*expression.Element, result)
	case datatype.TypeExpressionUnion:
		for _, member := range expression.Members {
			collectConstraints(member, result)
		}
	}
}

func expressionContainsRef(expression datatype.TypeExpression, ref datatype.TypeRef) bool {
	switch expression.Kind {
	case datatype.TypeExpressionRef:
		return expression.Ref != nil && *expression.Ref == ref
	case datatype.TypeExpressionList:
		return expressionContainsRef(*expression.Element, ref)
	case datatype.TypeExpressionUnion:
		for _, member := range expression.Members {
			if expressionContainsRef(member, ref) {
				return true
			}
		}
	}
	return false
}

func expressionAcceptsType(expression datatype.TypeExpression, ref datatype.TypeRef, system *datatype.System) bool {
	switch expression.Kind {
	case datatype.TypeExpressionRef:
		ok, err := system.AssignableRef(ref, *expression.Ref)
		return err == nil && ok
	case datatype.TypeExpressionVariable:
		if len(expression.Constraints) == 0 {
			return false
		}
		ok, err := system.Satisfies(ref, expression.Constraints)
		return err == nil && ok
	case datatype.TypeExpressionUnion:
		for _, member := range expression.Members {
			if expressionAcceptsType(member, ref, system) {
				return true
			}
		}
	}
	return false
}

func hasAssignableCategoryConsumer(ref datatype.TypeRef, category string, contracts []nodecontract.Contract, system *datatype.System) bool {
	for _, contract := range contracts {
		if contract.Authoring().Category != category {
			continue
		}
		for _, input := range contract.Machine().Ports.DataInputs {
			if input.Type.Kind != datatype.TypeExpressionRef {
				continue
			}
			ok, err := system.AssignableRef(ref, *input.Type.Ref)
			if err == nil && ok {
				return true
			}
		}
	}
	return false
}

func validateBreakNode(structure *datatype.StructureSpec, ref datatype.TypeRef, contracts []nodecontract.Contract) error {
	for _, contract := range contracts {
		if contract.NodeRef().NodeTypeID != structure.BreakNodeTypeID {
			continue
		}
		ports := contract.Machine().Ports
		if len(ports.DataInputs) != 1 || ports.DataInputs[0].Type.Kind != datatype.TypeExpressionRef || *ports.DataInputs[0].Type.Ref != ref {
			return errors.New("break node does not consume the exact structured type")
		}
		if len(ports.DataOutputs) != len(structure.Fields) {
			return errors.New("break node output count does not match structure fields")
		}
		for index, field := range structure.Fields {
			if ports.DataOutputs[index].ID != field.ID || !reflect.DeepEqual(ports.DataOutputs[index].Type, field.Type) {
				return fmt.Errorf("break node output %q does not match its structure field", field.ID)
			}
		}
		return nil
	}
	return errors.New("structure break node is missing")
}

func portInputType(ports nodecontract.PortSet, id string) datatype.TypeExpression {
	for _, port := range ports.DataInputs {
		if port.ID == id {
			return port.Type
		}
	}
	return datatype.TypeExpression{}
}

func portOutputType(ports nodecontract.PortSet, id string) datatype.TypeExpression {
	for _, port := range ports.DataOutputs {
		if port.ID == id {
			return port.Type
		}
	}
	return datatype.TypeExpression{}
}

func renderTypeCapabilityMatrix(rows []typeCapabilityRow) string {
	var builder strings.Builder
	builder.WriteString("## Type capability matrix\n\n")
	builder.WriteString("Generated closure view. A missing applicable capability fails Catalog construction.\n\n")
	builder.WriteString("| Type | Traits | Produced | Consumed | Structure break | Conversions | Waiver |\n")
	builder.WriteString("| --- | --- | --- | --- | --- | --- | --- |\n")
	for _, row := range rows {
		traits := make([]string, len(row.Traits))
		for index, trait := range row.Traits {
			traits[index] = string(trait)
		}
		fmt.Fprintf(&builder, "| `%s` | %s | %s | %s | %s | %d | %s |\n", row.TypeID, strings.Join(traits, ", "), yesNo(row.Produced), yesNo(row.Consumed), yesNo(row.Structured), row.Conversions, row.Waiver)
	}
	builder.WriteString("\n")
	return builder.String()
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
