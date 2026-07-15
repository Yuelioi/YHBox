package scriptengine

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestGuestIsDeterministicAndHasNoAmbientAuthority(t *testing.T) {
	request := testRequest()
	request.Source = `
return {
  input,
  now: Date.now(),
  random: [Math.random(), Math.random()],
  ambient: {
    require: typeof require,
    process: typeof process,
    fetch: typeof fetch,
    window: typeof window,
    registry: typeof Registry,
    variables: typeof Variables
  }
};`
	first := executeGuest(request)
	second := executeGuest(request)
	if first.Outcome != OutcomeSucceeded || second.Outcome != OutcomeSucceeded {
		t.Fatalf("executeGuest() failures = %#v / %#v", first.Failure, second.Failure)
	}
	if !bytes.Equal(first.Output, second.Output) {
		t.Fatalf("deterministic outputs differ:\n%s\n%s", first.Output, second.Output)
	}
	var output struct {
		Now     int64             `json:"now"`
		Ambient map[string]string `json:"ambient"`
	}
	if err := json.Unmarshal(first.Output, &output); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if output.Now != request.EpochUnixMillis {
		t.Fatalf("Date.now() = %d, want %d", output.Now, request.EpochUnixMillis)
	}
	for name, value := range output.Ambient {
		if value != "undefined" {
			t.Fatalf("ambient %s = %q", name, value)
		}
	}
}

func TestGuestExposesOnlyVirtualUTCIndependentClock(t *testing.T) {
	request := testRequest()
	request.Source = `
let constructor;
try { new Date(); constructor = "available"; } catch (_) { constructor = "unavailable"; }
return {constructor, now: Date.now(), type: typeof Date};`
	response := executeGuest(request)
	if response.Outcome != OutcomeSucceeded {
		t.Fatalf("executeGuest() failure = %#v", response.Failure)
	}
	if got, want := string(response.Output), `{"constructor":"unavailable","now":1700000000123,"type":"object"}`; got != want {
		t.Fatalf("output = %s, want %s", got, want)
	}
}

func TestGuestReturnsCanonicalJSON(t *testing.T) {
	request := testRequest()
	request.Source = `return {z: input.b, a: [true, null, input.a]};`
	response := executeGuest(request)
	if response.Outcome != OutcomeSucceeded {
		t.Fatalf("executeGuest() failure = %#v", response.Failure)
	}
	if got, want := string(response.Output), `{"a":[true,null,1],"z":"x"}`; got != want {
		t.Fatalf("output = %s, want %s", got, want)
	}
	if err := response.Validate(); err != nil {
		t.Fatalf("response Validate() error = %v", err)
	}
}

func TestGuestMapsStableFailureCodes(t *testing.T) {
	tests := []struct {
		name   string
		source string
		code   string
	}{
		{name: "syntax", source: `return {`, code: CodeSourceInvalid},
		{name: "throw", source: `throw new Error("secret detail");`, code: CodeGuestThrown},
		{name: "undefined", source: `return undefined;`, code: CodeContractViolation},
		{name: "non finite", source: `return {value: NaN};`, code: CodeContractViolation},
		{name: "negative zero", source: `return -0;`, code: CodeContractViolation},
		{name: "cycle", source: `const value = {}; value.self = value; return value;`, code: CodeContractViolation},
		{name: "array hole", source: `return [1,,3];`, code: CodeContractViolation},
		{name: "accessor", source: `return {get value() { return 1; }};`, code: CodeContractViolation},
		{name: "non plain", source: `return new Map();`, code: CodeContractViolation},
		{name: "stack", source: `function recurse() { return recurse(); } return recurse();`, code: CodeStackExceeded},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := testRequest()
			request.Source = test.source
			response := executeGuest(request)
			if response.Outcome != OutcomeFailed || response.Failure == nil || response.Failure.Code != test.code {
				t.Fatalf("executeGuest() = %#v, want code %q", response, test.code)
			}
			if bytes.Contains([]byte(response.Failure.Message), []byte("secret detail")) {
				t.Fatal("guest exception detail escaped the worker")
			}
		})
	}
}

func TestGuestInterruptsPureJavaScriptAtDeadline(t *testing.T) {
	request := testRequest()
	request.Source = `for (;;) {}`
	request.TimeoutMillis = 20
	started := time.Now()
	response := executeGuest(request)
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("deadline took %s", elapsed)
	}
	if response.Outcome != OutcomeFailed || response.Failure == nil || response.Failure.Code != CodeDeadlineExceeded {
		t.Fatalf("executeGuest() = %#v", response)
	}
}

func TestServeOneConsumesOneRequestAndProducesOneResponse(t *testing.T) {
	var input bytes.Buffer
	if err := WriteRequest(&input, testRequest()); err != nil {
		t.Fatalf("WriteRequest() error = %v", err)
	}
	var output bytes.Buffer
	if exitCode := ServeOne(&input, &output); exitCode != WorkerExitOK {
		t.Fatalf("ServeOne() exit = %d", exitCode)
	}
	response, err := ReadResponse(bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatalf("ReadResponse() error = %v", err)
	}
	if response.Outcome != OutcomeSucceeded {
		t.Fatalf("response = %#v", response)
	}
}

func TestServeOneReportsProtocolFailureWithoutExecuting(t *testing.T) {
	var output bytes.Buffer
	if exitCode := ServeOne(bytes.NewReader([]byte("invalid")), &output); exitCode != WorkerExitOK {
		t.Fatalf("ServeOne() exit = %d", exitCode)
	}
	response, err := ReadResponse(bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatalf("ReadResponse() error = %v", err)
	}
	if response.Failure == nil || response.Failure.Code != CodeRunnerProtocolViolation || response.AttemptID != "" {
		t.Fatalf("response = %#v", response)
	}
}
