package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	WaitTemplateNodeID     = "https://schemas.yotta.dev/nodes/automation/wait-template"
	WaitTemplateGoneNodeID = "https://schemas.yotta.dev/nodes/automation/wait-template-gone"
	ClickTemplateNodeID    = "https://schemas.yotta.dev/nodes/automation/click-template"

	WaitTemplateEffectID     = "https://schemas.yotta.dev/effects/automation/wait-template/v1"
	WaitTemplateGoneEffectID = "https://schemas.yotta.dev/effects/automation/wait-template-gone/v1"
	ClickTemplateEffectID    = "https://schemas.yotta.dev/effects/automation/click-template/v1"

	AutomationTemplateWaitingStatus = "automation.template.waiting"
	AutomationTemplateMatchedStatus = "automation.template.matched"
	AutomationTemplateTimeoutStatus = "automation.template.timeout"
)

type automationTemplateTypes struct {
	imageRef    datatype.TypeRef
	numberRef   datatype.TypeRef
	integerRef  datatype.TypeRef
	booleanRef  datatype.TypeRef
	pointRef    datatype.TypeRef
	regionRef   datatype.TypeRef
	durationRef datatype.TypeRef
	buttonRef   datatype.TypeRef
}

type automationTemplateNode struct {
	id, effectID, entrypoint, titleKey, icon string
	kind                                     string
}

func defineAutomationTemplateNodes(types automationTemplateTypes, capture, input, blobRead capability.Definition) ([]BuiltinDefinition, []nodecontract.Contract, error) {
	specs := []automationTemplateNode{
		{id: WaitTemplateNodeID, effectID: WaitTemplateEffectID, entrypoint: "automation.wait-template", titleKey: "node.automation.waitTemplate", icon: "clock-search", kind: "wait"},
		{id: ClickTemplateNodeID, effectID: ClickTemplateEffectID, entrypoint: "automation.click-template", titleKey: "node.automation.clickTemplate", icon: "click", kind: "click"},
		{id: WaitTemplateGoneNodeID, effectID: WaitTemplateGoneEffectID, entrypoint: "automation.wait-template-gone", titleKey: "node.automation.waitTemplateGone", icon: "clock-off", kind: "gone"},
	}
	definitions := make([]BuiltinDefinition, 0, len(specs))
	contracts := make([]nodecontract.Contract, 0, len(specs))
	for _, spec := range specs {
		contract, err := sealAutomationTemplateNode(spec, types, capture, input, blobRead)
		if err != nil {
			return nil, nil, err
		}
		definition, err := defineBuiltin(contract, spec.entrypoint, "v1", "exact-target/blob-template/cancellable-poll/v1", nil)
		if err != nil {
			return nil, nil, err
		}
		definitions = append(definitions, definition)
		contracts = append(contracts, contract)
	}
	return definitions, contracts, nil
}

func sealAutomationTemplateNode(spec automationTemplateNode, types automationTemplateTypes, capture, input, blobRead capability.Definition) (nodecontract.Contract, error) {
	schemaID := spec.id + "/config"
	configSchema := json.RawMessage(fmt.Sprintf(`{
		"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
		"properties":{"slot":{"type":"string","minLength":1,"maxLength":128,"pattern":"^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$",
			"x-yotta-title-key":"node.automation.config.slot.title","x-yotta-description-key":"node.automation.config.slot.description"}},
		"required":["slot"],"additionalProperties":false
	}`, schemaID))
	inputs := automationTemplateInputs(types)
	outputs := []nodecontract.DataOutputPort{
		{ID: "matched", Type: datatype.RefExpression(types.booleanRef)},
		{ID: "score", Type: datatype.RefExpression(types.numberRef)},
		{ID: "center", Type: datatype.RefExpression(types.pointRef)},
		{ID: "bounds", Type: datatype.RefExpression(types.regionRef)},
	}
	execOutputs := signalList("found", "timeout")
	if spec.kind == "gone" {
		inputs = inputs[:5]
		execOutputs = signalList("gone", "timeout")
	}
	if spec.kind == "click" {
		inputs = append(inputs,
			nodecontract.DataInputPort{ID: "button", Type: datatype.RefExpression(types.buttonRef), Required: true, Default: rawDefault(`"left"`)},
			nodecontract.DataInputPort{ID: "hold-duration", Type: datatype.RefExpression(types.durationRef), Required: true, Default: rawDefault("50")},
		)
		execOutputs = signalList("completed", "timeout")
	}
	requirements := []capability.Requirement{
		{ID: "capture-target", Capability: capture.Ref(), Operations: installed.CaptureOperations(), TargetSlot: "target", Scope: json.RawMessage(`{"operation":"capture"}`)},
		requirement(blobRead, "blob-read", []string{"read-range"}, "blob-store"),
	}
	bindings := []nodecontract.RequirementBindingSpec{{RequirementID: "capture-target", TargetSlotConfigKey: "slot"}}
	if spec.kind == "click" {
		requirements = append(requirements, capability.Requirement{
			ID: "input-target", Capability: input.Ref(), Operations: []string{installed.OperationClick}, TargetSlot: "target", Scope: json.RawMessage(`{"operation":"click"}`),
		})
		bindings = append(bindings, nodecontract.RequirementBindingSpec{RequirementID: "input-target", TargetSlotConfigKey: "slot"})
	}
	return nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
		NodeTypeID: spec.id, ConfigSchemaRoot: schemaID, ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: configSchema}},
		Ports: nodecontract.PortSet{DataInputs: inputs, DataOutputs: outputs, ExecInputs: signalList("in"), ExecOutputs: execOutputs, ErrorOutputs: signalList("failed")},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{nodecontract.EffectID(spec.effectID)}, Determinism: nodecontract.Recorded,
			Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutRequired,
		},
		Instruction: nodecontract.Invoke(), CapabilityRequirements: requirements, RequirementBindings: bindings,
		Errors: automationTemplateErrors(spec.kind == "click"), StatusEvents: []nodecontract.StatusEventSpec{
			{Code: AutomationTemplateWaitingStatus, Category: nodecontract.StatusWaiting},
			{Code: AutomationTemplateMatchedStatus, Category: nodecontract.StatusProgress},
			{Code: AutomationTemplateTimeoutStatus, Category: nodecontract.StatusProgress},
		}, ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: spec.titleKey + ".title", DescriptionKey: spec.titleKey + ".description", Category: "automation",
			Tags: []string{"automation", "template", "vision", spec.kind}, Icon: spec.icon,
			Ports: dataPortHints(spec.titleKey, inputs, outputs, map[string]string{"template": "template-image"}),
		},
	})
}

func automationTemplateInputs(types automationTemplateTypes) []nodecontract.DataInputPort {
	return []nodecontract.DataInputPort{
		{ID: "template", Type: datatype.RefExpression(types.imageRef), Required: true},
		{ID: "region", Type: datatype.RefExpression(types.regionRef), Required: true, Default: rawDefault(`{"x":0,"y":0,"width":1,"height":1,"unit":"ratio"}`)},
		{ID: "threshold", Type: datatype.RefExpression(types.numberRef), Required: true, Default: rawDefault("0.85")},
		{ID: "timeout", Type: datatype.RefExpression(types.durationRef), Required: true, Default: rawDefault("5000")},
		{ID: "poll-interval", Type: datatype.RefExpression(types.durationRef), Required: true, Default: rawDefault("100")},
		{ID: "settle-duration", Type: datatype.RefExpression(types.durationRef), Required: true, Default: rawDefault("200")},
	}
}

func automationTemplateErrors(includeInput bool) []nodecontract.ErrorSpec {
	codes := []string{
		installed.CodeInvalidRequest, installed.CodeIdentityChanged, installed.CodeTargetNotFound, installed.CodeTargetAmbiguous,
		installed.CodeCaptureFailed, installed.CodeUnsupportedHost, installed.CodeContractViolation,
		VisionImageInvalidCode, VisionTemplateInvalidCode, VisionRegionInvalidCode, VisionMatchFailedCode,
	}
	if includeInput {
		codes = append(codes, installed.CodeInputFailed)
	}
	result := make([]nodecontract.ErrorSpec, 0, len(codes))
	for _, code := range codes {
		result = append(result, nodecontract.ErrorSpec{Code: code, Category: "automation", RetryHint: false})
	}
	return result
}
