//go:build !windows

package scriptengine

func newPlatformRuntime(RuntimeOptions) platformRuntime {
	return unavailableRuntime{}
}
