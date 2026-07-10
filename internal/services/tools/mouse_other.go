//go:build !windows

package tools

func readCursor() (int, int, bool) { return 0, 0, false }

func screenToClient(uintptr, int, int) (int, int, bool) { return 0, 0, false }
