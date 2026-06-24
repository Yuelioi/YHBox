package platform

import (
	"fmt"
	"os/exec"
)

// taskkillArgs 根据 target 判断是 PID(纯数字)还是进程名，返回 taskkill 参数列表。
// 纯数字 → /F /PID <n>；否则 → /F /IM <name>。
func taskkillArgs(target string) []string {
	if isPID(target) {
		return []string{"/F", "/PID", target}
	}
	return []string{"/F", "/IM", target}
}

// isPID 判断字符串是否全为数字（即 Windows PID）。
func isPID(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// KillProcess 强制结束进程。
// target 为纯数字时当 PID 处理（taskkill /F /PID <n>），
// 否则当进程名处理（taskkill /F /IM <name>，支持通配符 *.exe）。
// 非零退出码包装为 error，并附带 taskkill 的输出。
func KillProcess(target string) error {
	args := taskkillArgs(target)
	out, err := exec.Command("taskkill", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("KillProcess %q: %w (output: %s)", target, err, out)
	}
	return nil
}
