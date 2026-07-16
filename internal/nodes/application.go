package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/appcontrol"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	LaunchApplicationNodeID          = "https://schemas.yotta.dev/nodes/application/launch"
	TerminateApplicationNodeID       = "https://schemas.yotta.dev/nodes/application/terminate"
	ApplicationLifecycleCapabilityID = "https://schemas.yotta.dev/capabilities/application/lifecycle/v1"
	LaunchApplicationEffectID        = "https://schemas.yotta.dev/effects/application/launch/v1"
	TerminateApplicationEffectID     = "https://schemas.yotta.dev/effects/application/terminate/v1"
)

func sealApplicationLifecycleCapability() (capability.Definition, error) {
	const scopeID = ApplicationLifecycleCapabilityID + "/scope"
	return capability.SealDefinition(capability.DefinitionDraft{
		CapabilityID:    ApplicationLifecycleCapabilityID,
		Operations:      []string{appcontrol.OperationLaunch, appcontrol.OperationTerminate},
		TargetKinds:     []string{appcontrol.TargetKind},
		ScopeSchemaRoot: scopeID,
		ScopeSchemaBundle: []datatype.SchemaResource{{ID: scopeID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"operation":{"enum":["launch","terminate"]}},"required":["operation"],"additionalProperties":false
		}`, scopeID))}},
		Credential: capability.CredentialNone, Risk: capability.RiskDangerous,
		Consent: capability.ConsentOnce, ProviderABI: appcontrol.ProviderABI,
	})
}

func defineApplicationNodes(integerRef datatype.TypeRef, lifecycle capability.Definition) ([]BuiltinDefinition, []nodecontract.Contract, error) {
	launch, err := sealApplicationNode(LaunchApplicationNodeID, "application.launch", "node.application.launch", "rocket", appcontrol.OperationLaunch, LaunchApplicationEffectID, integerRef, false, lifecycle)
	if err != nil {
		return nil, nil, err
	}
	terminate, err := sealApplicationNode(TerminateApplicationNodeID, "application.terminate", "node.application.terminate", "player-stop", appcontrol.OperationTerminate, TerminateApplicationEffectID, integerRef, true, lifecycle)
	if err != nil {
		return nil, nil, err
	}
	launchDefinition, err := defineBuiltin(launch, "application.launch", "v1", "exact-installed-executable-no-shell/v1", nil)
	if err != nil {
		return nil, nil, err
	}
	terminateDefinition, err := defineBuiltin(terminate, "application.terminate", "v1", "exact-installed-executable-identity/v1", nil)
	if err != nil {
		return nil, nil, err
	}
	return []BuiltinDefinition{launchDefinition, terminateDefinition}, []nodecontract.Contract{launch, terminate}, nil
}

func sealApplicationNode(nodeID, entrypoint, titleKey, icon, operation, effectID string, integerRef datatype.TypeRef, countOutput bool, lifecycle capability.Definition) (nodecontract.Contract, error) {
	schemaID := nodeID + "/config"
	outputs := []nodecontract.DataOutputPort{}
	if countOutput {
		outputs = append(outputs, nodecontract.DataOutputPort{ID: "terminated-count", Type: datatype.RefExpression(integerRef)})
	}
	return nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
		NodeTypeID: nodeID, ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"slot":{"type":"string","minLength":1,"maxLength":128,"pattern":"^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$",
				"x-yotta-title-key":"node.application.config.slot.title","x-yotta-description-key":"node.application.config.slot.description"}},
			"required":["slot"],"additionalProperties":false
		}`, schemaID))}},
		Ports: nodecontract.PortSet{DataInputs: []nodecontract.DataInputPort{}, DataOutputs: outputs, ExecInputs: signalList("in"), ExecOutputs: signalList("completed"), ErrorOutputs: signalList("failed")},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{nodecontract.EffectID(effectID)}, Determinism: nodecontract.Recorded,
			Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutRequired,
		},
		Instruction:            nodecontract.Invoke(),
		CapabilityRequirements: []capability.Requirement{{ID: "application", Capability: lifecycle.Ref(), Operations: []string{operation}, TargetSlot: "application", Scope: json.RawMessage(fmt.Sprintf(`{"operation":%q}`, operation))}},
		RequirementBindings:    []nodecontract.RequirementBindingSpec{{RequirementID: "application", TargetSlotConfigKey: "slot"}},
		Errors:                 applicationErrors(), StatusEvents: []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring:         nodecontract.Authoring{TitleKey: titleKey + ".title", DescriptionKey: titleKey + ".description", Category: "application", Tags: []string{"application", "process", operation}, Icon: icon},
	})
}

func applicationErrors() []nodecontract.ErrorSpec {
	codes := []string{appcontrol.CodeInvalidRequest, appcontrol.CodeIdentityChanged, appcontrol.CodeLaunchFailed, appcontrol.CodeTerminateFailed, appcontrol.CodeUnsupportedHost, appcontrol.CodeContractViolation}
	result := make([]nodecontract.ErrorSpec, 0, len(codes))
	for _, code := range codes {
		result = append(result, nodecontract.ErrorSpec{Code: code, Category: "application", RetryHint: false})
	}
	return result
}
