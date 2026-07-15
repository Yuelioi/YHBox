// Package scriptengine defines the sealed protocol used by Yotta's isolated
// JavaScript worker. The guest evaluator is intentionally not exported: the
// desktop process may execute scripts only through a platform sandbox launcher.
package scriptengine

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	Protocol = "yotta.script.worker/3.1"

	WorkerArgument = "--yotta-script-worker-3.1"

	MaxFrameBytes       = 2 << 20
	MaxSourceBytes      = 256 << 10
	MaxInputBytes       = 1 << 20
	MaxOutputBytes      = 1 << 20
	MaxFailureTextBytes = 2 << 10
	MaxJSONDepth        = 64
	MaxJSONNodes        = 65_536
	MaxCallStackDepth   = 256
	MinTimeoutMillis    = 1
	MaxTimeoutMillis    = 30_000
	MinEpochUnixMillis  = -8_640_000_000_000_000
	MaxEpochUnixMillis  = 8_640_000_000_000_000
)

const (
	OutcomeSucceeded = "succeeded"
	OutcomeFailed    = "failed"
)

const (
	CodeSourceInvalid           = "script.source_invalid"
	CodeGuestThrown             = "script.guest_thrown"
	CodeDeadlineExceeded        = "script.deadline_exceeded"
	CodeStackExceeded           = "script.stack_exceeded"
	CodeContractViolation       = "script.contract_violation"
	CodeRunnerProtocolViolation = "script.runner_protocol_violation"
	CodeRunnerCrashed           = "script.runner_crashed"
	CodeIsolationUnavailable    = "script.isolation_unavailable"
)

var attemptIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// Request is the complete authority-free input for one script attempt.
// RandomSeed is 32 lowercase-hex bytes so its JSON representation is stable
// and does not depend on unsafe integer handling.
type Request struct {
	Protocol        string          `json:"protocol"`
	AttemptID       string          `json:"attemptId"`
	Source          string          `json:"source"`
	Input           json.RawMessage `json:"input"`
	EpochUnixMillis int64           `json:"epochUnixMillis"`
	RandomSeed      string          `json:"randomSeed"`
	TimeoutMillis   int             `json:"timeoutMillis"`
}

type Failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Response is the only value a worker may return. A failed response never
// carries guest output; a successful response never carries a Failure.
type Response struct {
	Protocol  string          `json:"protocol"`
	AttemptID string          `json:"attemptId"`
	Outcome   string          `json:"outcome"`
	Output    json.RawMessage `json:"output,omitempty"`
	Failure   *Failure        `json:"failure,omitempty"`
}

func (request Request) Validate() error {
	if request.Protocol != Protocol {
		return fmt.Errorf("unsupported script worker protocol %q", request.Protocol)
	}
	if !attemptIDPattern.MatchString(request.AttemptID) {
		return errors.New("attemptId is invalid")
	}
	if request.Source == "" || len(request.Source) > MaxSourceBytes || !utf8.ValidString(request.Source) {
		return errors.New("source must be non-empty UTF-8 within the source budget")
	}
	if request.TimeoutMillis < MinTimeoutMillis || request.TimeoutMillis > MaxTimeoutMillis {
		return fmt.Errorf("timeoutMillis must be within %d..%d", MinTimeoutMillis, MaxTimeoutMillis)
	}
	if request.EpochUnixMillis < MinEpochUnixMillis || request.EpochUnixMillis > MaxEpochUnixMillis {
		return errors.New("epochUnixMillis is outside the ECMAScript Date range")
	}
	seed, err := hex.DecodeString(request.RandomSeed)
	if err != nil || len(seed) != 32 || hex.EncodeToString(seed) != request.RandomSeed {
		return errors.New("randomSeed must be exactly 32 lowercase-hex bytes")
	}
	if err := validateCanonicalJSON(request.Input, MaxInputBytes); err != nil {
		return fmt.Errorf("input: %w", err)
	}
	return nil
}

func (response Response) Validate() error {
	if response.Protocol != Protocol {
		return fmt.Errorf("unsupported script worker protocol %q", response.Protocol)
	}
	if response.AttemptID != "" && !attemptIDPattern.MatchString(response.AttemptID) {
		return errors.New("attemptId is invalid")
	}
	switch response.Outcome {
	case OutcomeSucceeded:
		if response.AttemptID == "" || response.Failure != nil {
			return errors.New("successful response has invalid attempt or failure fields")
		}
		if err := validateCanonicalJSON(response.Output, MaxOutputBytes); err != nil {
			return fmt.Errorf("output: %w", err)
		}
	case OutcomeFailed:
		if len(response.Output) != 0 || response.Failure == nil {
			return errors.New("failed response must contain only a failure")
		}
		if !validFailureCode(response.Failure.Code) {
			return fmt.Errorf("unknown failure code %q", response.Failure.Code)
		}
		if response.Failure.Message == "" || len(response.Failure.Message) > MaxFailureTextBytes || !utf8.ValidString(response.Failure.Message) {
			return errors.New("failure message is invalid")
		}
		if response.AttemptID == "" && response.Failure.Code != CodeRunnerProtocolViolation {
			return errors.New("only protocol failures may omit attemptId")
		}
	default:
		return fmt.Errorf("unknown script worker outcome %q", response.Outcome)
	}
	return nil
}

func validateCanonicalJSON(raw []byte, maxBytes int) error {
	if len(raw) == 0 || len(raw) > maxBytes {
		return fmt.Errorf("JSON value must be within 1..%d bytes", maxBytes)
	}
	if err := artifact.InspectJSONBudget(raw, MaxJSONDepth, MaxJSONNodes, maxBytes); err != nil {
		return err
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil {
		return err
	}
	if !bytes.Equal(canonical, raw) {
		return errors.New("JSON value is not RFC 8785 canonical")
	}
	return nil
}

func validFailureCode(code string) bool {
	switch code {
	case CodeSourceInvalid, CodeGuestThrown, CodeDeadlineExceeded, CodeStackExceeded,
		CodeContractViolation, CodeRunnerProtocolViolation, CodeRunnerCrashed,
		CodeIsolationUnavailable:
		return true
	default:
		return false
	}
}
