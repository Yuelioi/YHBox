//go:build !windows

package scriptengine

func newPlatformRuntime(RuntimeOptions) (platformRuntime, error) {
	return unavailableRuntime{}, nil
}
