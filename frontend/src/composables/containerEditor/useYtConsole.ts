// useYtConsole — yt 脚本控制台的 glue: 把编辑器草稿 + 子图 + 节点 spec 组装成执行器的
// NodeModel[], 跑用户脚本, 再把变更经 applyBulkMutation 一次性落地 (主图 + 触及子图一条可撤销条目)。
// 依赖以函数注入 (DI): 真实使用时由 view 传 editorStore / getSpec / KIND_DEFAULTS, 单测传 fake。
import type { Ref } from "vue";
import type { Container, Subgraph, GraphNode } from "@/lib/backend";
import { runConsoleScript, type NodeModel, type PinType, type RunResult, type Change } from "@/lib/ytConsole/executor";

export interface YtConsoleDeps {
  draft: Ref<Container | null>;
  applyBulkMutation: (touchedSgIDs: string[], mutator: (d: Container) => void) => void;
  selectedIds: () => string[];
  subgraphs: () => Subgraph[];
  subgraphById: (id: string) => Subgraph | undefined;
  specPinsOf: (kind: string) => Record<string, PinType>;
  defaultsOf: (kind: string) => Record<string, unknown>;
}

// reactivity-safe 写 literal (浅克隆 config + literal, 同 setPinLiteral)。
function writeLiteral(n: GraphNode, pin: string, value: unknown): void {
  n.config = { ...n.config, literal: { ...((n.config?.literal as Record<string, unknown>) ?? {}), [pin]: value } };
}

export function useYtConsole(deps: YtConsoleDeps) {
  function buildModel(): NodeModel[] {
    const d = deps.draft.value;
    if (!d) return [];
    const out: NodeModel[] = [];
    const push = (n: GraphNode, sgID: string | null) =>
      out.push({
        id: n.id,
        kind: n.kind,
        label: n.label ?? "",
        sgID,
        literal: (n.config?.literal as Record<string, unknown>) ?? {},
        defaults: deps.defaultsOf(n.kind),
        specPins: deps.specPinsOf(n.kind),
      });
    for (const n of d.graph.nodes) push(n, null);
    for (const sg of deps.subgraphs()) {
      if (!sg.graph) continue;
      for (const n of sg.graph.nodes) push(n, sg.id);
    }
    return out;
  }

  function applyChanges(applied: Change[]): void {
    if (!applied.length) return;
    const touched = [...new Set(applied.map((c) => c.sgID).filter((x): x is string => x !== null))];
    deps.applyBulkMutation(touched, (d) => {
      for (const c of applied) {
        const g = c.sgID === null ? d.graph : deps.subgraphById(c.sgID)?.graph;
        const n = g?.nodes.find((x) => x.id === c.nodeId);
        if (n) writeLiteral(n, c.pin, c.value);
      }
    });
  }

  function run(code: string): RunResult {
    const d = deps.draft.value;
    const r = runConsoleScript(code, {
      nodes: buildModel(),
      selectedIds: deps.selectedIds(),
      container: { id: d?.id ?? "", name: d?.name ?? "" },
    });
    if (!r.error) applyChanges(r.applied);
    return r;
  }

  return { run, buildModel };
}
