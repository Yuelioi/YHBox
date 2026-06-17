import { describe, it, expect } from "vitest";
import { ref } from "vue";
import { useYtConsole } from "./useYtConsole";
import type { Container, Subgraph } from "@/lib/backend";
import type { PinType } from "@/lib/ytConsole/executor";

function mainDraft(): Container {
  return {
    schemaVersion: 1, id: "c1", name: "C", vars: [],
    graph: { id: "g", version: 1, nodes: [{ id: "a", kind: "Sleep", x: 0, y: 0, config: { literal: { JitterPct: 0 } } }], edges: [] },
    tags: [], createdAt: "", updatedAt: "",
  } as unknown as Container;
}
function subA(): Subgraph {
  return {
    id: "sub-A", rev: 1, label: "A", isAnonymous: false, entry: { nodeID: "" }, outputPins: [],
    graph: { id: "sgA", version: 1, nodes: [{ id: "s1", kind: "Sleep", x: 0, y: 0, config: { literal: {} } }], edges: [] },
    createdAt: "",
  } as unknown as Subgraph;
}
const SPEC_PINS: Record<string, Record<string, PinType>> = { Sleep: { Duration: "number", JitterPct: "number" } };
const DEFAULTS: Record<string, Record<string, unknown>> = { Sleep: { Duration: 1000, JitterPct: 0 } };

function setup(cap: { touched?: string[] } = {}) {
  const draft = ref<Container>(mainDraft());
  const sub = subA();
  const subs = [sub];
  const yt = useYtConsole({
    draft,
    selectedIds: () => [],
    subgraphs: () => subs,
    subgraphById: (id) => subs.find((s) => s.id === id),
    specPinsOf: (k) => SPEC_PINS[k] ?? {},
    defaultsOf: (k) => DEFAULTS[k] ?? {},
    // fake: 记下 touchedSgIDs, 并同步跑 mutator (主图改 draft; 子图改经 subgraphById)。
    applyBulkMutation: (touched, mutator) => {
      cap.touched = touched;
      mutator(draft.value!);
    },
  });
  return { draft, sub, yt };
}
const lit = (n: { config?: { literal?: Record<string, unknown> } }) => n.config!.literal!;

describe("useYtConsole", () => {
  it("buildModel: 主图 + 子图节点都进, sgID/defaults/specPins 对", () => {
    const { yt } = setup();
    const m = yt.buildModel();
    expect(m.map((x) => [x.id, x.sgID])).toEqual([["a", null], ["s1", "sub-A"]]);
    expect(m[1].defaults).toEqual({ Duration: 1000, JitterPct: 0 });
    expect(m[1].specPins).toEqual({ Duration: "number", JitterPct: "number" });
  });

  it("run: 批量改主图+子图 → applyBulkMutation 收到 touchedSgIDs, 两边 literal 都写", () => {
    const cap: { touched?: string[] } = {};
    const { draft, sub, yt } = setup(cap);
    const r = yt.run(`yt.nodes.filter(n => n.has('JitterPct')).forEach(n => n.set('JitterPct', 10))`);
    expect(r.error).toBeUndefined();
    expect(r.nodeCount).toBe(2);
    expect(cap.touched).toEqual(["sub-A"]); // 主图节点 sgID=null 不进 touched
    expect(lit(draft.value.graph.nodes[0]).JitterPct).toBe(10);
    expect(lit(sub.graph.nodes[0]).JitterPct).toBe(10);
  });

  it("脚本抛错 → 不应用 (applyBulkMutation 不被调, 节点不变)", () => {
    const cap: { touched?: string[] } = {};
    const { draft, yt } = setup(cap);
    const r = yt.run(`throw new Error('boom')`);
    expect(r.error).toContain("boom");
    expect(cap.touched).toBeUndefined();
    expect(lit(draft.value.graph.nodes[0]).JitterPct).toBe(0);
  });

  it("全被拒 (无此 pin) → applied 空, 不调 applyBulkMutation", () => {
    const cap: { touched?: string[] } = {};
    const { yt } = setup(cap);
    const r = yt.run(`yt.nodes.forEach(n => n.set('Nope', 1))`);
    expect(r.applied).toEqual([]);
    expect(r.rejected.length).toBe(2);
    expect(cap.touched).toBeUndefined();
  });
});
