// Package botcore loader helpers — embedded yaml + exe-dir override merge。
package botcore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadYAMLWithOverride 从 fsys 读 embedPath，反序列化进 dst。
// 如果 exe 同目录 <overrideRelPath> 存在，**继续**用同 dst 反序列化（merge 语义）：
// override 部分覆盖已填字段；未提及的字段保持 embedded 默认值。
//
// 调用例：
//
//	var cfg Config
//	err := botcore.LoadYAMLWithOverride(
//	    rhythmFS, "configs/zh/config.yaml",
//	    "configs/rhythm/zh/config.yaml",  // exe-dir 相对路径
//	    &cfg,
//	)
//
// 错误语义：
//   - embed 缺失或解析失败：返回错误（配置错误）
//   - override 文件不存在：忽略，使用 embed 默认值（不算错）
//   - override 解析失败：返回错误（用户写错了 yaml 想知道）
//   - os.Executable 失败：忽略 override 阶段，使用 embed 默认值（极少触发，不算错）
func LoadYAMLWithOverride(fsys fs.FS, embedPath string, overrideRelPath string, dst any) error {
	data, err := fs.ReadFile(fsys, embedPath)
	if err != nil {
		return fmt.Errorf("read embed %s: %w", embedPath, err)
	}
	if err := yaml.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("parse embed %s: %w", embedPath, err)
	}

	exe, err := os.Executable()
	if err != nil {
		return nil
	}
	overrideFull := filepath.Join(filepath.Dir(exe), overrideRelPath)
	overrideData, err := os.ReadFile(overrideFull)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read override %s: %w", overrideFull, err)
	}
	if err := yaml.Unmarshal(overrideData, dst); err != nil {
		return fmt.Errorf("parse override %s: %w", overrideFull, err)
	}
	return nil
}
