package scriptengine

import "io"

const (
	WorkerExitOK       = 0
	WorkerExitProtocol = 2
)

func IsWorkerCommand(arguments []string) bool {
	return len(arguments) == 1 && arguments[0] == WorkerArgument
}

// ServeOne consumes exactly one request, writes exactly one response, and
// returns. A protocol error is itself reported as a typed response whenever
// stdout remains writable.
func ServeOne(input io.Reader, output io.Writer) int {
	request, err := ReadRequest(input)
	if err != nil {
		response := failedResponse("", CodeRunnerProtocolViolation, "script worker request violated the protocol")
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
