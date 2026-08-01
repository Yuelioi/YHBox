package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	ControlDualColorBarNodeID       = "https://schemas.yotta.dev/nodes/automation/control-dual-color-bar"
	ControlDualColorBarEffectID     = "https://schemas.yotta.dev/effects/automation/control-dual-color-bar/v1"
	ControlDualColorBarNotFoundCode = "automation.dual_color_bar_not_found"
)

func defineControlDualColorBarNode(types extendedTypes, visionTypes visionTypeSet, durationRef, keyCodeRef datatype.TypeRef) (BuiltinDefinition, nodecontract.Contract, error) {
	const schemaID = ControlDualColorBarNodeID + "/config"
	raw := func(value string) *json.RawMessage {
		result := json.RawMessage(value)
		return &result
	}
	inputs := []nodecontract.DataInputPort{
		{ID: "inner-range", Type: datatype.RefExpression(visionTypes.colorRange.TypeRef()), Required: true},
		{ID: "outer-range", Type: datatype.RefExpression(visionTypes.colorRange.TypeRef()), Required: true},
		{ID: "region", Type: datatype.RefExpression(types.regionRef), Required: true, Default: raw(`{"x":0,"y":0,"width":1,"height":1,"unit":"ratio"}`)},
		{ID: "inner-minimum-width", Type: datatype.RefExpression(types.integerRef), Required: true, Default: raw(`2`)},
		{ID: "inner-maximum-width", Type: datatype.RefExpression(types.integerRef), Required: true, Default: raw(`0`)},
		{ID: "outer-minimum-width", Type: datatype.RefExpression(types.integerRef), Required: true, Default: raw(`0`)},
		{ID: "band-height-ratio", Type: datatype.RefExpression(types.numberRef), Required: true, Default: raw(`0.3`)},
		{ID: "band-inner-height-ratio", Type: datatype.RefExpression(types.numberRef), Required: true, Default: raw(`0.85`)},
		{ID: "inner-confidence-weight", Type: datatype.RefExpression(types.numberRef), Required: true, Default: raw(`0.42`)},
		{ID: "outer-confidence-weight", Type: datatype.RefExpression(types.numberRef), Required: true, Default: raw(`0.58`)},
		{ID: "tolerance-ratio", Type: datatype.RefExpression(types.numberRef), Required: true, Default: raw(`0.08`)},
		{ID: "minimum-tolerance", Type: datatype.RefExpression(types.numberRef), Required: true, Default: raw(`2`)},
		{ID: "left-keys", Type: datatype.ListExpression(datatype.RefExpression(keyCodeRef)), Required: true, Default: raw(`["A"]`)},
		{ID: "right-keys", Type: datatype.ListExpression(datatype.RefExpression(keyCodeRef)), Required: true, Default: raw(`["D"]`)},
		{ID: "hold-duration", Type: datatype.RefExpression(durationRef), Required: true, Default: raw(`35`)},
		{ID: "neutral-duration", Type: datatype.RefExpression(durationRef), Required: true, Default: raw(`20`)},
		{ID: "cycle-duration", Type: datatype.RefExpression(durationRef), Required: true, Default: raw(`80`)},
		{ID: "maximum-iterations", Type: datatype.RefExpression(types.integerRef), Required: true, Default: raw(`267`)},
		{ID: "activation-keys", Type: datatype.ListExpression(datatype.RefExpression(keyCodeRef)), Required: true, Default: raw(`[]`)},
		{ID: "activation-hold-duration", Type: datatype.RefExpression(durationRef), Required: true, Default: raw(`60`)},
		{ID: "appearance-poll-duration", Type: datatype.RefExpression(durationRef), Required: true, Default: raw(`20`)},
		{ID: "activation-retry-duration", Type: datatype.RefExpression(durationRef), Required: true, Default: raw(`300`)},
		{ID: "appearance-timeout", Type: datatype.RefExpression(durationRef), Required: true, Default: raw(`0`)},
	}
	outputs := []nodecontract.DataOutputPort{
		{ID: "frames", Type: datatype.RefExpression(types.integerRef)},
		{ID: "left-actions", Type: datatype.RefExpression(types.integerRef)},
		{ID: "right-actions", Type: datatype.RefExpression(types.integerRef)},
		{ID: "neutral-actions", Type: datatype.RefExpression(types.integerRef)},
		{ID: "activation-actions", Type: datatype.RefExpression(types.integerRef)},
	}
	contract, err := nodecontract.Seal(nodecontract.Draft{
		Version: BuiltinNodeVersion, NodeTypeID: ControlDualColorBarNodeID, ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"slot":{"type":"string","minLength":1,"maxLength":128,"pattern":"^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$",
				"x-yotta-title-key":"node.automation.config.slot.title","x-yotta-description-key":"node.automation.config.slot.description"}},
			"required":["slot"],"additionalProperties":false
		}`, schemaID))}},
		Ports: nodecontract.PortSet{DataInputs: inputs, DataOutputs: outputs, ExecInputs: signalList("in"), ExecOutputs: signalList("completed"), ErrorOutputs: signalList("failed")},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{ControlDualColorBarEffectID}, Determinism: nodecontract.Recorded,
			Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutRequired,
		},
		Instruction:       nodecontract.Invoke(),
		ConfiguredTargets: automationTargetSpec("target", installed.TargetKindDesktopWindow),
		Errors: []nodecontract.ErrorSpec{
			{Code: ControlDualColorBarNotFoundCode, Category: "automation", RetryHint: true},
			{Code: VisionColorRangeInvalidCode, Category: "vision", RetryHint: false},
			{Code: VisionRegionInvalidCode, Category: "vision", RetryHint: false},
			{Code: VisionAnalysisFailedCode, Category: "vision", RetryHint: false},
			{Code: installed.CodeInvalidRequest, Category: "automation", RetryHint: false},
			{Code: installed.CodeTargetNotFound, Category: "automation", RetryHint: true},
			{Code: installed.CodeTargetAmbiguous, Category: "automation", RetryHint: false},
			{Code: installed.CodeCaptureFailed, Category: "automation", RetryHint: true},
			{Code: installed.CodeInputFailed, Category: "automation", RetryHint: true},
			{Code: installed.CodeUnsupportedHost, Category: "automation", RetryHint: false},
			{Code: installed.CodeContractViolation, Category: "automation", RetryHint: false},
		},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: "node.automation.controlDualColorBar.title", DescriptionKey: "node.automation.controlDualColorBar.description", Category: "automation",
			Tags: []string{"automation", "vision", "tracking", "dualcolorbartrack", "realtime", "roi"}, Icon: "adjustments-bolt",
			Ports: dataPortHints("node.automation.controlDualColorBar", inputs, outputs, nil),
		},
	})
	if err != nil {
		return BuiltinDefinition{}, nodecontract.Contract{}, err
	}
	definition, err := defineBuiltin(contract, "automation.control-dual-color-bar", "v1", "activated-roi-rgba-dual-color-bar-feedback-loop/v2", nil)
	return definition, contract, err
}
