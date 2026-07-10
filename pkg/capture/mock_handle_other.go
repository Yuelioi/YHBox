//go:build !windows

package capture

func validateMockHandle(Handle) error { return nil }
