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
	WaitStableNodeID = "https://schemas.yotta.dev/nodes/automation/wait-stable"
	WaitChangeNodeID = "https://schemas.yotta.dev/nodes/automation/wait-change"

	WaitStableEffectID    = "https://schemas.yotta.dev/effects/automation/wait-stable/v1"
	WaitChangeEffectID    = "https://schemas.yotta.dev/effects/automation/wait-change/v1"
	ObservationFailedCode = "automation.observation_failed"
)

func defineAutomationObservationNodes(types automationTemplateTypes, capture capability.Definition) ([]BuiltinDefinition, error) {
	numberType := datatype.RefExpression(types.numberRef)
	integerType := datatype.RefExpression(types.durationRef)
	gridType := datatype.RefExpression(types.integerRef)
	regionType := datatype.RefExpression(types.regionRef)
	defaultRegion := rawDefault(`{"x":0,"y":0,"width":1,"height":1,"unit":"ratio"}`)
	defaultThreshold := rawDefault("0.02")
	defaultTimeout := rawDefault("5000")
	defaultPoll := rawDefault("100")
	defaultStable := rawDefault("500")
	defaultGrid := rawDefault("32")
	defaultCellDelta := rawDefault("12")

	type spec struct {
		id, entrypoint, key, icon, effect, success string
		stable                                     bool
	}
	specs := []spec{
		{id: WaitStableNodeID, entrypoint: "automation.wait-stable", key: "node.automation.waitStable", icon: "freeze-row", effect: WaitStableEffectID, success: "stable", stable: true},
		{id: WaitChangeNodeID, entrypoint: "automation.wait-change", key: "node.automation.waitChange", icon: "activity", effect: WaitChangeEffectID, success: "changed"},
	}
	definitions := make([]BuiltinDefinition, 0, len(specs))
	for _, item := range specs {
		inputs := []nodecontract.DataInputPort{
			{ID: "region", Type: regionType, Required: true, Default: defaultRegion},
			{ID: "threshold", Type: numberType, Required: true, Default: defaultThreshold},
			{ID: "timeout", Type: integerType, Required: true, Default: defaultTimeout},
			{ID: "poll-interval", Type: integerType, Required: true, Default: defaultPoll},
			{ID: "grid-size", Type: gridType, Required: true, Default: defaultGrid},
			{ID: "cell-delta", Type: gridType, Required: true, Default: defaultCellDelta},
		}
		if item.stable {
			inputs = append(inputs, nodecontract.DataInputPort{ID: "stable-duration", Type: integerType, Required: true, Default: defaultStable})
		}
		outputs := []nodecontract.DataOutputPort{{ID: "changed-ratio", Type: numberType}, {ID: "mean-difference", Type: numberType}}
		schemaID := item.id + "/config"
		contract, err := nodecontract.Seal(nodecontract.Draft{
			Version: BuiltinNodeVersion, NodeTypeID: item.id, ConfigSchemaRoot: schemaID,
			ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
				"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
				"properties":{"slot":{"type":"string","minLength":1,"maxLength":128,"pattern":"^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$",
				"x-yotta-title-key":"node.automation.config.slot.title","x-yotta-description-key":"node.automation.config.slot.description"}},
				"required":["slot"],"additionalProperties":false
			}`, schemaID))}},
			Ports: nodecontract.PortSet{
				DataInputs:  inputs,
				DataOutputs: outputs,
				ExecInputs:  signalList("in"), ExecOutputs: signalList(item.success, "timeout"), ErrorOutputs: signalList("failed"),
			},
			Execution: nodecontract.ExecutionSpec{
				Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{nodecontract.EffectID(item.effect)}, Determinism: nodecontract.Recorded,
				Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
				Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutRequired,
			},
			Instruction: nodecontract.Invoke(),
			CapabilityRequirements: []capability.Requirement{{
				ID: "capture-target", Capability: capture.Ref(), Operations: installed.CaptureOperations(), TargetSlot: "target", Scope: json.RawMessage(`{"operation":"capture"}`),
			}},
			RequirementBindings: []nodecontract.RequirementBindingSpec{{RequirementID: "capture-target", TargetSlotConfigKey: "slot"}},
			Errors:              automationObservationErrors(), StatusEvents: []nodecontract.StatusEventSpec{},
			ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
			Authoring: nodecontract.Authoring{
				TitleKey: item.key + ".title", DescriptionKey: item.key + ".description", Category: "automation",
				Tags: []string{"automation", "capture", "observation", "wait"}, Icon: item.icon,
				Ports: dataPortHints(item.key, inputs, outputs, nil),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("seal built-in %s: %w", item.id, err)
		}
		definition, err := defineBuiltin(contract, item.entrypoint, "v1", "exact-target/cancellable-frame-observation/v1", nil)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, definition)
	}
	return definitions, nil
}

func automationObservationErrors() []nodecontract.ErrorSpec {
	codes := []string{
		installed.CodeInvalidRequest, installed.CodeIdentityChanged, installed.CodeTargetNotFound, installed.CodeTargetAmbiguous,
		installed.CodeCaptureFailed, installed.CodeUnsupportedHost, installed.CodeContractViolation,
		VisionImageInvalidCode, VisionRegionInvalidCode, ObservationFailedCode,
	}
	result := make([]nodecontract.ErrorSpec, 0, len(codes))
	for _, code := range codes {
		result = append(result, nodecontract.ErrorSpec{Code: code, Category: "automation", RetryHint: false})
	}
	return result
}
