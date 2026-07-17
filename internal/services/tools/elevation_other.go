//go:build !windows

package tools

func processIsElevated() bool { return false }
