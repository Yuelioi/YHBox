//go:build !windows

package platform

func KillProcess(string) error {
	return NewUnsupportedError("kill process")
}
