//go:build !windows

package capture

// Handle keeps the capture contract and mock adapter portable across host OSes.
type Handle = uintptr
