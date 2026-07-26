//go:build windows

package capture

import "github.com/lxn/win"

// Handle identifies the native window targeted by a Windows capture adapter.
type Handle = win.HWND
