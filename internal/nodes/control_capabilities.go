package nodes

import (
	"fmt"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	SwitchNodeID         = "https://schemas.yotta.dev/nodes/control/switch"
	StopwatchStartNodeID = "https://schemas.yotta.dev/nodes/time/stopwatch-start"
	StopwatchReadNodeID  = "https://schemas.yotta.dev/nodes/time/stopwatch-read"
	StopwatchStopNodeID  = "https://schemas.yotta.dev/nodes/time/stopwatch-stop"

	StopwatchEffectID       = "https://schemas.yotta.dev/effects/time/stopwatch/v1"
	StopwatchFailedCode     = "time.stopwatch_failed"
	SwitchFailedCode        = "control.switch_failed"
	StopwatchMaximumElapsed = int64(9_007_199_254_740_991)
	SwitchCaseCount         = 8
)

// defineControlCapabilityNodes restores legacy control conveniences as
// explicit typed dataflow. Switch cases are typed optional inputs rather than
// untrusted dynamic port names; Stopwatch passes a durable start instant
// instead of consulting an ambient process-global timer table.
func defineControlCapabilityNodes(types primitiveTypes) ([]BuiltinDefinition, error) {
	integerType := datatype.RefExpression(types.integerRef)
	equatableType := datatype.VariableExpression("T", string(datatype.TraitDurable), string(datatype.TraitEquatable))

	switchInputs := []nodecontract.DataInputPort{{ID: "value", Type: equatableType, Required: true}}
	switchOutputs := make([]nodecontract.SignalPort, 0, SwitchCaseCount+1)
	for index := 1; index <= SwitchCaseCount; index++ {
		id := fmt.Sprintf("case-%d", index)
		switchInputs = append(switchInputs, nodecontract.DataInputPort{ID: id, Type: equatableType})
		switchOutputs = append(switchOutputs, nodecontract.SignalPort{ID: id})
	}
	switchOutputs = append(switchOutputs, nodecontract.SignalPort{ID: "default"})

	type spec struct {
		id, entrypoint, conformance, key, icon string
		ports                                  nodecontract.PortSet
		execution                              nodecontract.ExecutionSpec
		errors                                 []nodecontract.ErrorSpec
	}
	recordedPull := nodecontract.ExecutionSpec{
		Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{StopwatchEffectID}, Determinism: nodecontract.Recorded,
		Evaluation: nodecontract.EvaluationPull, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
		Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutNone,
	}
	recordedPush := recordedPull
	recordedPush.Evaluation = nodecontract.EvaluationPush
	stopwatchError := []nodecontract.ErrorSpec{{Code: StopwatchFailedCode, Category: "time", RetryHint: false}}
	specs := []spec{
		{
			id: SwitchNodeID, entrypoint: "control.switch", conformance: "typed-first-match-eight-cases/v1",
			key: "node.control.switch", icon: "switch-3",
			ports:     nodecontract.PortSet{DataInputs: switchInputs, DataOutputs: []nodecontract.DataOutputPort{}, ExecInputs: signalList("in"), ExecOutputs: switchOutputs, ErrorOutputs: signalList("failed")},
			execution: controlExecution(), errors: []nodecontract.ErrorSpec{{Code: SwitchFailedCode, Category: "control", RetryHint: false}},
		},
		{
			id: StopwatchStartNodeID, entrypoint: "time.stopwatch-start", conformance: "explicit-recorded-start-instant/v1",
			key: "node.time.stopwatchStart", icon: "clock-play",
			ports:     nodecontract.PortSet{DataInputs: []nodecontract.DataInputPort{}, DataOutputs: []nodecontract.DataOutputPort{{ID: "started-at", Type: integerType}}, ExecInputs: signalList("in"), ExecOutputs: signalList("completed"), ErrorOutputs: signalList("failed")},
			execution: recordedPush, errors: stopwatchError,
		},
		{
			id: StopwatchReadNodeID, entrypoint: "time.stopwatch-read", conformance: "explicit-recorded-elapsed-read/v1",
			key: "node.time.stopwatchRead", icon: "clock-search",
			ports:     nodecontract.PortSet{DataInputs: []nodecontract.DataInputPort{{ID: "started-at", Type: integerType, Required: true}}, DataOutputs: []nodecontract.DataOutputPort{{ID: "elapsed", Type: integerType}}, ExecInputs: []nodecontract.SignalPort{}, ExecOutputs: []nodecontract.SignalPort{}, ErrorOutputs: signalList("failed")},
			execution: recordedPull, errors: stopwatchError,
		},
		{
			id: StopwatchStopNodeID, entrypoint: "time.stopwatch-stop", conformance: "explicit-recorded-stop-instant/v1",
			key: "node.time.stopwatchStop", icon: "clock-stop",
			ports:     nodecontract.PortSet{DataInputs: []nodecontract.DataInputPort{{ID: "started-at", Type: integerType, Required: true}}, DataOutputs: []nodecontract.DataOutputPort{{ID: "elapsed", Type: integerType}}, ExecInputs: signalList("in"), ExecOutputs: signalList("completed"), ErrorOutputs: signalList("failed")},
			execution: recordedPush, errors: stopwatchError,
		},
	}

	definitions := make([]BuiltinDefinition, 0, len(specs))
	for _, item := range specs {
		configID := item.id + "/config"
		contract, err := nodecontract.Seal(nodecontract.Draft{
			Version: BuiltinNodeVersion, NodeTypeID: item.id, ConfigSchemaRoot: configID, ConfigSchemaBundle: emptyConfigSchema(configID),
			Ports: item.ports, Execution: item.execution, Instruction: nodecontract.Invoke(), CapabilityRequirements: []capability.Requirement{},
			Errors: item.errors, StatusEvents: []nodecontract.StatusEventSpec{},
			ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
			Authoring: nodecontract.Authoring{
				TitleKey: item.key + ".title", DescriptionKey: item.key + ".description", Category: chooseControlCategory(item.id),
				Tags: []string{"control", "typed", "time"}, Icon: item.icon,
				Ports: dataPortHints(item.key, item.ports.DataInputs, item.ports.DataOutputs, nil),
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

func chooseControlCategory(nodeID string) string {
	if nodeID == SwitchNodeID {
		return "control"
	}
	return "time"
}
