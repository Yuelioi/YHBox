//go:build !windows

package codexcli

import "os/exec"

func configureCommand(*exec.Cmd) {}
