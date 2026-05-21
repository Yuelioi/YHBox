package dependency

import (
	"fmt"
	"sort"
)

// extractorByKind 全局 node kind → Extractor 注册表.
// specs/*.go init() 通过 RegisterExtractor 填充.
var extractorByKind = map[string]Extractor{}

// RegisterExtractor 注册某个 node kind 的依赖提取器. specs/*.go init() 调用.
// 跟 nodekind.Register 拆开是因为 nodekind 不能 import dependency
// (会经 container 闭环). 这边自管 registry.
func RegisterExtractor(kind string, ext Extractor) {
	if kind == "" {
		panic("dependency.RegisterExtractor: empty kind")
	}
	if ext == nil {
		panic(fmt.Sprintf("dependency.RegisterExtractor: nil extractor for %q", kind))
	}
	if _, dup := extractorByKind[kind]; dup {
		panic(fmt.Sprintf("dependency.RegisterExtractor: duplicate kind %q", kind))
	}
	extractorByKind[kind] = ext
}

// GetExtractor 给 scanner 用. 未注册 kind 返 nil (= 无依赖, scanner 跳过).
func GetExtractor(kind string) Extractor {
	return extractorByKind[kind]
}

// VerifyDetectGroupHasExtractors 启动期 fail-fast 检查 — 所有 detect group 节点必须注册 extractor.
// requiredKinds 是 nodekind 包提供的 "detect" group 全名单 (caller 在 main / test 调用).
// 不直接 import nodekind 避免循环 — 由 caller 传入字符串列表.
func VerifyDetectGroupHasExtractors(requiredKinds []string) error {
	var missing []string
	for _, k := range requiredKinds {
		if _, ok := extractorByKind[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("detect group missing DependencyExtractor for kinds: %v (register in specs/*.go init)", missing)
	}
	return nil
}
