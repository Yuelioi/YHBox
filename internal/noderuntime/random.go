package noderuntime

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

const centeredSampleCount = 5

func randomInteger(builtins nodes.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes.RandomSampleEffectID, Action: "random.integer-sampled",
				SummaryCode: "random.integer", Counters: counters,
			}, nodes.RandomEntropyUnavailableCode, runErr))
		}()
		minimum, err := integerInput(invocation, "minimum")
		if err != nil {
			return compiler.AdapterResult{}, randomFailure(nodes.RandomInvalidRangeCode, err)
		}
		maximum, err := integerInput(invocation, "maximum")
		if err != nil || minimum > maximum {
			return compiler.AdapterResult{}, randomFailure(nodes.RandomInvalidRangeCode, errors.Join(err, errors.New("minimum must not exceed maximum")))
		}
		distribution, err := stringInput(invocation, "distribution")
		if err != nil {
			return compiler.AdapterResult{}, randomFailure(nodes.RandomInvalidRangeCode, err)
		}
		span := uint64(maximum) - uint64(minimum) + 1
		var offset uint64
		samples := 1
		if distribution == "centered" {
			samples = centeredSampleCount
			var sum float64
			for range centeredSampleCount {
				unit, err := entropyUnit(invocation)
				if err != nil {
					return compiler.AdapterResult{}, randomFailure(nodes.RandomEntropyUnavailableCode, err)
				}
				sum += unit
			}
			offset = uint64((sum / centeredSampleCount) * float64(span))
			if offset >= span {
				offset = span - 1
			}
		} else if distribution == "uniform" {
			offset, err = entropyIndex(invocation, span)
			if err != nil {
				return compiler.AdapterResult{}, randomFailure(nodes.RandomEntropyUnavailableCode, err)
			}
		} else {
			return compiler.AdapterResult{}, randomFailure(nodes.RandomInvalidRangeCode, errors.New("unknown random distribution"))
		}
		counters["samples"] = int64(samples)
		return sealObservedResult(builtins, invocation, int64(uint64(minimum)+offset))
	}
}

func randomNumber(builtins nodes.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes.RandomSampleEffectID, Action: "random.number-sampled",
				SummaryCode: "random.number", Counters: counters,
			}, nodes.RandomEntropyUnavailableCode, runErr))
		}()
		minimum, err := numberInput(invocation, "minimum")
		if err != nil {
			return compiler.AdapterResult{}, randomFailure(nodes.RandomInvalidRangeCode, err)
		}
		maximum, err := numberInput(invocation, "maximum")
		if err != nil || minimum > maximum || math.IsInf(maximum-minimum, 0) {
			return compiler.AdapterResult{}, randomFailure(nodes.RandomInvalidRangeCode, errors.Join(err, errors.New("range must be ordered and representable")))
		}
		distribution, err := stringInput(invocation, "distribution")
		if err != nil {
			return compiler.AdapterResult{}, randomFailure(nodes.RandomInvalidRangeCode, err)
		}
		unit, samples, err := sampledUnit(invocation, distribution)
		if err != nil {
			code := nodes.RandomEntropyUnavailableCode
			if errors.Is(err, errUnknownDistribution) {
				code = nodes.RandomInvalidRangeCode
			}
			return compiler.AdapterResult{}, randomFailure(code, err)
		}
		counters["samples"] = int64(samples)
		result := minimum
		if minimum != maximum {
			result = minimum + unit*(maximum-minimum)
		}
		if math.IsNaN(result) || math.IsInf(result, 0) {
			return compiler.AdapterResult{}, randomFailure(nodes.RandomInvalidRangeCode, errors.New("sample is not a finite number"))
		}
		return sealObservedResult(builtins, invocation, result)
	}
}

func randomBoolean(builtins nodes.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes.RandomSampleEffectID, Action: "random.boolean-sampled",
				SummaryCode: "random.boolean", Counters: counters,
			}, nodes.RandomEntropyUnavailableCode, runErr))
		}()
		probability, err := numberInput(invocation, "probability")
		if err != nil || probability < 0 || probability > 1 {
			return compiler.AdapterResult{}, randomFailure(nodes.RandomInvalidProbabilityCode, errors.Join(err, errors.New("probability must be between zero and one")))
		}
		unit, err := entropyUnit(invocation)
		if err != nil {
			return compiler.AdapterResult{}, randomFailure(nodes.RandomEntropyUnavailableCode, err)
		}
		counters["samples"] = 1
		return sealObservedResult(builtins, invocation, unit < probability)
	}
}

func randomChoice(builtins nodes.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes.RandomSampleEffectID, Action: "random.choice-sampled",
				SummaryCode: "random.choice", Counters: counters,
			}, nodes.RandomEntropyUnavailableCode, runErr))
		}()
		input, ok := invocation.Inputs["list"]
		if !ok {
			return compiler.AdapterResult{}, randomFailure(nodes.RandomEmptyChoiceCode, errors.New("choice list is missing"))
		}
		var items []json.RawMessage
		if err := json.Unmarshal(input.InlineJSON(), &items); err != nil {
			return compiler.AdapterResult{}, randomFailure(nodes.RandomEmptyChoiceCode, fmt.Errorf("decode choice list: %w", err))
		}
		if len(items) == 0 {
			return compiler.AdapterResult{}, randomFailure(nodes.RandomEmptyChoiceCode, errors.New("choice list is empty"))
		}
		index, err := entropyIndex(invocation, uint64(len(items)))
		if err != nil {
			return compiler.AdapterResult{}, randomFailure(nodes.RandomEntropyUnavailableCode, err)
		}
		counters["samples"] = 1
		return sealObservedRawResult(builtins, invocation, items[index])
	}
}

func observeTime(builtins nodes.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes.TimeObserveEffectID, Action: "time.observed",
				SummaryCode: "time.observe", Counters: map[string]int64{"observations": 1},
			}, nodes.TimeObservationFailedCode, runErr))
		}()
		if invocation.ObservedAt.IsZero() {
			return compiler.AdapterResult{}, &compiler.NodeFailure{Code: nodes.TimeObservationFailedCode, Cause: errors.New("invocation observation time is missing")}
		}
		result, err := sealObservedResult(builtins, invocation, invocation.ObservedAt.UnixMilli())
		if err != nil {
			return compiler.AdapterResult{}, &compiler.NodeFailure{Code: nodes.TimeObservationFailedCode, Cause: err}
		}
		return result, nil
	}
}

var errUnknownDistribution = errors.New("unknown random distribution")

func sampledUnit(invocation compiler.Invocation, distribution string) (float64, int, error) {
	switch distribution {
	case "uniform":
		unit, err := entropyUnit(invocation)
		return unit, 1, err
	case "centered":
		var sum float64
		for range centeredSampleCount {
			unit, err := entropyUnit(invocation)
			if err != nil {
				return 0, 0, err
			}
			sum += unit
		}
		return sum / centeredSampleCount, centeredSampleCount, nil
	default:
		return 0, 0, errUnknownDistribution
	}
}

func entropyUnit(invocation compiler.Invocation) (float64, error) {
	value, err := entropyUint64(invocation)
	if err != nil {
		return 0, err
	}
	return float64(value>>11) * (1.0 / (1 << 53)), nil
}

func entropyIndex(invocation compiler.Invocation, limit uint64) (uint64, error) {
	if limit == 0 {
		return 0, errors.New("random index limit is zero")
	}
	threshold := -limit % limit
	for range 128 {
		value, err := entropyUint64(invocation)
		if err != nil {
			return 0, err
		}
		if value >= threshold {
			return value % limit, nil
		}
	}
	return 0, errors.New("entropy source did not yield an unbiased sample")
}

func entropyUint64(invocation compiler.Invocation) (uint64, error) {
	if invocation.ReadEntropy == nil {
		return 0, errors.New("entropy source is missing")
	}
	var raw [8]byte
	if err := invocation.ReadEntropy(raw[:]); err != nil {
		return 0, fmt.Errorf("read entropy: %w", err)
	}
	return binary.BigEndian.Uint64(raw[:]), nil
}

func integerInput(invocation compiler.Invocation, id string) (int64, error) {
	input, ok := invocation.Inputs[id]
	if !ok {
		return 0, fmt.Errorf("integer input %q is missing", id)
	}
	var number json.Number
	if err := json.Unmarshal(input.InlineJSON(), &number); err != nil {
		return 0, fmt.Errorf("decode integer input %q: %w", id, err)
	}
	value, err := number.Int64()
	if err != nil {
		return 0, fmt.Errorf("decode integer input %q: %w", id, err)
	}
	return value, nil
}

func numberInput(invocation compiler.Invocation, id string) (float64, error) {
	input, ok := invocation.Inputs[id]
	if !ok {
		return 0, fmt.Errorf("number input %q is missing", id)
	}
	var value float64
	if err := json.Unmarshal(input.InlineJSON(), &value); err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("decode finite number input %q: %w", id, err)
	}
	return value, nil
}

func stringInput(invocation compiler.Invocation, id string) (string, error) {
	input, ok := invocation.Inputs[id]
	if !ok {
		return "", fmt.Errorf("string input %q is missing", id)
	}
	var value string
	if err := json.Unmarshal(input.InlineJSON(), &value); err != nil {
		return "", fmt.Errorf("decode string input %q: %w", id, err)
	}
	return value, nil
}

func sealObservedResult(builtins nodes.Builtins, invocation compiler.Invocation, value any) (compiler.AdapterResult, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return compiler.AdapterResult{}, err
	}
	return sealObservedRawResult(builtins, invocation, raw)
}

func sealObservedRawResult(builtins nodes.Builtins, invocation compiler.Invocation, raw json.RawMessage) (compiler.AdapterResult, error) {
	resolved, ok := invocation.OutputTypes["result"]
	if !ok {
		return compiler.AdapterResult{}, errors.New("recorded output type is unresolved")
	}
	envelope, err := datatype.SealInlineJSON(builtins.Catalog, resolved, raw)
	if err != nil {
		return compiler.AdapterResult{}, fmt.Errorf("seal recorded output: %w", err)
	}
	return compiler.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"result": envelope}}, nil
}

func randomFailure(code string, cause error) error {
	return &compiler.NodeFailure{Code: code, Cause: cause}
}
