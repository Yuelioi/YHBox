package container

// validateSentinelScope 拦截 Break/Continue/Throw 作用域错置 (P1.2).
//
// Break / Continue: 必须 exec 可达自同图某个 Loop 节点的 body 出口 — 否则 sentinel
// 漏到顶层 dispatch, 当前只能 emit generic error. fishing-v2 所有 Break 都跟其
// target Loop 同图同 subgraph, 此检查不会误报.
//
// Throw: 必须在某个 Try 节点的 body subgraph 内 — Try 节点 config.SubgraphID 引用
// 的子图. main graph + 非 Try-body subgraph 内放 Throw → 漏到顶层 panic-level fail.
//
// Limitations (双 graph 不分析):
//   - Break inside subgraph A, A called from Subgraph 节点 in Loop body of main →
//     运行时 OK (sentinel 透传到 main Loop), 但此检查 emit BREAK_OUTSIDE_LOOP.
//     当前 fishing-v2 无此用法, 真要支持往 follow-up: 加 call-graph 分析标"调用栈含
//     Loop body 的 subgraph 集合".
//   - Throw 同理: 只放过 Try-body 子图本身, 不传 transitively.
func validateSentinelScope(c *Container) []ValidationError {
	if c == nil {
		return nil
	}
	var errs []ValidationError

	// 找出所有被 Try.SubgraphID 引用的子图 — Throw 只允许在这些里.
	tryBodySubgraphs := map[string]struct{}{}
	collectTryBodies := func(g Graph) {
		for _, n := range g.Nodes {
			if n.Kind != "Try" {
				continue
			}
			if sgID, _ := n.Config["SubgraphID"].(string); sgID != "" {
				tryBodySubgraphs[sgID] = struct{}{}
			}
		}
	}
	collectTryBodies(c.Graph)
	for i := range c.Subgraphs {
		collectTryBodies(c.Subgraphs[i].Graph)
	}

	checkGraph := func(g Graph, path []string, subgraphID string) []ValidationError {
		var out []ValidationError

		inLoopBody := buildLoopBodyReachSet(g)
		_, graphIsTryBody := tryBodySubgraphs[subgraphID]

		for _, n := range g.Nodes {
			switch n.Kind {
			case "Break":
				if _, ok := inLoopBody[n.ID]; !ok {
					out = append(out, ValidationError{
						Severity:  SeverityError,
						Code:      CodeBreakOutsideLoop,
						GraphPath: path,
						NodeID:    n.ID,
						Message:   "Break 节点必须在 Loop 的 body 下游 (同图内 exec 可达)",
					})
				}
			case "Continue":
				if _, ok := inLoopBody[n.ID]; !ok {
					out = append(out, ValidationError{
						Severity:  SeverityError,
						Code:      CodeContinueOutsideLoop,
						GraphPath: path,
						NodeID:    n.ID,
						Message:   "Continue 节点必须在 Loop 的 body 下游 (同图内 exec 可达)",
					})
				}
			case "Throw":
				if !graphIsTryBody {
					out = append(out, ValidationError{
						Severity:  SeverityError,
						Code:      CodeThrowOutsideTry,
						GraphPath: path,
						NodeID:    n.ID,
						Message:   "Throw 节点必须在被 Try.SubgraphID 引用的子图内",
					})
				}
			}
		}
		return out
	}

	errs = append(errs, checkGraph(c.Graph, []string{"main"}, "")...)
	for i := range c.Subgraphs {
		sg := &c.Subgraphs[i]
		errs = append(errs, checkGraph(sg.Graph, []string{"subgraph", sg.ID}, sg.ID)...)
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
