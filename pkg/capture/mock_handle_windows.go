//go:build windows

package capture

import "errors"

func validateMockHandle(hwnd Handle) error {
	if hwnd != 0 && !isWindow(hwnd) {
		return errors.New("mock.ClientSize: invalid hwnd")
	}
	return nil
}
