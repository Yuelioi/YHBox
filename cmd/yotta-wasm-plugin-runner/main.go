package main

import (
	"context"
	"fmt"
	"os"

	"github.com/yottaapp/yotta/internal/wasmrunner"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != wasmrunner.WorkerArgument {
		os.Exit(wasmrunner.ExitInvalid)
	}
	if err := wasmrunner.Run(context.Background(), os.Stdin, os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Wasm plugin runner failed")
		os.Exit(wasmrunner.ExitRuntime)
	}
	os.Exit(wasmrunner.ExitOK)
}
