//go:build !windows

package input

// Handle keeps the backend contract buildable where no native Windows adapter exists.
type Handle = uintptr
