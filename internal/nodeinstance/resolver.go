// Package nodeinstance resolves config-dependent effective Node Contracts.
// Resolvers are pure and content-addressed: they may only inspect canonical
// node config and the frozen machine contract supplied by the Catalog.
package nodeinstance

import (
	"errors"
	"fmt"
	"math"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	SwitchResolverID       = "https://schemas.yotta.dev/resolvers/control/switch/v1"
	SwitchCaseCountKey     = "caseCount"
	SwitchDefaultCaseCount = 3
	SwitchLegacyCaseCount  = 8
	SwitchMinimumCaseCount = 1
	SwitchMaximumCaseCount = 32
)

func SwitchResolver() nodecontract.InstanceResolver {
	digest, err := artifact.Sum("yotta/node-instance-resolver/v1", []byte(SwitchResolverID))
	if err != nil {
		panic(err)
	}
	return nodecontract.InstanceResolver{
		ResolverID: SwitchResolverID, SemanticDigest: digest,
		MaxPorts: SwitchMaximumCaseCount*2 + 4,
	}
}

func SwitchCaseCount(config map[string]any) (int, error) {
	value, exists := config[SwitchCaseCountKey]
	if !exists {
		return SwitchLegacyCaseCount, nil
	}
	var count int
	switch typed := value.(type) {
	case int:
		count = typed
	case int64:
		count = int(typed)
	case float64:
		if math.Trunc(typed) != typed {
			return 0, errors.New("switch case count must be an integer")
		}
		count = int(typed)
	default:
		return 0, errors.New("switch case count must be an integer")
	}
	if count < SwitchMinimumCaseCount || count > SwitchMaximumCaseCount {
		return 0, fmt.Errorf("switch case count must be between %d and %d", SwitchMinimumCaseCount, SwitchMaximumCaseCount)
	}
	return count, nil
}

func Resolve(machine nodecontract.MachineContract, config map[string]any) (nodecontract.MachineContract, error) {
	if machine.InstanceResolver == nil {
		return machine, nil
	}
	expected := SwitchResolver()
	if *machine.InstanceResolver != expected {
		return nodecontract.MachineContract{}, errors.New("instance resolver is not installed or does not match its semantic digest")
	}
	count, err := SwitchCaseCount(config)
	if err != nil {
		return nodecontract.MachineContract{}, err
	}
	result := machine
	result.Ports = clonePorts(machine.Ports)
	if len(result.Ports.DataInputs) != 1 || result.Ports.DataInputs[0].ID != "value" {
		return nodecontract.MachineContract{}, errors.New("switch resolver requires the value input prototype")
	}
	caseType := result.Ports.DataInputs[0].Type
	for index := 1; index <= count; index++ {
		id := fmt.Sprintf("case-%d", index)
		result.Ports.DataInputs = append(result.Ports.DataInputs, nodecontract.DataInputPort{ID: id, Type: caseType})
		result.Ports.ExecOutputs = append(result.Ports.ExecOutputs, nodecontract.SignalPort{ID: id})
	}
	if len(result.Ports.DataInputs)+len(result.Ports.DataOutputs)+len(result.Ports.ExecInputs)+len(result.Ports.ExecOutputs)+len(result.Ports.ErrorOutputs) > expected.MaxPorts {
		return nodecontract.MachineContract{}, errors.New("effective switch ports exceed the resolver budget")
	}
	return result, nil
}

func clonePorts(source nodecontract.PortSet) nodecontract.PortSet {
	return nodecontract.PortSet{
		DataInputs:   append([]nodecontract.DataInputPort(nil), source.DataInputs...),
		DataOutputs:  append([]nodecontract.DataOutputPort(nil), source.DataOutputs...),
		ExecInputs:   append([]nodecontract.SignalPort(nil), source.ExecInputs...),
		ExecOutputs:  append([]nodecontract.SignalPort(nil), source.ExecOutputs...),
		ErrorOutputs: append([]nodecontract.SignalPort(nil), source.ErrorOutputs...),
	}
}
