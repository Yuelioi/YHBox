//go:build windows

package capture

import (
	"errors"
	"fmt"
	"sync"
)

var (
	wgcInitOnce sync.Once
	wgcInitErr  error
)

func ensureWGCInit() error {
	wgcInitOnce.Do(func() {
		wgcInitErr = initWGC()
	})
	return wgcInitErr
}

// NewIBackend distinguishes conservative automatic selection from explicit WGC:
// auto requires Windows build 20348 because older DirectX games can return black
// or stale WGC frames, while explicit WGC is accepted from its API floor 18362.
func NewIBackend(name string) (b IBackend, warning string, err error) {
	switch name {
	case "", "auto":
		switch AutoBackend() {
		case BackendWGC:
			if wb, e := newWGCBackend(); e == nil {
				return wb, "", nil
			}
			if gb, e := newGDIBackend(); e == nil {
				return gb, "", nil
			}
			return nil, "", errors.New("auto: WGC + GDI 都初始化失败")
		default:
			if gb, e := newGDIBackend(); e == nil {
				return gb, "", nil
			}
			return nil, "", errors.New("auto: GDI init 失败")
		}
	case "wgc":
		if WindowsBuild() >= 18362 {
			if wb, e := newWGCBackend(); e == nil {
				return wb, "", nil
			}
			if gb, e := newGDIBackend(); e == nil {
				return gb, "WGC 初始化失败, fallback 到 GDI", nil
			}
			return nil, "", errors.New("wgc: WGC + GDI 都失败")
		}
		if gb, e := newGDIBackend(); e == nil {
			return gb, fmt.Sprintf("WGC 要求 Windows 10 1903+, 当前 build %d, fallback 到 GDI", WindowsBuild()), nil
		}
		return nil, "", errors.New("wgc: unsupported OS 且 GDI fallback 也失败")
	case "gdi":
		if gb, e := newGDIBackend(); e == nil {
			return gb, "", nil
		}
		return nil, "", errors.New("gdi: init 失败 (无 fallback for explicit gdi)")
	case "mock":
		if mb, e := newMockBackend(); e == nil {
			return mb, "", nil
		} else {
			return nil, "", e
		}
	default:
		return nil, "", fmt.Errorf("unknown capture backend %q (supported: auto/wgc/gdi/mock)", name)
	}
}
