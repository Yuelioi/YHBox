package noderuntime

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
)

const centeredSampleCount = 5

func randomInteger(builtins nodes.Builtins) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: nodes.RandomSampleEffectID, Action: "random.integer-sampled",
				SummaryCode: "random.integer", Counters: counters,
			}, nodes.RandomEntropyUnavailableCode, runErr))
		}()
		minimum, err := integerInput(invocation, "minimum")
		if err != nil {
			return nodeadapter.AdapterResult{}, randomFailure(nodes.RandomInvalidRangeCode, err)
		}
		maximum, err := integerInput(invocation, "maximum")
		if err != nil || minimum > maximum {
			return nodeadapter.AdapterResult{}, randomFailure(nodes.RandomInvalidRangeCode, errors.Join(err, errors.New("minimum must not exceed maximum")))
		}
		distribution, err := stringInput(invocation, "distribution")
		if err != nil {
			return nodeadapter.AdapterResult{}, randomFailure(nodes.RandomInvalidRangeCode, err)
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
					return nodeadapter.AdapterResult{}, randomFailure(nodes.RandomEntropyUnavailableCode, err)
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
				return nodeadapter.AdapterResult{}, randomFailure(nodes.RandomEntropyUnavailableCode, err)
			}
		} else {
			return nodeadapter.AdapterResult{}, randomFailure(nodes.RandomInvalidRangeCode, errors.New("unknown random distribution"))
		}
		counters["samples"] = int64(samples)
		return sealObservedResult(builtins, invocation, int64(uint64(minimum)+offset))
	}
}

func randomNumber(builtins nodes.Builtins) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: nodes.RandomSampleEffectID, Action: "random.number-sampled",
				SummaryCode: "random.number", Counters: counters,
			}, nodes.RandomEntropyUnavailableCode, runErr))
		}()
		minimum, err := numberInput(invocation, "minimum")
		if err != nil {
			return nodeadapter.AdapterResult{}, randomFailure(nodes.RandomInvalidRangeCode, err)
		}
		maximum, err := numberInput(invocation, "maximum")
		if err != nil || minimum > maximum || math.IsInf(maximum-minimum, 0) {
			return nodeadapter.AdapterResult{}, randomFailure(nodes.RandomInvalidRangeCode, errors.Join(err, errors.New("range must be ordered and representable")))
		}
		distribution, err := stringInput(invocation, "distribution")
		if err != nil {
			return nodeadapter.AdapterResult{}, randomFailure(nodes.RandomInvalidRangeCode, err)
		}
		unit, samples, err := sampledUnit(invocation, distribution)
		if err != nil {
			code := nodes.RandomEntropyUnavailableCode
			if errors.Is(err, errUnknownDistribution) {
				code = nodes.RandomInvalidRangeCode
			}
			return nodeadapter.AdapterResult{}, randomFailure(code, err)
		}
		counters["samples"] = int64(samples)
		result := minimum
		if minimum != maximum {
			result = minimum + unit*(maximum-minimum)
		}
		if math.IsNaN(result) || math.IsInf(result, 0) {
			return nodeadapter.AdapterResult{}, randomFailure(nodes.RandomInvalidRangeCode, errors.New("sample is not a finite number"))
		}
		return sealObservedResult(builtins, invocation, result)
	}
}

func randomBoolean(builtins nodes.Builtins) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: nodes.RandomSampleEffectID, Action: "random.boolean-sampled",
				SummaryCode: "random.boolean", Counters: counters,
			}, nodes.RandomEntropyUnavailableCode, runErr))
		}()
		probability, err := numberInput(invocation, "probability")
		if err != nil || probability < 0 || probability > 1 {
			return nodeadapter.AdapterResult{}, randomFailure(nodes.RandomInvalidProbabilityCode, errors.Join(err, errors.New("probability must be between zero and one")))
		}
		unit, err := entropyUnit(invocation)
		if err != nil {
			return nodeadapter.AdapterResult{}, randomFailure(nodes.RandomEntropyUnavailableCode, err)
		}
		counters["samples"] = 1
		return sealObservedResult(builtins, invocation, unit < probability)
	}
}

func randomChoice(builtins nodes.Builtins) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: nodes.RandomSampleEffectID, Action: "random.choice-sampled",
				SummaryCode: "random.choice", Counters: counters,
			}, nodes.RandomEntropyUnavailableCode, runErr))
		}()
		input, ok := invocation.Inputs["list"]
		if !ok {
			return nodeadapter.AdapterResult{}, randomFailure(nodes.RandomEmptyChoiceCode, errors.New("choice list is missing"))
		}
		var items []json.RawMessage
		if err := json.Unmarshal(input.InlineJSON(), &items); err != nil {
			return nodeadapter.AdapterResult{}, randomFailure(nodes.RandomEmptyChoiceCode, fmt.Errorf("decode choice list: %w", err))
		}
		if len(items) == 0 {
			return nodeadapter.AdapterResult{}, randomFailure(nodes.RandomEmptyChoiceCode, errors.New("choice list is empty"))
		}
		index, err := entropyIndex(invocation, uint64(len(items)))
		if err != nil {
			return nodeadapter.AdapterResult{}, randomFailure(nodes.RandomEntropyUnavailableCode, err)
		}
		counters["samples"] = 1
		return sealObservedRawResult(builtins, invocation, items[index])
	}
}

func observeTime(builtins nodes.Builtins) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: nodes.TimeObserveEffectID, Action: "time.observed",
				SummaryCode: "time.observe", Counters: map[string]int64{"observations": 1},
			}, nodes.TimeObservationFailedCode, runErr))
		}()
		if invocation.ObservedAt.IsZero() {
			return nodeadapter.AdapterResult{}, &nodeadapter.NodeFailure{Code: nodes.TimeObservationFailedCode, Cause: errors.New("invocation observation time is missing")}
		}
		result, err := sealObservedResult(builtins, invocation, invocation.ObservedAt.UnixMilli())
		if err != nil {
			return nodeadapter.AdapterResult{}, &nodeadapter.NodeFailure{Code: nodes.TimeObservationFailedCode, Cause: err}
		}
		return result, nil
	}
}

var errUnknownDistribution = errors.New("unknown random distribution")

func sampledUnit(invocation nodeadapter.Invocation, distribution string) (float64, int, error) {
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

func entropyUnit(invocation nodeadapter.Invocation) (float64, error) {
	value, err := entropyUint64(invocation)
	if err != nil {
		return 0, err
	}
	return float64(value>>11) * (1.0 / (1 << 53)), nil
}

func entropyIndex(invocation nodeadapter.Invocation, limit uint64) (uint64, error) {
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

func entropyUint64(invocation nodeadapter.Invocation) (uint64, error) {
	if invocation.ReadEntropy == nil {
		return 0, errors.New("entropy source is missing")
	}
	var raw [8]byte
	if err := invocation.ReadEntropy(raw[:]); err != nil {
		return 0, fmt.Errorf("read entropy: %w", err)
	}
	return binary.BigEndian.Uint64(raw[:]), nil
}

func integerInput(invocation nodeadapter.Invocation, id string) (int64, error) {
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

func numberInput(invocation nodeadapter.Invocation, id string) (float64, error) {
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

func stringInput(invocation nodeadapter.Invocation, id string) (string, error) {
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

func sealObservedResult(builtins nodes.Builtins, invocation nodeadapter.Invocation, value any) (nodeadapter.AdapterResult, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nodeadapter.AdapterResult{}, err
	}
	return sealObservedRawResult(builtins, invocation, raw)
}

func sealObservedRawResult(builtins nodes.Builtins, invocation nodeadapter.Invocation, raw json.RawMessage) (nodeadapter.AdapterResult, error) {
	resolved, ok := invocation.OutputTypes["result"]
	if !ok {
		return nodeadapter.AdapterResult{}, errors.New("recorded output type is unresolved")
	}
	envelope, err := datatype.SealInlineJSON(builtins.Catalog, resolved, raw)
	if err != nil {
		return nodeadapter.AdapterResult{}, fmt.Errorf("seal recorded output: %w", err)
	}
	return nodeadapter.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"result": envelope}}, nil
}

func randomFailure(code string, cause error) error {
	return &nodeadapter.NodeFailure{Code: code, Cause: cause}
}
