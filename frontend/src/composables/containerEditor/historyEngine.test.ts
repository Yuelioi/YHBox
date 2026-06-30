import { describe, it, expect } from "vitest";
import * as h from "./historyEngine";
import type { Container, Subgraph } from "@/lib/backend";

// 最小 Container, 用 jit 字段区分版本。
function draft(jit: number): Container {
  return {
    schemaVersion: 1, id: "c1", name: "c1", vars: [],
    graph: { id: "g1", schemaVersion: 1, nodes: [{ id: "n", kind: "Sleep", x: 0, y: 0, config: { literal: { JitterPct: jit } } }], edges: [] },
    tags: [], createdAt: "", updatedAt: "",
  } as unknown as Container;
}
function jitOf(c: Container): unknown {
  return (c.graph.nodes[0].config!.literal as Record<string, unknown>).JitterPct;
}
function sg(id: string, jit: number): Subgraph {
  return {
    id, rev: 1, label: id, isAnonymous: false, entry: { nodeID: "" }, outputPins: [],
    graph: { id: "sg-" + id, schemaVersion: 1, nodes: [{ id: "sn", kind: "Sleep", x: 0, y: 0, config: { literal: { JitterPct: jit } } }], edges: [] },
    createdAt: "",
  } as unknown as Subgraph;
}

describe("historyEngine", () => {
  it("init → 单条目, idx 0, 不能 undo/redo", () => {
    const s = h.initHistory(draft(0));
    expect(s.entries).toHaveLength(1);
    expect(s.idx).toBe(0);
    expect(h.canUndo(s)).toBe(false);
    expect(h.canRedo(s)).toBe(false);
  });

  it("makeEntry / restore 深拷贝独立 (改原 draft 不影响快照)", () => {
    const d = draft(1);
    const e = h.makeEntry(d);
    (d.graph.nodes[0].config!.literal as Record<string, unknown>).JitterPct = 999;
    expect(jitOf(h.restore(e).draft)).toBe(1);
  });

  it("push → 可 undo; undo/redo 在版本间移动", () => {
    let s = h.initHistory(draft(0));
    s = h.pushEntry(s, h.makeEntry(draft(1)));
    s = h.pushEntry(s, h.makeEntry(draft(2)));
    expect(s.idx).toBe(2);
    expect(h.canUndo(s)).toBe(true);

    let r = h.undo(s);
    expect(jitOf(h.restore(r.entry!).draft)).toBe(1);
    r = h.undo(r.state);
    expect(jitOf(h.restore(r.entry!).draft)).toBe(0);
    expect(h.canUndo(r.state)).toBe(false);
    const rr = h.redo(r.state);
    expect(jitOf(h.restore(rr.entry!).draft)).toBe(1);
  });

  it("undo 后 push → 截断 redo 分支", () => {
    let s = h.initHistory(draft(0));
    s = h.pushEntry(s, h.makeEntry(draft(1)));
    s = h.pushEntry(s, h.makeEntry(draft(2)));
    s = h.undo(s).state; // idx 1
    s = h.pushEntry(s, h.makeEntry(draft(9))); // 截断原 idx2
    expect(s.idx).toBe(2);
    expect(s.entries).toHaveLength(3);
    expect(h.canRedo(s)).toBe(false);
    expect(jitOf(h.restore(s.entries[2]).draft)).toBe(9);
  });

  it("超 cap 丢最旧, idx 跟着收", () => {
    let s = h.initHistory(draft(0));
    for (let i = 1; i <= h.HISTORY_CAP + 5; i++) s = h.pushEntry(s, h.makeEntry(draft(i)));
    expect(s.entries).toHaveLength(h.HISTORY_CAP);
    expect(s.idx).toBe(h.HISTORY_CAP - 1);
    // 最新版本仍在
    expect(jitOf(h.restore(s.entries[s.idx]).draft)).toBe(h.HISTORY_CAP + 5);
  });

  it("replaceTop coalesce: 原地换 top, idx 不变", () => {
    let s = h.initHistory(draft(0));
    s = h.pushEntry(s, h.makeEntry(draft(1)));
    const idxBefore = s.idx;
    s = h.replaceTop(s, h.makeEntry(draft(7)));
    expect(s.idx).toBe(idxBefore);
    expect(s.entries).toHaveLength(2);
    expect(jitOf(h.restore(s.entries[s.idx]).draft)).toBe(7);
  });

  it("批量改子图 round-trip: undo 还原改前子图, redo 还原改后", () => {
    // baseline (主图 draft0)
    let s = h.initHistory(draft(0));
    // 一次普通主图改 → idx1 (无子图)
    s = h.pushEntry(s, h.makeEntry(draft(1)));
    // 批量改: 改前 augment 当前条目带子图改前状态 (jit=10), 再 push 改后条目 (主图 draft2 + 子图 jit=20)
    s = h.augmentTop(s, { A: sg("A", 10) });
    s = h.pushEntry(s, h.makeEntry(draft(2), { A: sg("A", 20) }));
    expect(s.idx).toBe(2);

    // undo → 回到 idx1: 主图 draft1 + 子图还原成改前 jit=10
    const u = h.undo(s);
    const ru = h.restore(u.entry!);
    expect(jitOf(ru.draft)).toBe(1);
    expect(ru.subgraphs).toHaveLength(1);
    // Subgraph 跟 Container 一样有 .graph.nodes, 直接喂 jitOf。
    expect(jitOf(ru.subgraphs[0] as unknown as Container)).toBe(10);

    // redo → idx2: 主图 draft2 + 子图改后 jit=20
    const re = h.redo(u.state);
    const rr = h.restore(re.entry!);
    expect(jitOf(rr.draft)).toBe(2);
    expect(jitOf(rr.subgraphs[0] as unknown as Container)).toBe(20);
  });

  it("augmentTop merge 不覆盖已有 sgState 的其它 key", () => {
    let s = h.initHistory(draft(0));
    s = h.augmentTop(s, { A: sg("A", 1) });
    s = h.augmentTop(s, { B: sg("B", 2) });
    expect(Object.keys(s.entries[0].sgState ?? {}).sort()).toEqual(["A", "B"]);
  });
});
