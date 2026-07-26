//go:build windows

package input

import "github.com/lxn/win"

// Handle identifies the native window targeted by the legacy Windows input adapter.
type Handle = win.HWND
