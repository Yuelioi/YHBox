package noderuntime

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/scriptengine"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func scriptExecute(builtins nodes.Builtins, runtime ScriptExecutor) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		action := compiler.AdapterAction{
			EffectID: nodes.ScriptExecuteEffectID,
			Action:   "script.executed", SummaryCode: "script.execute",
			Counters: map[string]int64{},
			Facts:    map[string]string{"protocol": scriptengine.Protocol},
		}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(
				ctx, invocation, action, scriptengine.CodeRunnerCrashed, runErr,
			))
		}()

		source, ok := invocation.Config["source"].(string)
		if !ok || source == "" || len(source) > scriptengine.MaxSourceBytes {
			return compiler.AdapterResult{}, scriptFailure(scriptengine.CodeSourceInvalid, errors.New("script source is invalid"))
		}
		timeout, err := configInt64(invocation.Config["timeoutMilliseconds"])
		if err != nil || timeout < scriptengine.MinTimeoutMillis || timeout > scriptengine.MaxTimeoutMillis {
			return compiler.AdapterResult{}, scriptFailure(scriptengine.CodeContractViolation, errors.New("script timeout is invalid"))
		}
		input, ok := invocation.Inputs["input"]
		if !ok || len(input.InlineJSON()) == 0 {
			return compiler.AdapterResult{}, scriptFailure(scriptengine.CodeContractViolation, errors.New("script input is missing"))
		}
		if invocation.InvocationID == "" || invocation.Attempt < 1 || invocation.ObservedAt.IsZero() || invocation.ReadEntropy == nil {
			return compiler.AdapterResult{}, scriptFailure(scriptengine.CodeRunnerCrashed, errors.New("script invocation facts are incomplete"))
		}
		digest, err := artifact.Sum("yotta/script-source/v1", []byte(source))
		if err != nil {
			return compiler.AdapterResult{}, scriptFailure(scriptengine.CodeSourceInvalid, err)
		}
		action.Facts["source_digest"] = digest.String()
		action.Counters["source_bytes"] = int64(len(source))
		action.Counters["input_bytes"] = int64(len(input.InlineJSON()))

		seed := make([]byte, 32)
		if err := invocation.ReadEntropy(seed); err != nil {
			return compiler.AdapterResult{}, scriptFailure(scriptengine.CodeRunnerCrashed, fmt.Errorf("read script seed: %w", err))
		}
		request := scriptengine.Request{
			Protocol: scriptengine.Protocol, AttemptID: invocation.InvocationID,
			Source: source, Input: json.RawMessage(append([]byte(nil), input.InlineJSON()...)),
			EpochUnixMillis: invocation.ObservedAt.UnixMilli(), RandomSeed: hex.EncodeToString(seed),
			TimeoutMillis: int(timeout),
		}
		response, err := runtime.Execute(ctx, request)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return compiler.AdapterResult{}, err
			}
			return compiler.AdapterResult{}, scriptFailure(scriptengine.CodeRunnerCrashed, err)
		}
		if err := response.Validate(); err != nil || response.AttemptID != request.AttemptID {
			return compiler.AdapterResult{}, scriptFailure(scriptengine.CodeRunnerProtocolViolation, errors.Join(err, errors.New("script response identity is invalid")))
		}
		if response.Outcome == scriptengine.OutcomeFailed {
			return compiler.AdapterResult{}, scriptFailure(response.Failure.Code, errors.New("isolated script execution failed"))
		}
		resolved, ok := invocation.OutputTypes["result"]
		if !ok {
			return compiler.AdapterResult{}, scriptFailure(scriptengine.CodeContractViolation, errors.New("script result type is unresolved"))
		}
		envelope, err := datatype.SealInlineJSON(builtins.Catalog, resolved, response.Output)
		if err != nil {
			return compiler.AdapterResult{}, scriptFailure(scriptengine.CodeContractViolation, fmt.Errorf("seal script result: %w", err))
		}
		action.Counters["output_bytes"] = int64(len(response.Output))
		return compiler.AdapterResult{
			Outputs: map[string]datatype.ValueEnvelope{"result": envelope}, ExecOutputs: []string{"completed"},
		}, nil
	}
}

func scriptFailure(code string, cause error) error {
	return &compiler.NodeFailure{Code: code, Output: "failed", Cause: cause}
}
