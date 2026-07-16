package nodes31

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	PointerButtonTypeID = "https://schemas.yotta.dev/types/automation/pointer-button/v1"
	KeyCodeTypeID       = "https://schemas.yotta.dev/types/automation/key-code/v1"

	AutomationInputCapabilityID = "https://schemas.yotta.dev/capabilities/automation/input/v1"

	ClickPointerNodeID        = "https://schemas.yotta.dev/nodes/automation/click-pointer/v1"
	MovePointerNodeID         = "https://schemas.yotta.dev/nodes/automation/move-pointer/v1"
	ScrollPointerNodeID       = "https://schemas.yotta.dev/nodes/automation/scroll-pointer/v1"
	DragPointerNodeID         = "https://schemas.yotta.dev/nodes/automation/drag-pointer/v1"
	MovePointerRelativeNodeID = "https://schemas.yotta.dev/nodes/automation/move-pointer-relative/v1"
	PressKeysNodeID           = "https://schemas.yotta.dev/nodes/automation/press-keys/v1"
	TypeTextNodeID            = "https://schemas.yotta.dev/nodes/automation/type-text/v1"

	ClickPointerEffectID        = "https://schemas.yotta.dev/effects/automation/click-pointer/v1"
	MovePointerEffectID         = "https://schemas.yotta.dev/effects/automation/move-pointer/v1"
	ScrollPointerEffectID       = "https://schemas.yotta.dev/effects/automation/scroll-pointer/v1"
	DragPointerEffectID         = "https://schemas.yotta.dev/effects/automation/drag-pointer/v1"
	MovePointerRelativeEffectID = "https://schemas.yotta.dev/effects/automation/move-pointer-relative/v1"
	PressKeysEffectID           = "https://schemas.yotta.dev/effects/automation/press-keys/v1"
	TypeTextEffectID            = "https://schemas.yotta.dev/effects/automation/type-text/v1"
)

type automationInputTypes struct {
	stringRef   datatype.TypeRef
	integerRef  datatype.TypeRef
	booleanRef  datatype.TypeRef
	pointRef    datatype.TypeRef
	durationRef datatype.TypeRef
	buttonRef   datatype.TypeRef
	keyCodeRef  datatype.TypeRef
}

type automationInputNode struct {
	id          string
	entrypoint  string
	titleKey    string
	icon        string
	operation   string
	effectID    string
	inputs      []nodecontract.DataInputPort
	conformance string
}

func sealAutomationInputTypes() (datatype.Definition, datatype.Definition, error) {
	buttonSchema := json.RawMessage(fmt.Sprintf(`{
		"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"string",
		"enum":["left","right","middle"]
	}`, PointerButtonTypeID+"/schema"))
	button, err := sealStructuredType(PointerButtonTypeID, buttonSchema, datatype.Authoring{
		TitleKey: "type.automation.pointer_button.title", DescriptionKey: "type.automation.pointer_button.description", Color: "#fb7185", Icon: "pointer",
	})
	if err != nil {
		return datatype.Definition{}, datatype.Definition{}, err
	}
	keySchema := json.RawMessage(fmt.Sprintf(`{
		"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"string",
		"enum":[
			"A","B","C","D","E","F","G","H","I","J","K","L","M","N","O","P","Q","R","S","T","U","V","W","X","Y","Z",
			"0","1","2","3","4","5","6","7","8","9","F1","F2","F3","F4","F5","F6","F7","F8","F9","F10","F11","F12",
			"ESC","SPACE","ENTER","SHIFT","CTRL","ALT","TAB","BACKSPACE","DELETE","INSERT","HOME","END","PGUP","PGDN","UP","DOWN","LEFT","RIGHT",",",".","CAPSLOCK"
		]
	}`, KeyCodeTypeID+"/schema"))
	keyCode, err := sealStructuredType(KeyCodeTypeID, keySchema, datatype.Authoring{
		TitleKey: "type.automation.key_code.title", DescriptionKey: "type.automation.key_code.description", Color: "#a78bfa", Icon: "keyboard",
	})
	return button, keyCode, err
}

func sealAutomationInputCapability() (capability.Definition, error) {
	const scopeID = AutomationInputCapabilityID + "/scope"
	return capability.SealDefinition(capability.DefinitionDraft{
		CapabilityID: AutomationInputCapabilityID, Operations: installed.InputOperations(), TargetKinds: []string{installed.TargetKind},
		ScopeSchemaRoot: scopeID, ScopeSchemaBundle: []datatype.SchemaResource{{ID: scopeID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"operation":{"enum":["click","drag","move","move-relative","press-keys","scroll","type-text"]}},
			"required":["operation"],"additionalProperties":false
		}`, scopeID))}},
		Credential: capability.CredentialNone, Risk: capability.RiskDangerous, Consent: capability.ConsentOnce,
		ProviderABI: installed.ProviderABI,
	})
}

func defineAutomationInputNodes(types automationInputTypes, input capability.Definition) ([]BuiltinDefinition, []nodecontract.Contract, error) {
	stringType := datatype.RefExpression(types.stringRef)
	integerType := datatype.RefExpression(types.integerRef)
	booleanType := datatype.RefExpression(types.booleanRef)
	pointType := datatype.RefExpression(types.pointRef)
	durationType := datatype.RefExpression(types.durationRef)
	buttonType := datatype.RefExpression(types.buttonRef)
	keyListType := datatype.ListExpression(datatype.RefExpression(types.keyCodeRef))
	point := func(id string) nodecontract.DataInputPort {
		return nodecontract.DataInputPort{ID: id, Type: pointType, Required: true, Default: rawDefault(`{"x":0.5,"y":0.5,"unit":"ratio"}`)}
	}
	duration := func(id string, value string) nodecontract.DataInputPort {
		return nodecontract.DataInputPort{ID: id, Type: durationType, Required: true, Default: rawDefault(value)}
	}
	button := nodecontract.DataInputPort{ID: "button", Type: buttonType, Required: true, Default: rawDefault(`"left"`)}
	specs := []automationInputNode{
		{
			id: ClickPointerNodeID, entrypoint: "automation.click-pointer", titleKey: "node.automation.clickPointer", icon: "pointer", operation: installed.OperationClick, effectID: ClickPointerEffectID,
			inputs: []nodecontract.DataInputPort{point("point"), button, duration("hold-duration", "50")}, conformance: "exact-target/atomic-pointer-click/v1",
		},
		{
			id: MovePointerNodeID, entrypoint: "automation.move-pointer", titleKey: "node.automation.movePointer", icon: "location", operation: installed.OperationMove, effectID: MovePointerEffectID,
			inputs: []nodecontract.DataInputPort{point("point")}, conformance: "exact-target/pointer-move/v1",
		},
		{
			id: ScrollPointerNodeID, entrypoint: "automation.scroll-pointer", titleKey: "node.automation.scrollPointer", icon: "mouse", operation: installed.OperationScroll, effectID: ScrollPointerEffectID,
			inputs: []nodecontract.DataInputPort{point("point"), {ID: "notches", Type: integerType, Required: true, Default: rawDefault("1")}, {ID: "horizontal", Type: booleanType, Required: true, Default: rawDefault("false")}}, conformance: "exact-target/pointer-scroll/v1",
		},
		{
			id: DragPointerNodeID, entrypoint: "automation.drag-pointer", titleKey: "node.automation.dragPointer", icon: "drag-drop", operation: installed.OperationDrag, effectID: DragPointerEffectID,
			inputs: []nodecontract.DataInputPort{point("from"), point("to"), button, duration("duration", "300")}, conformance: "exact-target/cooperative-pointer-drag/v1",
		},
		{
			id: MovePointerRelativeNodeID, entrypoint: "automation.move-pointer-relative", titleKey: "node.automation.movePointerRelative", icon: "arrows-move", operation: installed.OperationMoveRelative, effectID: MovePointerRelativeEffectID,
			inputs: []nodecontract.DataInputPort{{ID: "delta-x", Type: integerType, Required: true, Default: rawDefault("0")}, {ID: "delta-y", Type: integerType, Required: true, Default: rawDefault("0")}, duration("duration", "0")}, conformance: "exact-target/cooperative-relative-pointer-move/v1",
		},
		{
			id: PressKeysNodeID, entrypoint: "automation.press-keys", titleKey: "node.automation.pressKeys", icon: "keyboard", operation: installed.OperationPressKeys, effectID: PressKeysEffectID,
			inputs: []nodecontract.DataInputPort{{ID: "keys", Type: keyListType, Required: true}, duration("hold-duration", "50")}, conformance: "exact-target/atomic-key-chord/v1",
		},
		{
			id: TypeTextNodeID, entrypoint: "automation.type-text", titleKey: "node.automation.typeText", icon: "text-size", operation: installed.OperationTypeText, effectID: TypeTextEffectID,
			inputs: []nodecontract.DataInputPort{{ID: "text", Type: stringType, Required: true}}, conformance: "exact-target/cooperative-unicode-text/v1",
		},
	}
	definitions := make([]BuiltinDefinition, 0, len(specs))
	contracts := make([]nodecontract.Contract, 0, len(specs))
	for _, spec := range specs {
		contract, err := sealAutomationInputNode(spec, input)
		if err != nil {
			return nil, nil, err
		}
		definition, err := defineBuiltin(contract, spec.entrypoint, "v1", spec.conformance, nil)
		if err != nil {
			return nil, nil, err
		}
		definitions = append(definitions, definition)
		contracts = append(contracts, contract)
	}
	return definitions, contracts, nil
}

func sealAutomationInputNode(spec automationInputNode, input capability.Definition) (nodecontract.Contract, error) {
	schemaID := spec.id + "/config"
	return nodecontract.Seal(nodecontract.Draft{
		NodeTypeID: spec.id, ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"slot":{"type":"string","minLength":1,"maxLength":128,"pattern":"^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$",
				"x-yotta-title-key":"node.automation.config.slot.title","x-yotta-description-key":"node.automation.config.slot.description"}},
			"required":["slot"],"additionalProperties":false
		}`, schemaID))}},
		Ports: nodecontract.PortSet{DataInputs: spec.inputs, DataOutputs: []nodecontract.DataOutputPort{}, ExecInputs: signalList("in"), ExecOutputs: signalList("completed"), ErrorOutputs: signalList("failed")},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{nodecontract.EffectID(spec.effectID)}, Determinism: nodecontract.Recorded,
			Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutRequired,
		},
		Instruction: nodecontract.Invoke(),
		CapabilityRequirements: []capability.Requirement{{
			ID: "target", Capability: input.Ref(), Operations: []string{spec.operation}, TargetSlot: "target", Scope: json.RawMessage(fmt.Sprintf(`{"operation":%q}`, spec.operation)),
		}},
		RequirementBindings: []nodecontract.RequirementBindingSpec{{RequirementID: "target", TargetSlotConfigKey: "slot"}},
		Errors:              automationInputErrors(), StatusEvents: []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: spec.titleKey + ".title", DescriptionKey: spec.titleKey + ".description", Category: "automation",
			Tags: []string{"automation", "input", spec.operation}, Icon: spec.icon,
		},
	})
}

func automationInputErrors() []nodecontract.ErrorSpec {
	codes := []string{
		installed.CodeInvalidRequest, installed.CodeIdentityChanged, installed.CodeTargetNotFound, installed.CodeTargetAmbiguous,
		installed.CodeInputFailed, installed.CodeUnsupportedHost, installed.CodeContractViolation,
	}
	result := make([]nodecontract.ErrorSpec, 0, len(codes))
	for _, code := range codes {
		result = append(result, nodecontract.ErrorSpec{Code: code, Category: "automation", RetryHint: false})
	}
	return result
}

func AutomationInputEffectID(nodeID string) (string, bool) {
	for _, pair := range [][2]string{
		{ClickPointerNodeID, ClickPointerEffectID}, {MovePointerNodeID, MovePointerEffectID},
		{ScrollPointerNodeID, ScrollPointerEffectID}, {DragPointerNodeID, DragPointerEffectID},
		{MovePointerRelativeNodeID, MovePointerRelativeEffectID}, {PressKeysNodeID, PressKeysEffectID},
		{TypeTextNodeID, TypeTextEffectID},
	} {
		if pair[0] == nodeID {
			return pair[1], true
		}
	}
	return "", false
}
