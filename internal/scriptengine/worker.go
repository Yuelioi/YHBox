package scriptengine

import (
	"errors"
	"io"
)

const (
	WorkerExitOK       = 0
	WorkerExitProtocol = 2
)

var (
	errWorkerNotAppContainer = errors.New("worker is not an AppContainer")
	errWorkerNotLPAC         = errors.New("worker is not an LPAC")
	errWorkerNotInJob        = errors.New("worker is not in a Job Object")
)

func IsWorkerCommand(arguments []string) bool {
	return len(arguments) == 1 && arguments[0] == WorkerArgument
}

// ServeOne consumes exactly one request, writes exactly one response, and
// returns. A protocol error is itself reported as a typed response whenever
// stdout remains writable.
func ServeOne(input io.Reader, output io.Writer) int {
	return serveOne(input, output, verifyWorkerConfinement)
}

func serveOne(input io.Reader, output io.Writer, verify func() error) int {
	request, err := ReadRequest(input)
	if err != nil {
		response := failedResponse("", CodeRunnerProtocolViolation, "script worker request violated the protocol")
		if writeErr := WriteResponse(output, response); writeErr != nil {
			return WorkerExitProtocol
		}
		return WorkerExitOK
	}
	if err := verify(); err != nil {
		message := "script worker confinement could not be verified"
		switch {
		case errors.Is(err, errWorkerNotAppContainer):
			message = "script worker is not an AppContainer"
		case errors.Is(err, errWorkerNotLPAC):
			message = "script worker is not a less-privileged AppContainer"
		case errors.Is(err, errWorkerNotInJob):
			message = "script worker is not assigned to a Job Object"
		}
		response := failedResponse(request.AttemptID, CodeIsolationUnavailable, message)
		if writeErr := WriteResponse(output, response); writeErr != nil {
			return WorkerExitProtocol
		}
		return WorkerExitOK
	}
	if err := WriteResponse(output, executeGuest(request)); err != nil {
		return WorkerExitProtocol
	}
	return WorkerExitOK
}
