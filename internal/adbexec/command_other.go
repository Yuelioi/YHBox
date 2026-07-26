//go:build !windows

package adbexec

import "os/exec"

func hideCommandWindow(*exec.Cmd) {}
