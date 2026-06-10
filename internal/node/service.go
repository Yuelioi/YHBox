// internal/node/service.go
package node

import (
	"fmt"
	"slices"

	"yotta/internal/services/expr"
)

// NodeService Wails3 RPC service. FE 启动调 GetAllNodeSpecs / GetAllTypes / AsyncOptions.
type NodeService struct {
	// AsyncOptions 注册表: asyncSource → handler.
	asyncSources map[string]AsyncOptionsHandler
}

// AsyncOptionsHandler 处理一个 asyncSource 的回调.
//   - nodeID: 当前节点 instance ID (FE 传)
//   - specKind: 节点 Kind ("CheckTemplate")
//   - params: 节点当前已填的其他字段值 (用于上下文相关 dropdown, e.g. Switch case 数)
type AsyncOptionsHandler func(nodeID, specKind string, params map[string]any) ([]EnumOption, error)

func NewService() *NodeService {
	return &NodeService{
		asyncSources: map[string]AsyncOptionsHandler{},
	}
}

// RegisterAsyncSource main.go 在启动期调, 注册各种 async data source.
// 注册顺序无关. 重复注册 panic (fail fast author bug).
func (s *NodeService) RegisterAsyncSource(name string, handler AsyncOptionsHandler) {
	if _, exists := s.asyncSources[name]; exists {
		panic(fmt.Sprintf("AsyncSource %q already registered", name))
	}
	s.asyncSources[name] = handler
}

// GetAllNodeSpecs FE 启动拉一次, 缓存. RPC binding 自动生成 TS 类型.
func (s *NodeService) GetAllNodeSpecs() []Spec {
	all := All()
	out := make([]Spec, 0, len(all))
	for _, rn := range all {
		out = append(out, rn.Spec)
	}
	return out
}

// GetAllTypes FE 启动拉一次, 获颜色 + widget 映射.
func (s *NodeService) GetAllTypes() []TypeSpec {
	return AllTypes()
}

// GetExprFunctions FE 启动拉一次 — Expr 编辑器补全/未知函数检查的函数元数据.
// 单一来源 expr/builtins.go; 加函数 = 改那张表 + zh/en 各 1 条 desc, FE 零手抄.
func (s *NodeService) GetExprFunctions() []expr.FunctionSpec {
	return expr.Functions()
}

// GetScriptBindableKinds — Script 编辑器补全用: 可在脚本里调用的节点 kind 全集.
func (s *NodeService) GetScriptBindableKinds() []string {
	var kinds []string
	for _, rn := range All() {
		if ScriptBindable(rn) {
			kinds = append(kinds, rn.Spec.Kind)
		}
	}
	slices.Sort(kinds)
	return kinds
}

// GetErrorCodes FE 启动拉一次, 供 Switch/UI 提示已知错误码全集.
func (s *NodeService) GetErrorCodes() []ErrCode {
	out := make([]ErrCode, 0, len(ErrorCodes))
	for c := range ErrorCodes {
		out = append(out, c)
	}
	return out
}

// AsyncOptions 动态 dropdown 数据源. Widget.Props.AsyncSource 标识用哪个 handler.
func (s *NodeService) AsyncOptions(nodeID, specKind, asyncSource string, currentInputs map[string]any) ([]EnumOption, error) {
	h, ok := s.asyncSources[asyncSource]
	if !ok {
		return nil, fmt.Errorf("unknown asyncSource %q", asyncSource)
	}
	return h(nodeID, specKind, currentInputs)
}
