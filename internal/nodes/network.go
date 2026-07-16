package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/httpegress"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	HTTPGetNodeID       = "https://schemas.yotta.dev/nodes/network/http-get"
	HTTPGetCapabilityID = "https://schemas.yotta.dev/capabilities/network/http-get/v1"
	HTTPGetEffectID     = "https://schemas.yotta.dev/effects/network/http-get/v1"
)

func sealHTTPGetCapability() (capability.Definition, error) {
	const scopeID = HTTPGetCapabilityID + "/scope"
	return capability.SealDefinition(capability.DefinitionDraft{
		CapabilityID: HTTPGetCapabilityID, Operations: []string{httpegress.OperationGet}, TargetKinds: []string{httpegress.TargetKind},
		ScopeSchemaRoot: scopeID, ScopeSchemaBundle: []datatype.SchemaResource{{ID: scopeID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"method":{"const":"GET"}},"required":["method"],"additionalProperties":false
		}`, scopeID))}},
		Credential: capability.CredentialNone, Risk: capability.RiskSensitive, Consent: capability.ConsentOnce,
		ProviderABI: httpegress.ProviderABI,
	})
}

func defineHTTPGetNode(types extendedTypes, integerRef datatype.TypeRef, network capability.Definition) (BuiltinDefinition, nodecontract.Contract, error) {
	const schemaID = HTTPGetNodeID + "/config"
	contract, err := nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
		NodeTypeID: HTTPGetNodeID, ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"slot":{"type":"string","minLength":1,"maxLength":128,"pattern":"^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$",
				"x-yotta-title-key":"node.network.httpGet.config.slot.title","x-yotta-description-key":"node.network.httpGet.config.slot.description"}},
			"required":["slot"],"additionalProperties":false
		}`, schemaID))}},
		Ports: nodecontract.PortSet{
			DataInputs: []nodecontract.DataInputPort{
				{ID: "path", Type: datatype.RefExpression(types.stringRef), Required: true, Default: rawDefault(`"/"`)},
				{ID: "query", Type: datatype.RefExpression(types.jsonRef), Required: true, Default: rawDefault(`{}`)},
			},
			DataOutputs: []nodecontract.DataOutputPort{
				{ID: "status", Type: datatype.RefExpression(integerRef)},
				{ID: "body", Type: datatype.RefExpression(types.stringRef)},
				{ID: "content-type", Type: datatype.RefExpression(types.stringRef)},
			},
			ExecInputs: signalList("in"), ExecOutputs: signalList("completed"), ErrorOutputs: signalList("failed"),
		},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{HTTPGetEffectID}, Determinism: nodecontract.Recorded,
			Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutRequired,
		},
		Instruction: nodecontract.Invoke(),
		CapabilityRequirements: []capability.Requirement{{
			ID: "origin", Capability: network.Ref(), Operations: []string{httpegress.OperationGet}, TargetSlot: "origin",
			Scope: json.RawMessage(`{"method":"GET"}`),
		}},
		RequirementBindings: []nodecontract.RequirementBindingSpec{{RequirementID: "origin", TargetSlotConfigKey: "slot"}},
		Errors:              httpGetErrors(), StatusEvents: []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: "node.network.httpGet.title", DescriptionKey: "node.network.httpGet.description", Category: "network",
			Tags: []string{"network", "http", "get", "api"}, Icon: "world-www",
		},
	})
	if err != nil {
		return BuiltinDefinition{}, nodecontract.Contract{}, err
	}
	definition, err := defineBuiltin(contract, "network.http-get", "v1", "installed-origin-relative-get/no-redirects/v1", nil)
	return definition, contract, err
}

func rawDefault(value string) *json.RawMessage {
	raw := json.RawMessage(value)
	return &raw
}

func httpGetErrors() []nodecontract.ErrorSpec {
	codes := []string{httpegress.CodeInvalidRequest, httpegress.CodeResolutionDenied, httpegress.CodeRequestFailed, httpegress.CodeResponseTooLarge, httpegress.CodeInvalidResponse, httpegress.CodeContractViolation}
	result := make([]nodecontract.ErrorSpec, 0, len(codes))
	for _, code := range codes {
		result = append(result, nodecontract.ErrorSpec{Code: code, Category: "network", RetryHint: false})
	}
	return result
}
