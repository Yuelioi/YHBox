// Package specs registers dependency.Extractor 实现到全局 registry.
// 给 library.ScanSubgraphDependencies / library copy 用 (扫子图引用的 templates / clips
// / callee subgraphs). 调用方 (main.go / cmd/import-fish-data / library tests) 走
// anonymous import 触发 init() 副作用.
//
// 旧 nodekind/specs/*.go 同时注册 nodekind 跟 dependency.Extractor. atomic #5 删
// nodekind 后, 这些 extractor 拆出来独立维护. 节点本身的 dependency 声明走新框架
// 的 node.Dependencies(in) 接口 (用于 e.g. Try.SubgraphID), library scanner 暂仍
// 走 dependency.Extractor — 两套并存. 收口到 nodepkg.Dependencies 是 FE registry
// 重构 plan 一起做.
package specs

import (
	"yhbox/internal/services/container/dependency"
)

type subgraphExtractor struct{}

func (subgraphExtractor) Extract(cfg map[string]any) []dependency.Dependency {
	id, _ := cfg["SubgraphID"].(string)
	if id == "" {
		return nil
	}
	return []dependency.Dependency{{Kind: dependency.KindSubgraph, Key: id}}
}

type templateRefExtractor struct{}

func (templateRefExtractor) Extract(cfg map[string]any) []dependency.Dependency {
	key, _ := cfg["template"].(string)
	if key == "" {
		return nil
	}
	return []dependency.Dependency{{Kind: dependency.KindTemplate, Key: key}}
}

type onEventExtractor struct{}

func (onEventExtractor) Extract(cfg map[string]any) []dependency.Dependency {
	if kind, _ := cfg["kind"].(string); kind != "template_appeared" {
		return nil
	}
	key, _ := cfg["template"].(string)
	if key == "" {
		return nil
	}
	return []dependency.Dependency{{Kind: dependency.KindTemplate, Key: key}}
}

type playClipExtractor struct{}

func (playClipExtractor) Extract(cfg map[string]any) []dependency.Dependency {
	id, _ := cfg["clipID"].(string)
	if id == "" {
		return nil
	}
	return []dependency.Dependency{{Kind: dependency.KindClip, Key: id}}
}

func init() {
	dependency.RegisterExtractor("Subgraph", subgraphExtractor{})
	dependency.RegisterExtractor("CollapsedNode", subgraphExtractor{})
	dependency.RegisterExtractor("Try", subgraphExtractor{})
	dependency.RegisterExtractor("WaitTemplate", templateRefExtractor{})
	dependency.RegisterExtractor("CheckTemplate", templateRefExtractor{})
	dependency.RegisterExtractor("ClickTemplate", templateRefExtractor{})
	dependency.RegisterExtractor("OnEvent", onEventExtractor{})
	dependency.RegisterExtractor("PlayClip", playClipExtractor{})
}
