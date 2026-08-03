package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	PointerButtonTypeID = "https://schemas.yotta.dev/types/automation/pointer-button/v1"
	PointerMotionTypeID = "https://schemas.yotta.dev/types/automation/pointer-motion/v1"
	KeyCodeTypeID       = "https://schemas.yotta.dev/types/automation/key-code/v1"
	HeldInputTypeID     = "https://schemas.yotta.dev/types/automation/held-input/v1"

	ClickPointerNodeID        = "https://schemas.yotta.dev/nodes/automation/click-pointer"
	MovePointerNodeID         = "https://schemas.yotta.dev/nodes/automation/move-pointer"
	ScrollPointerNodeID       = "https://schemas.yotta.dev/nodes/automation/scroll-pointer"
	DragPointerNodeID         = "https://schemas.yotta.dev/nodes/automation/drag-pointer"
	MovePointerRelativeNodeID = "https://schemas.yotta.dev/nodes/automation/move-pointer-relative"
	PressKeysNodeID           = "https://schemas.yotta.dev/nodes/automation/press-keys"
	TypeTextNodeID            = "https://schemas.yotta.dev/nodes/automation/type-text"
	HoldKeysNodeID            = "https://schemas.yotta.dev/nodes/automation/hold-keys"
	HoldPointerButtonNodeID   = "https://schemas.yotta.dev/nodes/automation/hold-pointer-button"
	ReleaseHeldInputNodeID    = "https://schemas.yotta.dev/nodes/automation/release-held-input"

	ClickPointerEffectID        = "https://schemas.yotta.dev/effects/automation/click-pointer/v1"
	MovePointerEffectID         = "https://schemas.yotta.dev/effects/automation/move-pointer/v1"
	ScrollPointerEffectID       = "https://schemas.yotta.dev/effects/automation/scroll-pointer/v1"
	DragPointerEffectID         = "https://schemas.yotta.dev/effects/automation/drag-pointer/v1"
	MovePointerRelativeEffectID = "https://schemas.yotta.dev/effects/automation/move-pointer-relative/v1"
	PressKeysEffectID           = "https://schemas.yotta.dev/effects/automation/press-keys/v1"
	TypeTextEffectID            = "https://schemas.yotta.dev/effects/automation/type-text/v1"
	HoldKeysEffectID            = "https://schemas.yotta.dev/effects/automation/hold-keys/v1"
	HoldPointerButtonEffectID   = "https://schemas.yotta.dev/effects/automation/hold-pointer-button/v1"
	ReleaseHeldInputEffectID    = "https://schemas.yotta.dev/effects/automation/release-held-input/v1"
)

type automationInputTypes struct {
	stringRef   datatype.TypeRef
	integerRef  datatype.TypeRef
	booleanRef  datatype.TypeRef
	pointRef    datatype.TypeRef
	durationRef datatype.TypeRef
	buttonRef   datatype.TypeRef
	motionRef   datatype.TypeRef
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

func sealAutomationInputTypes() (datatype.Definition, datatype.Definition, datatype.Definition, error) {
	buttonSchema := json.RawMessage(fmt.Sprintf(`{
		"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"string",
		"enum":["left","right","middle"]
	}`, PointerButtonTypeID+"/schema"))
	button, err := sealStructuredType(PointerButtonTypeID, buttonSchema, datatype.Authoring{
		TitleKey: "type.automation.pointer_button.title", DescriptionKey: "type.automation.pointer_button.description", Color: "#fb7185", Icon: "pointer",
	})
	if err != nil {
		return datatype.Definition{}, datatype.Definition{}, datatype.Definition{}, err
	}
	motionSchema := json.RawMessage(fmt.Sprintf(`{
		"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"string",
		"enum":["instant","linear","bezier"]
	}`, PointerMotionTypeID+"/schema"))
	motion, err := sealStructuredType(PointerMotionTypeID, motionSchema, datatype.Authoring{
		TitleKey: "type.automation.pointer_motion.title", DescriptionKey: "type.automation.pointer_motion.description", Color: "#38bdf8", Icon: "route",
	})
	if err != nil {
		return datatype.Definition{}, datatype.Definition{}, datatype.Definition{}, err
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
	return button, motion, keyCode, err
}

func sealHeldInputType() (datatype.Definition, error) {
	const schemaID = HeldInputTypeID + "/schema"
	return datatype.SealDefinition(datatype.DefinitionDraft{
		TypeID: HeldInputTypeID, SchemaDialect: datatype.JSONSchemaDialect, SchemaRoot: schemaID,
		SchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"string","const":"runtime-held-input"
		}`, schemaID))}},
		Representations: []datatype.RepresentationSpec{{Kind: datatype.RepresentationHandleRef, Codec: datatype.CodecHandleRefV1}},
		Authoring: datatype.Authoring{
			TitleKey: "type.automation.held_input.title", DescriptionKey: "type.automation.held_input.description", Color: "#f97316", Icon: "hand-stop",
		},
	})
}

func defineAutomationHeldInputNodes(types automationInputTypes, heldType datatype.TypeRef) ([]BuiltinDefinition, []nodecontract.Contract, error) {
	heldExpression := datatype.RefExpression(heldType)
	keyListType := datatype.ListExpression(datatype.RefExpression(types.keyCodeRef))
	start := func(id, effectID, entrypoint, titleKey, icon, operation, conformance string, inputs []nodecontract.DataInputPort) (BuiltinDefinition, nodecontract.Contract, error) {
		schemaID := id + "/config"
		contract, err := nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
			NodeTypeID: id, ConfigSchemaRoot: schemaID, ConfigSchemaBundle: automationSlotSchema(schemaID),
			Ports: nodecontract.PortSet{
				DataInputs:  inputs,
				DataOutputs: []nodecontract.DataOutputPort{{ID: "held", Type: heldExpression, ResourceLease: &nodecontract.ResourceLeaseBinding{TargetID: "target", Operations: []string{installed.OperationReleaseHeld}}}},
				ExecInputs:  signalList("in"), ExecOutputs: signalList("completed"), ErrorOutputs: signalList("failed"),
			},
			Execution: automationEffectExecution(effectID), Instruction: nodecontract.Invoke(),
			ConfiguredTargets: automationTargetSpec("target", installed.TargetKindDesktopWindow),
			Errors:            automationInputErrors(), ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
			Authoring: nodecontract.Authoring{TitleKey: titleKey + ".title", DescriptionKey: titleKey + ".description", Category: "automation", Tags: []string{"automation", "input", "held", operation}, Icon: icon},
		})
		if err != nil {
			return BuiltinDefinition{}, nodecontract.Contract{}, err
		}
		definition, err := defineBuiltin(contract, entrypoint, "v1", conformance, nil)
		return definition, contract, err
	}
	holdKeys, holdKeysContract, err := start(
		HoldKeysNodeID, HoldKeysEffectID, "automation.hold-keys", "node.automation.holdKeys", "keyboard",
		installed.OperationHoldKeys, "exact-target/run-owned-held-keys/v1",
		[]nodecontract.DataInputPort{{ID: "keys", Type: keyListType, Required: true}},
	)
	if err != nil {
		return nil, nil, err
	}
	holdButton, holdButtonContract, err := start(
		HoldPointerButtonNodeID, HoldPointerButtonEffectID, "automation.hold-pointer-button", "node.automation.holdPointerButton", "hand-click",
		installed.OperationHoldButton, "exact-target/run-owned-held-button/v1",
		[]nodecontract.DataInputPort{{ID: "point", Type: datatype.RefExpression(types.pointRef), Required: true}, {ID: "button", Type: datatype.RefExpression(types.buttonRef), Required: true, Default: rawDefault(`"left"`)}},
	)
	if err != nil {
		return nil, nil, err
	}
	releaseSchemaID := ReleaseHeldInputNodeID + "/config"
	releaseContract, err := nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
		NodeTypeID: ReleaseHeldInputNodeID, ConfigSchemaRoot: releaseSchemaID, ConfigSchemaBundle: automationSlotSchema(releaseSchemaID),
		Ports: nodecontract.PortSet{
			DataInputs: []nodecontract.DataInputPort{{ID: "held", Type: heldExpression, Required: true, ResourceLease: &nodecontract.ResourceLeaseBinding{TargetID: "target", Operations: []string{installed.OperationReleaseHeld}}}},
			ExecInputs: signalList("in"), ExecOutputs: signalList("completed"), ErrorOutputs: signalList("failed"),
		},
		Execution: automationEffectExecution(ReleaseHeldInputEffectID), Instruction: nodecontract.Invoke(),
		ConfiguredTargets: automationTargetSpec("target", installed.TargetKindDesktopWindow),
		Errors:            automationInputErrors(), ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{TitleKey: "node.automation.releaseHeldInput.title", DescriptionKey: "node.automation.releaseHeldInput.description", Category: "automation", Tags: []string{"automation", "input", "held", "release"}, Icon: "hand-stop"},
	})
	if err != nil {
		return nil, nil, err
	}
	release, err := defineBuiltin(releaseContract, "automation.release-held-input", "v1", "run-owned-held-input/release/v1", nil)
	if err != nil {
		return nil, nil, err
	}
	return []BuiltinDefinition{holdKeys, holdButton, release}, []nodecontract.Contract{holdKeysContract, holdButtonContract, releaseContract}, nil
}

func automationSlotSchema(schemaID string) []datatype.SchemaResource {
	return []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
		"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
		"properties":{"slot":{"type":"string","minLength":1,"maxLength":128,"pattern":"^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$",
			"x-yotta-title-key":"node.automation.config.slot.title","x-yotta-description-key":"node.automation.config.slot.description"}},
		"required":["slot"],"additionalProperties":false
	}`, schemaID))}}
}

func automationEffectExecution(effectID string) nodecontract.ExecutionSpec {
	return nodecontract.ExecutionSpec{
		Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{nodecontract.EffectID(effectID)}, Determinism: nodecontract.Recorded,
		Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
		Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutRequired,
	}
}

func defineAutomationInputNodes(types automationInputTypes) ([]BuiltinDefinition, []nodecontract.Contract, error) {
	stringType := datatype.RefExpression(types.stringRef)
	integerType := datatype.RefExpression(types.integerRef)
	booleanType := datatype.RefExpression(types.booleanRef)
	pointType := datatype.RefExpression(types.pointRef)
	durationType := datatype.RefExpression(types.durationRef)
	buttonType := datatype.RefExpression(types.buttonRef)
	motionType := datatype.RefExpression(types.motionRef)
	keyListType := datatype.ListExpression(datatype.RefExpression(types.keyCodeRef))
	point := func(id string) nodecontract.DataInputPort {
		return nodecontract.DataInputPort{ID: id, Type: pointType, Required: true, Default: rawDefault(`{"x":0.5,"y":0.5,"unit":"ratio"}`)}
	}
	duration := func(id string, value string) nodecontract.DataInputPort {
		return nodecontract.DataInputPort{ID: id, Type: durationType, Required: true, Default: rawDefault(value)}
	}
	button := nodecontract.DataInputPort{ID: "button", Type: buttonType, Required: true, Default: rawDefault(`"left"`)}
	motion := nodecontract.DataInputPort{ID: "motion", Type: motionType, Required: true, Default: rawDefault(`"linear"`)}
	specs := []automationInputNode{
		{
			id: ClickPointerNodeID, entrypoint: "automation.click-pointer", titleKey: "node.automation.clickPointer", icon: "pointer", operation: installed.OperationClick, effectID: ClickPointerEffectID,
			inputs: []nodecontract.DataInputPort{point("point"), button, duration("hold-duration", "50")}, conformance: "exact-target/atomic-pointer-click/v1",
		},
		{
			id: MovePointerNodeID, entrypoint: "automation.move-pointer", titleKey: "node.automation.movePointer", icon: "location", operation: installed.OperationMove, effectID: MovePointerEffectID,
			inputs: []nodecontract.DataInputPort{point("point"), duration("duration", "300"), motion}, conformance: "exact-target/pointer-move/v1",
		},
		{
			id: ScrollPointerNodeID, entrypoint: "automation.scroll-pointer", titleKey: "node.automation.scrollPointer", icon: "mouse", operation: installed.OperationScroll, effectID: ScrollPointerEffectID,
			inputs: []nodecontract.DataInputPort{point("point"), {ID: "notches", Type: integerType, Required: true, Default: rawDefault("1")}, {ID: "horizontal", Type: booleanType, Required: true, Default: rawDefault("false")}}, conformance: "exact-target/pointer-scroll/v1",
		},
		{
			id: DragPointerNodeID, entrypoint: "automation.drag-pointer", titleKey: "node.automation.dragPointer", icon: "drag-drop", operation: installed.OperationDrag, effectID: DragPointerEffectID,
			inputs: []nodecontract.DataInputPort{point("from"), point("to"), button, duration("duration", "300"), motion}, conformance: "exact-target/cooperative-pointer-drag/v1",
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
		targetKinds := []string{installed.TargetKindDesktopWindow, installed.TargetKindAndroidDevice, installed.TargetKindBrowserCDP}
		if spec.operation == installed.OperationMoveRelative {
			targetKinds = []string{installed.TargetKindDesktopWindow}
		} else if spec.operation == installed.OperationPressKeys {
			targetKinds = []string{installed.TargetKindDesktopWindow, installed.TargetKindBrowserCDP}
		}
		contract, err := sealAutomationInputNode(spec, targetKinds)
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

func sealAutomationInputNode(spec automationInputNode, targetKinds []string) (nodecontract.Contract, error) {
	schemaID := spec.id + "/config"
	return nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
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
		Instruction:       nodecontract.Invoke(),
		ConfiguredTargets: automationTargetSpec("target", targetKinds...),
		Errors:            automationInputErrors(), StatusEvents: []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: spec.titleKey + ".title", DescriptionKey: spec.titleKey + ".description", Category: "automation",
			Tags: []string{"automation", "input", spec.operation}, Icon: spec.icon,
		},
	})
}

func automationInputErrors() []nodecontract.ErrorSpec {
	codes := []string{
		installed.CodeInvalidRequest, installed.CodeTargetNotFound, installed.CodeTargetAmbiguous,
		installed.CodeInputFailed, installed.CodeUnsupportedHost, installed.CodeContractViolation,
	}
	result := make([]nodecontract.ErrorSpec, 0, len(codes))
	for _, code := range codes {
		result = append(result, nodecontract.ErrorSpec{Code: code, Category: "automation", RetryHint: false})
	}
	return result
}

func automationTargetSpec(id string, kinds ...string) []nodecontract.ConfiguredTargetSpec {
	return []nodecontract.ConfiguredTargetSpec{{
		ID: id, TargetSlot: "target", SlotConfigKey: "slot", TargetKinds: append([]string(nil), kinds...),
	}}
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
