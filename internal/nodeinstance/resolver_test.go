package nodeinstance

import (
	"testing"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestSwitchResolverBuildsOnlyConfiguredCasePorts(t *testing.T) {
	machine := nodecontract.MachineContract{
		Ports: nodecontract.PortSet{
			DataInputs:   []nodecontract.DataInputPort{{ID: "value", Type: datatype.VariableExpression("T")}},
			ExecInputs:   []nodecontract.SignalPort{{ID: "in"}},
			ExecOutputs:  []nodecontract.SignalPort{{ID: "default"}},
			ErrorOutputs: []nodecontract.SignalPort{{ID: "failed"}},
		},
	}
	resolver := SwitchResolver()
	machine.InstanceResolver = &resolver
	resolved, err := Resolve(machine, map[string]any{SwitchCaseCountKey: float64(3)})
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Ports.DataInputs) != 4 || len(resolved.Ports.ExecOutputs) != 4 || resolved.Ports.DataInputs[3].ID != "case-3" || resolved.Ports.ExecOutputs[3].ID != "case-3" {
		t.Fatalf("effective ports = %#v / %#v", resolved.Ports.DataInputs, resolved.Ports.ExecOutputs)
	}
}

func TestSwitchResolverKeepsLegacyEightCasesWhenConfigIsAbsent(t *testing.T) {
	count, err := SwitchCaseCount(map[string]any{})
	if err != nil || count != SwitchLegacyCaseCount {
		t.Fatalf("count=%d err=%v", count, err)
	}
}
