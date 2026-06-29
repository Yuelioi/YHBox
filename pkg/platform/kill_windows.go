package platform

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// taskkillArgs 根据 target 判断是 PID(纯数字)还是进程名，返回 taskkill 参数列表。
// 纯数字 → /F /PID <n>；否则 → /F /IM <name>。
func taskkillArgs(target string) []string {
	normalized := normalizeKillTarget(target)
	if isPID(normalized) {
		return []string{"/F", "/PID", normalized}
	}
	return []string{"/F", "/IM", normalized}
}

func normalizeKillTarget(target string) string {
	target = strings.Trim(strings.TrimSpace(target), `"'`)
	if strings.ContainsAny(target, `\/`) {
		return filepath.Base(target)
	}
	return target
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
// 完整 exe 路径会折成文件名；否则当进程名处理（taskkill /F /IM <name>，支持通配符 *.exe）。
// 非零退出码包装为 error，并附带 taskkill 的输出。
func KillProcess(target string) error {
	args := taskkillArgs(target)
	out, err := exec.Command("taskkill", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("KillProcess %q: %w (output: %s)", target, err, out)
	}
	return nil
}
