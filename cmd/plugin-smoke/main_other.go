//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	_, _ = fmt.Fprintln(os.Stderr, "plugin smoke requires the Windows LPAC/AppContainer host")
	os.Exit(2)
}
