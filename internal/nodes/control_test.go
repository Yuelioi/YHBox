package nodes

import (
	"testing"

	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestControlAndEventNodesHaveExplicitExecutionSemantics(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	if len(builtins.Types) != 22 || len(builtins.Definitions()) != 145 {
		t.Fatalf("types=%d nodes=%d", len(builtins.Types), len(builtins.Definitions()))
	}
	runStarted, _ := builtins.Definition(RunStartedNodeID)
	branch, _ := builtins.Definition(BranchNodeID)
	delay, _ := builtins.Definition(DelayNodeID)
	end, _ := builtins.Definition(EndBranchNodeID)
	repeat, _ := builtins.Definition(RepeatNodeID)
	forEach, _ := builtins.Definition(ForEachNodeID)
	retry, _ := builtins.Definition(RetryNodeID)
	if runStarted.Contract.Machine().Execution.Class != nodecontract.ExecutionEvent ||
		len(runStarted.Contract.Machine().Ports.ExecInputs) != 0 ||
		!signalIDsEqual(runStarted.Contract.Machine().Ports.ExecOutputs, []string{"started"}) {
		t.Fatalf("run started = %#v", runStarted.Contract.Machine())
	}
	if branch.Contract.Machine().Execution.Class != nodecontract.ExecutionControl ||
		!signalIDsEqual(branch.Contract.Machine().Ports.ExecOutputs, []string{"true", "false"}) ||
		len(branch.Contract.Machine().Execution.Effects) != 0 {
		t.Fatalf("branch = %#v", branch.Contract.Machine())
	}
	if delay.Contract.Machine().Execution.Class != nodecontract.ExecutionEffect ||
		delay.Contract.Machine().Execution.Determinism != nodecontract.Recorded ||
		len(delay.Contract.Machine().Execution.Effects) != 1 ||
		delay.Contract.Machine().Execution.Effects[0] != DelayWaitEffectID ||
		!signalIDsEqual(delay.Contract.Machine().Ports.ExecOutputs, []string{"done"}) {
		t.Fatalf("delay = %#v", delay.Contract.Machine())
	}
	if end.Contract.Machine().Execution.Class != nodecontract.ExecutionControl || len(end.Contract.Machine().Ports.ExecOutputs) != 0 {
		t.Fatalf("end branch = %#v", end.Contract.Machine())
	}
	if runStarted.Contract.Machine().Instruction.Kind != nodecontract.InstructionRunRoot ||
		repeat.Contract.Machine().Instruction.Kind != nodecontract.InstructionCountedLoop ||
		forEach.Contract.Machine().Instruction.Kind != nodecontract.InstructionForEach ||
		retry.Contract.Machine().Instruction.Kind != nodecontract.InstructionRetry {
		t.Fatalf("lowered instructions = %#v / %#v / %#v / %#v", runStarted.Contract.Machine().Instruction, repeat.Contract.Machine().Instruction, forEach.Contract.Machine().Instruction, retry.Contract.Machine().Instruction)
	}
	if runStarted.Contract.Machine().ImplementationABI[0].Kind != nodecontract.ABIHostInstruction ||
		repeat.Contract.Machine().ImplementationABI[0].Kind != nodecontract.ABIHostInstruction ||
		branch.Contract.Machine().ImplementationABI[0].Kind != nodecontract.ABIBuiltin {
		t.Fatalf("instruction ABIs = %#v / %#v / %#v", runStarted.Contract.Machine().ImplementationABI, repeat.Contract.Machine().ImplementationABI, branch.Contract.Machine().ImplementationABI)
	}
	projection, err := nodeauthoring.Project(nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: GeneratorVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	duration, ok := projection.Type(DurationMillisecondsTypeID)
	if !ok || duration.Control != nodeauthoring.ControlInteger || string(duration.Constraints.Minimum) != "0" ||
		string(duration.Constraints.Maximum) != "86400000" {
		t.Fatalf("duration authoring = %#v", duration)
	}
	retryProjection, ok := projection.Node(RetryNodeID)
	if !ok {
		t.Fatal("retry authoring projection is missing")
	}
	foundRetryInput := false
	for _, signal := range retryProjection.Signals {
		if signal.ID == "retry" && signal.Direction == "input" && signal.Channel == "error" {
			foundRetryInput = true
		}
	}
	if !foundRetryInput {
		t.Fatalf("retry signal projection = %#v", retryProjection.Signals)
	}
	repeatProjection, _ := projection.Node(RepeatNodeID)
	delayProjection, _ := projection.Node(DelayNodeID)
	colorProjection, _ := projection.Node(FindColorBlobsNodeID)
	if !containsString(repeatProjection.Tags, "eventtick") || !containsString(delayProjection.Tags, "polling") ||
		!containsString(colorProjection.Tags, "dualcolorbartrack") || !containsString(colorProjection.Tags, "roicolorscan") {
		t.Fatalf("replacement discovery tags = repeat:%v delay:%v color:%v", repeatProjection.Tags, delayProjection.Tags, colorProjection.Tags)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func signalIDsEqual(ports []nodecontract.SignalPort, expected []string) bool {
	if len(ports) != len(expected) {
		return false
	}
	for index := range ports {
		if ports[index].ID != expected[index] {
			return false
		}
	}
	return true
}
