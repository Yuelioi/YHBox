package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/httpegress"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	HTTPGetNodeID   = "https://schemas.yotta.dev/nodes/network/http-get"
	HTTPGetEffectID = "https://schemas.yotta.dev/effects/network/http-get/v1"
)

func defineHTTPGetNode(types extendedTypes, integerRef datatype.TypeRef) (BuiltinDefinition, nodecontract.Contract, error) {
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
		ConfiguredTargets: []nodecontract.ConfiguredTargetSpec{{
			ID: "network", TargetSlot: "network", SlotConfigKey: "slot", TargetKinds: []string{httpegress.TargetKind},
		}},
		Errors: httpGetErrors(), StatusEvents: []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: "node.network.httpGet.title", DescriptionKey: "node.network.httpGet.description", Category: "network",
			Tags: []string{"network", "http", "get", "api"}, Icon: "world-www",
		},
	})
	if err != nil {
		return BuiltinDefinition{}, nodecontract.Contract{}, err
	}
	definition, err := defineBuiltin(contract, "network.http-get", "v1", "configured-base-url/http-get/v1", nil)
	return definition, contract, err
}

func rawDefault(value string) *json.RawMessage {
	raw := json.RawMessage(value)
	return &raw
}

func httpGetErrors() []nodecontract.ErrorSpec {
	codes := []string{httpegress.CodeInvalidRequest, httpegress.CodeRequestFailed, httpegress.CodeResponseTooLarge, httpegress.CodeInvalidResponse, httpegress.CodeContractViolation}
	result := make([]nodecontract.ErrorSpec, 0, len(codes))
	for _, code := range codes {
		result = append(result, nodecontract.ErrorSpec{Code: code, Category: "network", RetryHint: false})
	}
	return result
}
