package scriptengine

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/bits"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja/parser"

	"github.com/yottaapp/yotta/internal/artifact"
)

const guestPrefix = `(function(input) {
"use strict";
`

const guestSuffix = `
})`

const outputValidatorSource = `(function() {
  "use strict";
  const arrayPrototype = Array.prototype;
  const objectPrototype = Object.prototype;
  const isArray = Array.isArray;
  const getPrototypeOf = Object.getPrototypeOf;
  const ownKeys = Reflect.ownKeys;
  const getOwnPropertyDescriptor = Object.getOwnPropertyDescriptor;
  const stringify = JSON.stringify;
  const isFiniteNumber = Number.isFinite;
  const isSame = Object.is;

  return function(root, maxDepth, maxNodes, maxStringLength) {
    let nodes = 0;
    const ancestors = new Set();

    function visit(value, depth) {
      nodes++;
      if (nodes > maxNodes) throw new TypeError("output node budget exceeded");
      if (depth > maxDepth) throw new TypeError("output depth budget exceeded");
      if (value === null) return;

      switch (typeof value) {
      case "boolean":
        return;
      case "number":
        if (!isFiniteNumber(value) || isSame(value, -0)) {
          throw new TypeError("output contains a non-canonical number");
        }
        return;
      case "string":
        if (value.length > maxStringLength) throw new TypeError("output string budget exceeded");
        return;
      case "object":
        break;
      default:
        throw new TypeError("output contains a non-JSON value");
      }

      if (ancestors.has(value)) throw new TypeError("output contains a cycle");
      ancestors.add(value);

      if (isArray(value)) {
        if (getPrototypeOf(value) !== arrayPrototype) throw new TypeError("output contains a non-plain array");
        if (value.length > maxNodes) throw new TypeError("output array budget exceeded");
        const keys = ownKeys(value);
        if (keys.length !== value.length + 1 || keys[keys.length - 1] !== "length") {
          throw new TypeError("output array contains holes or extra properties");
        }
        for (let index = 0; index < value.length; index++) {
          if (keys[index] !== String(index)) throw new TypeError("output array contains holes or extra properties");
          const descriptor = getOwnPropertyDescriptor(value, keys[index]);
          if (!descriptor || !descriptor.enumerable || !("value" in descriptor)) {
            throw new TypeError("output contains an accessor");
          }
          visit(descriptor.value, depth + 1);
        }
      } else {
        const prototype = getPrototypeOf(value);
        if (prototype !== objectPrototype && prototype !== null) {
          throw new TypeError("output contains a non-plain object");
        }
        const keys = ownKeys(value);
        for (const key of keys) {
          if (typeof key !== "string") throw new TypeError("output contains a symbol property");
          if (key.length > maxStringLength) throw new TypeError("output key budget exceeded");
          const descriptor = getOwnPropertyDescriptor(value, key);
          if (!descriptor || !descriptor.enumerable || !("value" in descriptor)) {
            throw new TypeError("output contains an accessor");
          }
          visit(descriptor.value, depth + 1);
        }
      }

      ancestors.delete(value);
    }

    visit(root, 0);
    return stringify(root);
  };
})()`

type deadlineSignal struct{}

func executeGuest(request Request) Response {
	if err := request.Validate(); err != nil {
		return failedResponse(request.AttemptID, CodeRunnerProtocolViolation, "script request is invalid")
	}

	program, err := goja.Compile("script.js", guestPrefix+request.Source+guestSuffix, true)
	if err != nil {
		return failedResponse(request.AttemptID, CodeSourceInvalid, "script source is not valid ECMAScript")
	}

	var input any
	if err := json.Unmarshal(request.Input, &input); err != nil {
		return failedResponse(request.AttemptID, CodeRunnerProtocolViolation, "script input is invalid")
	}

	vm := goja.New()
	vm.SetParserOptions(parser.WithDisableSourceMaps)
	vm.SetMaxCallStackSize(MaxCallStackDepth)
	vm.SetTimeSource(func() time.Time { return time.UnixMilli(request.EpochUnixMillis).UTC() })
	vm.SetRandSource(randomSource(request.RandomSeed))
	if err := installVirtualDate(vm, request.EpochUnixMillis); err != nil {
		return failedResponse(request.AttemptID, CodeRunnerCrashed, "script worker failed to initialize")
	}

	validatorValue, err := vm.RunString(outputValidatorSource)
	if err != nil {
		return failedResponse(request.AttemptID, CodeRunnerCrashed, "script worker failed to initialize")
	}
	validateOutput, ok := goja.AssertFunction(validatorValue)
	if !ok {
		return failedResponse(request.AttemptID, CodeRunnerCrashed, "script worker failed to initialize")
	}

	timer := time.NewTimer(time.Duration(request.TimeoutMillis) * time.Millisecond)
	done := make(chan struct{})
	defer func() {
		close(done)
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}()
	go func() {
		select {
		case <-timer.C:
			vm.Interrupt(deadlineSignal{})
		case <-done:
		}
	}()

	functionValue, err := vm.RunProgram(program)
	if err != nil {
		return runtimeFailure(request.AttemptID, err, CodeGuestThrown, "script initialization failed")
	}
	guestFunction, ok := goja.AssertFunction(functionValue)
	if !ok {
		return failedResponse(request.AttemptID, CodeRunnerCrashed, "script worker produced an invalid program")
	}
	result, err := guestFunction(goja.Undefined(), vm.ToValue(input))
	if err != nil {
		return runtimeFailure(request.AttemptID, err, CodeGuestThrown, "script threw an exception")
	}

	serialized, err := validateOutput(
		goja.Undefined(),
		result,
		vm.ToValue(MaxJSONDepth),
		vm.ToValue(MaxJSONNodes),
		vm.ToValue(MaxOutputBytes),
	)
	if err != nil {
		return runtimeFailure(request.AttemptID, err, CodeContractViolation, "script output is not an exact JSON value")
	}
	raw := []byte(serialized.String())
	if len(raw) == 0 || len(raw) > MaxOutputBytes {
		return failedResponse(request.AttemptID, CodeContractViolation, "script output exceeds its byte budget")
	}
	if err := artifact.InspectJSONBudget(raw, MaxJSONDepth, MaxJSONNodes, MaxOutputBytes); err != nil {
		return failedResponse(request.AttemptID, CodeContractViolation, "script output exceeds its structural budget")
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || len(canonical) > MaxOutputBytes {
		return failedResponse(request.AttemptID, CodeContractViolation, "script output is not canonical JSON")
	}
	return Response{
		Protocol:  Protocol,
		AttemptID: request.AttemptID,
		Outcome:   OutcomeSucceeded,
		Output:    canonical,
	}
}

func installVirtualDate(vm *goja.Runtime, epochUnixMillis int64) error {
	if err := vm.Set("__yottaEpochUnixMillis", epochUnixMillis); err != nil {
		return err
	}
	_, err := vm.RunString(`(function(epoch) {
  "use strict";
  const virtualDate = Object.freeze({
    now: Object.freeze(function() { return epoch; })
  });
  Object.defineProperty(globalThis, "Date", {
    value: virtualDate,
    writable: false,
    enumerable: false,
    configurable: false
  });
})(__yottaEpochUnixMillis);
delete globalThis.__yottaEpochUnixMillis;`)
	return err
}

func runtimeFailure(attemptID string, err error, fallbackCode, fallbackMessage string) Response {
	var interrupted *goja.InterruptedError
	if errors.As(err, &interrupted) {
		if _, ok := interrupted.Value().(deadlineSignal); ok {
			return failedResponse(attemptID, CodeDeadlineExceeded, "script exceeded its wall-time budget")
		}
		return failedResponse(attemptID, CodeRunnerCrashed, "script worker was interrupted")
	}
	var stackOverflow *goja.StackOverflowError
	if errors.As(err, &stackOverflow) {
		return failedResponse(attemptID, CodeStackExceeded, "script exceeded its call-stack budget")
	}
	return failedResponse(attemptID, fallbackCode, fallbackMessage)
}

func failedResponse(attemptID, code, message string) Response {
	return Response{
		Protocol:  Protocol,
		AttemptID: attemptID,
		Outcome:   OutcomeFailed,
		Failure:   &Failure{Code: code, Message: message},
	}
}

func randomSource(seedHex string) goja.RandSource {
	seed, _ := hex.DecodeString(seedHex)
	state := [4]uint64{
		binary.LittleEndian.Uint64(seed[0:8]),
		binary.LittleEndian.Uint64(seed[8:16]),
		binary.LittleEndian.Uint64(seed[16:24]),
		binary.LittleEndian.Uint64(seed[24:32]),
	}
	if state == [4]uint64{} {
		state[0] = 0x9e3779b97f4a7c15
	}
	return func() float64 {
		result := bits.RotateLeft64(state[1]*5, 7) * 9
		t := state[1] << 17
		state[2] ^= state[0]
		state[3] ^= state[1]
		state[1] ^= state[2]
		state[0] ^= state[3]
		state[2] ^= t
		state[3] = bits.RotateLeft64(state[3], 45)
		return float64(result>>11) * (1.0 / (1 << 53))
	}
}
