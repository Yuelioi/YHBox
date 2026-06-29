import { describe, it, expect } from "vitest";
import { ytCompletionKind } from "./completions";

describe("ytCompletionKind", () => {
  it("yt. → 顶层成员", () => {
    expect(ytCompletionKind("yt.")).toBe("yt");
    expect(ytCompletionKind("foo(); yt.no")).toBe("yt");
  });

  it("yt.nodes. / yt.selected. → 数组方法", () => {
    expect(ytCompletionKind("yt.nodes.")).toBe("array");
    expect(ytCompletionKind("yt.nodes.fil")).toBe("array");
    expect(ytCompletionKind("yt.selected.")).toBe("array");
  });

  it("其它 X. → NodeHandle 成员", () => {
    expect(ytCompletionKind("n.")).toBe("node");
    expect(ytCompletionKind("yt.nodes.filter(n => n.")).toBe("node");
    expect(ytCompletionKind("yt.nodes[0].")).toBe("node");
    expect(ytCompletionKind("x.s")).toBe("node");
  });

  it("非成员上下文 → null", () => {
    expect(ytCompletionKind("yt")).toBeNull(); // 没点, 不补成员
    expect(ytCompletionKind("const a = 1")).toBeNull();
    expect(ytCompletionKind("")).toBeNull();
  });

  it("yt.<半个成员名> 仍判 yt (过滤到该成员)", () => {
    // 敲到 'yt.nodes' (光标在词尾, 还没打点) → 仍是 yt 成员上下文, 补全过滤出 nodes。
    expect(ytCompletionKind("yt.nodes")).toBe("yt");
  });

  it("优先级: yt.nodes. 判 array 而非 node", () => {
    // 'yt.nodes.' 同时能匹配 array 和 generic X. → 必须先判 array
    expect(ytCompletionKind("yt.nodes.")).toBe("array");
  });
});
