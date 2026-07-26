package main

import (
	"os"

	"github.com/yottaapp/yotta/internal/scriptengine"
)

func main() {
	if !scriptengine.IsWorkerCommand(os.Args[1:]) {
		os.Exit(scriptengine.WorkerExitProtocol)
	}
	os.Exit(scriptengine.ServeOne(os.Stdin, os.Stdout))
}
