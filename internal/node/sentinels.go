package node

import "errors"

// ErrMustUseRegion — defensive sentinel: Region 节点 (Loop/Try/Subgraph/CollapsedNode) 的 Run()
// 被框架直调时返此 error. 正常路径是 RunNodeAsRegion → RunRegion + runner-provided body 回调.
// 节点 Run() 写法:
//
//	func (Loop) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
//		return nil, node.ErrMustUseRegion
//	}
//
// 取代之前散在 control/sentinels.go::errLoopMustUseRegion 跟 system/sentinels.go::
// errSubgraphMustUseRegion / errTryMustUseRegion 的三份本地副本.
var ErrMustUseRegion = errors.New("region node must be invoked via RunNodeAsRegion, not RunNode")

// ErrPureDataMustEvaluate — IsPureData=true 节点的 Run() 防御性 sentinel. 这些节点走
// Evaluator 接口的 EvaluatePureData 路径, Run 永不该被调.
// 取代 purefunc / variable 各自的 errPureDataNotEvaluatable 副本.
var ErrPureDataMustEvaluate = errors.New("pure-data node — evaluated via Evaluator (EvaluatePureData), not RunNode")
