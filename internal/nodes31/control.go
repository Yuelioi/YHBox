package nodes31

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	DurationMillisecondsTypeID = "https://schemas.yotta.dev/types/time/duration-milliseconds/v1"

	RunStartedNodeID = "https://schemas.yotta.dev/nodes/event/run-started/v1"
	BranchNodeID     = "https://schemas.yotta.dev/nodes/control/branch/v1"
	DelayNodeID      = "https://schemas.yotta.dev/nodes/control/delay/v1"
	EndBranchNodeID  = "https://schemas.yotta.dev/nodes/control/end-branch/v1"

	DelayWaitEffectID          = "https://schemas.yotta.dev/effects/time/delay/v1"
	DelayWaitingStatus         = "control.delay.waiting"
	DelayFailedCode            = "control.delay_failed"
	MaxDelayMilliseconds int64 = 86_400_000
)

func sealDurationMillisecondsType() (datatype.Definition, error) {
	return sealStructuredType(
		DurationMillisecondsTypeID,
		json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema",
			"type":"integer","minimum":0,"maximum":%d
		}`, DurationMillisecondsTypeID+"/schema", MaxDelayMilliseconds)),
		datatype.Authoring{
			TitleKey: "type.time.durationMilliseconds.title", DescriptionKey: "type.time.durationMilliseconds.description",
			Color: "#a78bfa", Icon: "clock-hour-4", Examples: []json.RawMessage{json.RawMessage("1000")},
		},
	)
}

func defineControlNodes(types primitiveTypes, durationRef datatype.TypeRef) ([]BuiltinDefinition, error) {
	booleanType := datatype.RefExpression(types.booleanRef)
	durationType := datatype.RefExpression(durationRef)
	defaultTrue := json.RawMessage("true")
	defaultDelay := json.RawMessage("1000")
	type controlNode struct {
		id, entrypoint, conformance, key, category, icon string
		ports                                            nodecontract.PortSet
		execution                                        nodecontract.ExecutionSpec
		statuses                                         []nodecontract.StatusEventSpec
		errors                                           []nodecontract.ErrorSpec
	}
	nodes := []controlNode{
		{
			id: RunStartedNodeID, entrypoint: "event.run-started", conformance: "exactly-once-run-root/v1",
			key: "node.event.runStarted", category: "event", icon: "player-play",
			ports: signalPorts(nil, []string{"started"}), execution: eventExecution(),
		},
		{
			id: BranchNodeID, entrypoint: "control.branch", conformance: "strict-boolean-exclusive-route/v1",
			key: "node.control.branch", category: "control", icon: "git-branch",
			ports: nodecontract.PortSet{
				DataInputs:  []nodecontract.DataInputPort{{ID: "condition", Type: booleanType, Required: true, Default: &defaultTrue}},
				DataOutputs: []nodecontract.DataOutputPort{}, ExecInputs: signalList("in"),
				ExecOutputs: signalList("true", "false"), ErrorOutputs: []nodecontract.SignalPort{},
			},
			execution: controlExecution(),
		},
		{
			id: DelayNodeID, entrypoint: "control.delay", conformance: "cancellable-host-wait/v1",
			key: "node.control.delay", category: "control", icon: "clock-pause",
			ports: nodecontract.PortSet{
				DataInputs:  []nodecontract.DataInputPort{{ID: "duration-milliseconds", Type: durationType, Required: true, Default: &defaultDelay}},
				DataOutputs: []nodecontract.DataOutputPort{}, ExecInputs: signalList("in"),
				ExecOutputs: signalList("done"), ErrorOutputs: []nodecontract.SignalPort{},
			},
			execution: effectExecution(DelayWaitEffectID),
			statuses:  []nodecontract.StatusEventSpec{{Code: DelayWaitingStatus, Category: nodecontract.StatusWaiting}},
			errors:    []nodecontract.ErrorSpec{{Code: DelayFailedCode, Category: "control", RetryHint: false}},
		},
		{
			id: EndBranchNodeID, entrypoint: "control.end-branch", conformance: "explicit-branch-termination/v1",
			key: "node.control.endBranch", category: "control", icon: "player-stop",
			ports: signalPorts([]string{"in"}, nil), execution: controlExecution(),
		},
	}
	definitions := make([]BuiltinDefinition, 0, len(nodes))
	for _, item := range nodes {
		configID := item.id + "/config"
		contract, err := nodecontract.Seal(nodecontract.Draft{
			NodeTypeID: item.id, ConfigSchemaRoot: configID, ConfigSchemaBundle: emptyConfigSchema(configID),
			Ports: item.ports, Execution: item.execution, CapabilityRequirements: []capability.Requirement{},
			Errors: item.errors, StatusEvents: item.statuses,
			ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
			Authoring: nodecontract.Authoring{
				TitleKey: item.key + ".title", DescriptionKey: item.key + ".description", Category: item.category,
				Tags: []string{item.category, "execution"}, Icon: item.icon,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("seal built-in %s: %w", item.id, err)
		}
		definition, err := defineBuiltin(contract, item.entrypoint, "v1", item.conformance, nil)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, definition)
	}
	return definitions, nil
}

func signalPorts(inputs, outputs []string) nodecontract.PortSet {
	return nodecontract.PortSet{
		DataInputs: []nodecontract.DataInputPort{}, DataOutputs: []nodecontract.DataOutputPort{},
		ExecInputs: signalList(inputs...), ExecOutputs: signalList(outputs...), ErrorOutputs: []nodecontract.SignalPort{},
	}
}

func signalList(ids ...string) []nodecontract.SignalPort {
	result := make([]nodecontract.SignalPort, len(ids))
	for index, id := range ids {
		result[index] = nodecontract.SignalPort{ID: id}
	}
	return result
}

func eventExecution() nodecontract.ExecutionSpec {
	return nodecontract.ExecutionSpec{
		Class: nodecontract.ExecutionEvent, Effects: []nodecontract.EffectID{}, Determinism: nodecontract.Deterministic,
		Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
		Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutNone,
	}
}

func controlExecution() nodecontract.ExecutionSpec {
	return nodecontract.ExecutionSpec{
		Class: nodecontract.ExecutionControl, Effects: []nodecontract.EffectID{}, Determinism: nodecontract.Deterministic,
		Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
		Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutNone,
	}
}

func effectExecution(effect nodecontract.EffectID) nodecontract.ExecutionSpec {
	return nodecontract.ExecutionSpec{
		Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{effect}, Determinism: nodecontract.Recorded,
		Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
		Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutNone,
	}
}
