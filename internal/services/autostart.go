package services

// autostart.go 把 exe 自启写入 HKCU\Software\Microsoft\Windows\CurrentVersion\Run。
// 用 HKCU（而非 HKLM）的两点理由：
//  1. 不需要管理员权限就能改
//  2. 多用户机器上每个用户独立配置
//
// 注意：用户手动改注册表 / 杀毒软件拦截写入失败时，setting 仍然保存为 true 但
// 实际不会自启。SettingsService.Update 把 ApplyAutostart 错误降级为 warning。

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

const (
	autostartRegPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	autostartRegName = "Yotta"
)

// ApplyAutostart 写或删 HKCU Run 表项。
//   - enabled=true：写入当前 exe 的绝对路径（外加 "" 包起来，路径带空格也安全）
//   - enabled=false：删除 Yotta 子键（不存在也不报错）
func ApplyAutostart(enabled bool) error {
	if !enabled {
		k, err := registry.OpenKey(registry.CURRENT_USER, autostartRegPath, registry.SET_VALUE)
		if err != nil {
			if err == registry.ErrNotExist {
				return nil
			}
			return fmt.Errorf("open Run key: %w", err)
		}
		defer k.Close()
		if err := k.DeleteValue(autostartRegName); err != nil && err != registry.ErrNotExist {
			return fmt.Errorf("delete %q: %w", autostartRegName, err)
		}
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate exe: %w", err)
	}
	// 注册表 Run 项约定路径自带双引号包裹，避免路径带空格被 cmd 切断
	value := `"` + exe + `"`

	k, _, err := registry.CreateKey(registry.CURRENT_USER, autostartRegPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open/create Run key: %w", err)
	}
	defer k.Close()
	return k.SetStringValue(autostartRegName, value)
}
