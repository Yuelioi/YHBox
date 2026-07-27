package compiler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

type Builtin func(context.Context, map[string]json.RawMessage) (map[string]json.RawMessage, error)

type InstalledBuiltin struct {
	Implementation nodecatalog.ImplementationLock
	Run            Builtin
}

type Interpreter struct {
	catalog  nodecatalog.Snapshot
	builtins map[string]InstalledBuiltin
}

// RunResult stores canonical ValueEnvelope artifacts, not untyped JSON.
type RunResult struct {
	NodeOutputs map[string]map[string]json.RawMessage
}

func NewInterpreter(catalog nodecatalog.Snapshot, builtins map[string]InstalledBuiltin) *Interpreter {
	installed := make(map[string]InstalledBuiltin, len(builtins))
	for name, builtin := range builtins {
		installed[name] = builtin
	}
	return &Interpreter{catalog: catalog, builtins: installed}
}

func (i *Interpreter) Run(ctx context.Context, program ProgramSnapshot) (RunResult, error) {
	if !program.Valid() {
		return RunResult{}, errors.New("interpreter requires a valid Program snapshot")
	}
	if !i.catalog.Valid() || program.state.document.Body.CatalogHash != i.catalog.Hash() {
		return RunResult{}, errors.New("interpreter catalog does not match Program v1")
	}
	body := program.state.document.Body
	if len(program.CapabilityPlan().Entries()) != 0 {
		return RunResult{}, errors.New("pure-data preview cannot execute a capability plan")
	}
	var graph *programGraph
	for index := range body.Graphs {
		if body.Graphs[index].ID == body.EntryGraph {
			graph = &body.Graphs[index]
			break
		}
	}
	if graph == nil {
		return RunResult{}, errors.New("program entry graph is missing")
	}
	nodes := make(map[string]programNode, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodes[node.ID] = node
	}
	result := RunResult{NodeOutputs: map[string]map[string]json.RawMessage{}}
	retainedBytes := 0
	for _, nodeID := range graph.DataOrder {
		if err := ctx.Err(); err != nil {
			return RunResult{}, err
		}
		node, ok := nodes[nodeID]
		if !ok {
			return RunResult{}, fmt.Errorf("program order references missing node %q", nodeID)
		}
		if node.Execution.Class != nodecontract.ExecutionPureData {
			return RunResult{}, fmt.Errorf("pure-data preview cannot execute class %q", node.Execution.Class)
		}
		installed, ok := i.builtins[node.Implementation.Entrypoint]
		if !ok || installed.Run == nil {
			return RunResult{}, fmt.Errorf("builtin entrypoint %q is not installed", node.Implementation.Entrypoint)
		}
		if installed.Implementation != node.Implementation {
			return RunResult{}, fmt.Errorf("installed builtin %q does not match the pinned implementation lock", node.Implementation.Entrypoint)
		}
		inputs := make(map[string]json.RawMessage, len(node.Inputs))
		for portID, plan := range node.Inputs {
			var envelopeRaw []byte
			switch plan.Kind {
			case inputLiteral:
				envelopeRaw = plan.Value
			case inputEdge:
				value, exists := result.NodeOutputs[plan.From.NodeID][plan.From.PortID]
				if !exists {
					return RunResult{}, fmt.Errorf("upstream value %s.%s is unavailable", plan.From.NodeID, plan.From.PortID)
				}
				envelopeRaw = value
			default:
				return RunResult{}, fmt.Errorf("unknown input plan %q", plan.Kind)
			}
			envelope, err := datatype.OpenValueEnvelope(i.catalog, envelopeRaw)
			if err != nil {
				return RunResult{}, fmt.Errorf("open input %s.%s: %w", nodeID, portID, err)
			}
			expected, ok := node.InputTypes[portID]
			if !ok || !reflect.DeepEqual(envelope.Type(), expected) {
				return RunResult{}, fmt.Errorf("input %s.%s violates its Program-resolved type", nodeID, portID)
			}
			inputs[portID] = envelope.InlineJSON()
		}
		outputs, err := installed.Run(ctx, inputs)
		if err != nil {
			return RunResult{}, fmt.Errorf("run node %q: %w", nodeID, err)
		}
		ports := node.OutputTypes
		sealedOutputs := make(map[string]json.RawMessage, len(outputs))
		for portID, value := range outputs {
			if err := ctx.Err(); err != nil {
				return RunResult{}, err
			}
			if len(value) == 0 || len(value) > datatype.MaxInlineValueBytes {
				return RunResult{}, fmt.Errorf("builtin output %q exceeds inline value budget", portID)
			}
			resolved, allowed := ports[portID]
			if !allowed {
				return RunResult{}, fmt.Errorf("builtin returned undeclared output %q", portID)
			}
			canonical, err := artifact.Canonicalize(value)
			if err != nil {
				return RunResult{}, err
			}
			envelope, err := datatype.SealInlineJSON(i.catalog, resolved, canonical)
			if err != nil {
				return RunResult{}, err
			}
			retainedBytes += len(envelope.Artifact())
			if retainedBytes > MaxRunRetainedValueBytes {
				return RunResult{}, errors.New("run retained value budget exceeded")
			}
			sealedOutputs[portID] = envelope.Artifact()
		}
		for portID := range ports {
			if _, exists := sealedOutputs[portID]; !exists {
				return RunResult{}, fmt.Errorf("builtin omitted output %q", portID)
			}
		}
		result.NodeOutputs[nodeID] = sealedOutputs
	}
	return result, nil
}
