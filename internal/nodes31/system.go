package nodes31

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	ObservabilityMessageTypeID   = "https://schemas.yotta.dev/types/observability/message/v1"
	MaxObservabilityMessageRunes = 16_384

	LogNodeID   = "https://schemas.yotta.dev/nodes/observability/log/v1"
	ThrowNodeID = "https://schemas.yotta.dev/nodes/control/throw/v1"

	LogWriteEffectID = "https://schemas.yotta.dev/effects/observability/log/v1"
	LogWriteFailed   = "observability.log_write_failed"
	LogContractError = "observability.log_contract_violation"
	ControlThrown    = "control.thrown"
)

func sealObservabilityMessageType() (datatype.Definition, error) {
	return sealStructuredType(
		ObservabilityMessageTypeID,
		json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema",
			"type":"string","maxLength":%d
		}`, ObservabilityMessageTypeID+"/schema", MaxObservabilityMessageRunes)),
		datatype.Authoring{
			TitleKey: "type.observability.message.title", DescriptionKey: "type.observability.message.description",
			Color: "#f97316", Icon: "message-circle",
		},
	)
}

func defineSystemNodes(messageRef datatype.TypeRef) ([]BuiltinDefinition, error) {
	messageType := datatype.RefExpression(messageRef)
	defaultMessage := json.RawMessage(`""`)

	logSchemaID := LogNodeID + "/config"
	logContract, err := nodecontract.Seal(nodecontract.Draft{
		NodeTypeID: LogNodeID, ConfigSchemaRoot: logSchemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: logSchemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"level":{"type":"string","enum":["debug","info","warn","error"],"default":"info",
				"x-yotta-title-key":"node.observability.log.config.level.title",
				"x-yotta-description-key":"node.observability.log.config.level.description"}},
			"additionalProperties":false
		}`, logSchemaID))}},
		Ports: nodecontract.PortSet{
			DataInputs: []nodecontract.DataInputPort{{ID: "message", Type: messageType, Required: true}}, DataOutputs: []nodecontract.DataOutputPort{},
			ExecInputs: signalList("in"), ExecOutputs: signalList("completed"), ErrorOutputs: signalList("failed"),
		},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{LogWriteEffectID}, Determinism: nodecontract.Recorded,
			Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutNone,
		},
		Instruction: nodecontract.Invoke(), CapabilityRequirements: []capability.Requirement{},
		Errors: []nodecontract.ErrorSpec{
			{Code: LogWriteFailed, Category: "observability", RetryHint: false},
			{Code: LogContractError, Category: "observability", RetryHint: false},
		},
		StatusEvents:      []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: "node.observability.log.title", DescriptionKey: "node.observability.log.description",
			Category: "io", Tags: []string{"debug", "log", "observability"}, Icon: "message-circle",
		},
	})
	if err != nil {
		return nil, err
	}
	throwSchemaID := ThrowNodeID + "/config"
	throwContract, err := nodecontract.Seal(nodecontract.Draft{
		NodeTypeID: ThrowNodeID, ConfigSchemaRoot: throwSchemaID, ConfigSchemaBundle: emptyConfigSchema(throwSchemaID),
		Ports: nodecontract.PortSet{
			DataInputs:  []nodecontract.DataInputPort{{ID: "message", Type: messageType, Required: true, Default: &defaultMessage}},
			DataOutputs: []nodecontract.DataOutputPort{}, ExecInputs: signalList("in"), ExecOutputs: []nodecontract.SignalPort{}, ErrorOutputs: []nodecontract.SignalPort{},
		},
		Execution: controlExecution(), Instruction: nodecontract.Invoke(), CapabilityRequirements: []capability.Requirement{},
		Errors: []nodecontract.ErrorSpec{{Code: ControlThrown, Category: "control", RetryHint: false}}, StatusEvents: []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: "node.control.throw.title", DescriptionKey: "node.control.throw.description",
			Category: "control", Tags: []string{"control", "error", "terminal", "throw"}, Icon: "alert-triangle",
		},
	})
	if err != nil {
		return nil, err
	}
	logDefinition, err := defineBuiltin(logContract, "observability.log", "v1", "redacted-run-attributed-log/v1", nil)
	if err != nil {
		return nil, err
	}
	throwDefinition, err := defineBuiltin(throwContract, "control.throw", "v1", "stable-terminal-failure/v1", nil)
	if err != nil {
		return nil, err
	}
	return []BuiltinDefinition{logDefinition, throwDefinition}, nil
}
