package container

// validateSentinelScope 拦截 Break/Continue 作用域错置.
//
// Break / Continue: 必须 exec 可达自同图某个 Loop 节点的 body 出口 — 否则 sentinel
// 漏到顶层 dispatch, 当前只能 emit generic error. fishing-v2 所有 Break 都跟其
// target Loop 同图同 subgraph, 此检查不会误报.
//
// Throw 不再受 scope 限制: Try 删除后 Throw 实现 node.Coded, 由 region 的 Fail 出口
// 截获 (没接就冒泡到顶层), 任意位置合法.
//
// Limitations (双 graph 不分析):
//   - Break inside subgraph A, A called from Subgraph 节点 in Loop body of main →
//     运行时 OK (sentinel 透传到 main Loop), 但此检查 emit BREAK_OUTSIDE_LOOP.
//     当前 fishing-v2 无此用法, 真要支持往 follow-up: 加 call-graph 分析标"调用栈含
//     Loop body 的 subgraph 集合".
func validateSentinelScope(c *Container) []ValidationError {
	if c == nil {
		return nil
	}
	var errs []ValidationError

	checkGraph := func(g Graph, path []string) []ValidationError {
		var out []ValidationError

		inLoopBody := buildLoopBodyReachSet(g)

		for _, n := range g.Nodes {
			switch n.Kind {
			case "Break":
				if _, ok := inLoopBody[n.ID]; !ok {
					out = append(out, ValidationError{
						Severity:  SeverityError,
						Code:      CodeBreakOutsideLoop,
						GraphPath: path,
						NodeID:    n.ID,
					})
				}
			case "Continue":
				if _, ok := inLoopBody[n.ID]; !ok {
					out = append(out, ValidationError{
						Severity:  SeverityError,
						Code:      CodeContinueOutsideLoop,
						GraphPath: path,
						NodeID:    n.ID,
					})
				}
			}
		}
		return out
	}

	errs = append(errs, checkGraph(c.Graph, []string{"main"})...)
	for i := range c.Subgraphs {
		sg := &c.Subgraphs[i]
		errs = append(errs, checkGraph(sg.Graph, []string{"subgraph", sg.ID})...)
	}
	return errs
}

// buildLoopBodyReachSet 跑 BFS 从图内所有 Loop.body 出口走 exec 边, 返回 "在某个 Loop
// body 下游" 的 nodeID 集合. 嵌套 Loop 不专门处理 — 内层 Loop 自己的 Break 也在外层
// 集合里, 同样判 OK.
//
// 只走 exec 边 — data 边不传 control flow. Edge.From 从 nodekind 角度判 IsDataOutPin
// → 是 data 出口跳过.
func buildLoopBodyReachSet(g Graph) map[string]struct{} {
	reach := map[string]struct{}{}

	// edge index: fromNodeID → list of toNodeIDs (exec 边).
	type fromPin struct {
		nodeID string
		pin    string
	}
	execOut := map[fromPin][]string{}
	for _, e := range g.Edges {
		fromID, fromPinName := splitRef(e.From)
		toID, _ := splitRef(e.To)
		// data 边跳过 — exec 流不走 data.
		if IsDataOutPin(graphNodeKind(g, fromID), fromPinName) {
			continue
		}
		execOut[fromPin{fromID, fromPinName}] = append(execOut[fromPin{fromID, fromPinName}], toID)
	}

	// 从每个 Loop.body seed BFS.
	for _, n := range g.Nodes {
		if n.Kind != "Loop" {
			continue
		}
		seeds := execOut[fromPin{n.ID, "Body"}]
		queue := append([]string{}, seeds...)
		for len(queue) > 0 {
			nid := queue[0]
			queue = queue[1:]
			if _, seen := reach[nid]; seen {
				continue
			}
			reach[nid] = struct{}{}
			// 走该 node 所有 exec-out edge.
			for key, tos := range execOut {
				if key.nodeID != nid {
					continue
				}
				queue = append(queue, tos...)
			}
		}
	}
	return reach
}

// graphNodeKind 查 graph 里 nodeID 对应 kind. 找不到返 "" (validator 其他规则会 flag DANGLING).
func graphNodeKind(g Graph, nodeID string) string {
	for i := range g.Nodes {
		if g.Nodes[i].ID == nodeID {
			return g.Nodes[i].Kind
		}
	}
	return ""
}
