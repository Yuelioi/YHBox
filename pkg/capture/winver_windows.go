// OS build 号检测，给"auto 选 backend"用：
//   - Win11 / Server 2022 (build >= 20348) 走 WGC（黄框可关 + 部分场景帧更稳）
//   - 老系统走 GDI（兼容性最好）
//
// 用 ntdll.RtlGetVersion 而不是 GetVersionEx：后者从 Win8.1 起被 Microsoft 限制，
// 不带兼容性 manifest 时会一直返回 6.2（Win8）。RtlGetVersion 是 NT 内部 API，
// 不受兼容性 shim 影响，永远拿真实版本。
//
// _windows.go 后缀因为 syscall.LazyDLL 是 Windows 专属。

package capture

import (
	"syscall"
	"unsafe"
)

// osVersionInfoEx 跟 Windows OSVERSIONINFOEXW 对齐（284 字节）。
// 只需要 dwBuildNumber，其它字段读出来不用。
type osVersionInfoEx struct {
	OSVersionInfoSize uint32
	MajorVersion      uint32
	MinorVersion      uint32
	BuildNumber       uint32
	PlatformID        uint32
	CSDVersion        [128]uint16
	ServicePackMajor  uint16
	ServicePackMinor  uint16
	SuiteMask         uint16
	ProductType       byte
	Reserved          byte
}

var (
	procRtlGetVersion = syscall.NewLazyDLL("ntdll.dll").NewProc("RtlGetVersion")
)

// WindowsBuild 返回 Windows build 号。调用失败返 0（caller 应当成"未知，保守走 GDI"）。
// 典型值参考：
//   - 19041..19045: Windows 10 (1903 到 22H2)
//   - 20348: Windows Server 2022
//   - 22000+: Windows 11
func WindowsBuild() uint32 {
	var info osVersionInfoEx
	info.OSVersionInfoSize = uint32(unsafe.Sizeof(info))
	r, _, _ := procRtlGetVersion.Call(uintptr(unsafe.Pointer(&info)))
	if r != 0 {
		// NTSTATUS != 0 表示失败（STATUS_SUCCESS = 0）
		return 0
	}
	return info.BuildNumber
}

// AutoBackend 根据 OS 选最优后端：
//   - build >= 20348 (Win11 / Server 2022)：WGC，黄框可关 + DX 游戏后台帧更稳
//   - 其它：GDI，兼容性优先
//
// 调用方在 settings.capture.method = "auto" 时用这个值。
func AutoBackend() Backend {
	if WindowsBuild() >= 20348 {
		return BackendWGC
	}
	return BackendGDI
}
