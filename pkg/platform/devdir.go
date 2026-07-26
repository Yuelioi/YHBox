package platform

import "path/filepath"

// DevOutputDir 是开发者本地工具（calibrate / CSV 导出 / 帧截图等）的默认输出目录。
// 项目根 .gitignore 已排除此目录；用户最终发行版无需关心，运行时也只在
// dev/调试路径上使用。
const DevOutputDir = "debug"

// DevOutputPath 拼接 DevOutputDir 子路径。
func DevOutputPath(parts ...string) string {
	return filepath.Join(append([]string{DevOutputDir}, parts...)...)
}
