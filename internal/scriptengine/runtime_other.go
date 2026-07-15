//go:build !windows

package scriptengine

func newPlatformRuntime(RuntimeOptions) Runtime {
	return unavailableRuntime{}
}
