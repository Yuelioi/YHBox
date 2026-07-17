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
	AutomationWindowCapabilityID       = "https://schemas.yotta.dev/capabilities/automation/window/v1"
	AutomationAppLifecycleCapabilityID = "https://schemas.yotta.dev/capabilities/automation/app-lifecycle/v1"
	ActivateWindowNodeID               = "https://schemas.yotta.dev/nodes/automation/activate-window"
	ActivateWindowEffectID             = "https://schemas.yotta.dev/effects/automation/activate-window/v1"
	StopTargetAppNodeID                = "https://schemas.yotta.dev/nodes/automation/stop-target-app"
	StopTargetAppEffectID              = "https://schemas.yotta.dev/effects/automation/stop-target-app/v1"
)

func sealAutomationWindowCapability() (capability.Definition, error) {
	const scopeID = AutomationWindowCapabilityID + "/scope"
	return capability.SealDefinition(capability.DefinitionDraft{
		CapabilityID: AutomationWindowCapabilityID, Operations: []string{installed.OperationActivate}, TargetKinds: []string{installed.TargetKindDesktopWindow, installed.TargetKindAndroidDevice},
		ScopeSchemaRoot: scopeID, ScopeSchemaBundle: []datatype.SchemaResource{{ID: scopeID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"operation":{"const":"activate"}},"required":["operation"],"additionalProperties":false
		}`, scopeID))}},
		Credential: capability.CredentialNone, Risk: capability.RiskDangerous, Consent: capability.ConsentOnce,
		ProviderABI: installed.ProviderABI,
	})
}

func sealAutomationAppLifecycleCapability() (capability.Definition, error) {
	const scopeID = AutomationAppLifecycleCapabilityID + "/scope"
	return capability.SealDefinition(capability.DefinitionDraft{
		CapabilityID: AutomationAppLifecycleCapabilityID, Operations: []string{installed.OperationStopApp}, TargetKinds: []string{installed.TargetKindAndroidDevice},
		ScopeSchemaRoot: scopeID, ScopeSchemaBundle: []datatype.SchemaResource{{ID: scopeID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"operation":{"const":"stop-app"}},"required":["operation"],"additionalProperties":false
		}`, scopeID))}},
		Credential: capability.CredentialNone, Risk: capability.RiskDangerous, Consent: capability.ConsentOnce,
		ProviderABI: installed.ProviderABI,
	})
}

func defineActivateWindowNode(window capability.Definition) (BuiltinDefinition, nodecontract.Contract, error) {
	return defineAutomationWindowNode(window, ActivateWindowNodeID, ActivateWindowEffectID, installed.OperationActivate, "automation.activate-window", "node.automation.activateWindow", "window-maximize", "exact-target/target-activation/v1")
}

func defineStopTargetAppNode(window capability.Definition) (BuiltinDefinition, nodecontract.Contract, error) {
	return defineAutomationWindowNode(window, StopTargetAppNodeID, StopTargetAppEffectID, installed.OperationStopApp, "automation.stop-target-app", "node.automation.stopTargetApp", "player-stop", "exact-target/target-app-stop/v1")
}

func defineAutomationWindowNode(window capability.Definition, nodeID, effectID, operation, entrypoint, titleKey, icon, conformance string) (BuiltinDefinition, nodecontract.Contract, error) {
	schemaID := nodeID + "/config"
	contract, err := nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
		NodeTypeID: nodeID, ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"slot":{"type":"string","minLength":1,"maxLength":128,"pattern":"^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$",
				"x-yotta-title-key":"node.automation.config.slot.title","x-yotta-description-key":"node.automation.config.slot.description"}},
			"required":["slot"],"additionalProperties":false
		}`, schemaID))}},
		Ports: nodecontract.PortSet{
			ExecInputs: signalList("in"), ExecOutputs: signalList("completed"), ErrorOutputs: signalList("failed"),
		},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{nodecontract.EffectID(effectID)}, Determinism: nodecontract.Recorded,
			Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutRequired,
		},
		Instruction: nodecontract.Invoke(),
		CapabilityRequirements: []capability.Requirement{{
			ID: "target", Capability: window.Ref(), Operations: []string{operation}, TargetSlot: "target", Scope: json.RawMessage(fmt.Sprintf(`{"operation":%q}`, operation)),
		}},
		RequirementBindings: []nodecontract.RequirementBindingSpec{{RequirementID: "target", TargetSlotConfigKey: "slot"}},
		Errors: []nodecontract.ErrorSpec{
			{Code: installed.CodeIdentityChanged, Category: "automation", RetryHint: false},
			{Code: installed.CodeTargetNotFound, Category: "automation", RetryHint: true},
			{Code: installed.CodeTargetAmbiguous, Category: "automation", RetryHint: false},
			{Code: installed.CodeWindowFailed, Category: "automation", RetryHint: true},
			{Code: installed.CodeUnsupportedHost, Category: "automation", RetryHint: false},
			{Code: installed.CodeContractViolation, Category: "automation", RetryHint: false},
		},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: titleKey + ".title", DescriptionKey: titleKey + ".description", Category: "automation",
			Tags: []string{"automation", "target", operation}, Icon: icon,
		},
	})
	if err != nil {
		return BuiltinDefinition{}, nodecontract.Contract{}, err
	}
	definition, err := defineBuiltin(contract, entrypoint, "v1", conformance, nil)
	return definition, contract, err
}
