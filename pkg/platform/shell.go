package platform

import (
	"fmt"
	"syscall"
	"unsafe"
)

// ShellExecuteW nShowCmd 常量 (winuser.h SW_*).
const (
	SWHide            = 0 // SW_HIDE        — 隐藏启动 (无窗口)
	SWShowNormal      = 1 // SW_SHOWNORMAL  — 正常窗口
	SWShowMaximized   = 3 // SW_SHOWMAXIMIZED
	SWShowMinNoActive = 7 // SW_SHOWMINNOACTIVE — 最小化启动且不抢焦点
)

// ShellOpen 用 ShellExecuteW 的 "open" verb 启动 target —— 可执行文件直接运行、URL 走
// 默认浏览器、文档/文件夹走系统默认关联程序。启动即返回 (fire-and-forget)，不等进程退出，
// 不闪控制台窗口。args / workingDir 留空即省略；showCmd 用上面 SW* 常量。
//
// ShellExecuteW 返回值约定: > 32 成功, ≤ 32 是错误码 (e.g. 2=文件未找到, 5=拒绝访问)。
func ShellOpen(target, args, workingDir string, showCmd int) error {
	verb, _ := syscall.UTF16PtrFromString("open")
	file, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return fmt.Errorf("非法 target 路径 %q: %w", target, err)
	}
	var argsPtr, dirPtr *uint16
	if args != "" {
		argsPtr, _ = syscall.UTF16PtrFromString(args)
	}
	if workingDir != "" {
		dirPtr, _ = syscall.UTF16PtrFromString(workingDir)
	}

	ret, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(argsPtr)), // nil → NULL，无参数
		uintptr(unsafe.Pointer(dirPtr)),  // nil → NULL，默认工作目录
		uintptr(showCmd),
	)
	if ret <= 32 {
		return fmt.Errorf("ShellExecute 失败 (code=%d)", ret)
	}
	return nil
}
