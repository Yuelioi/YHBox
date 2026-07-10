//go:build !windows

package platform

func IsAdmin() bool { return true }

func RelaunchAsAdmin() bool { return false }

func EnsureAdmin() {}
