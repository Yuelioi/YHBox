// 撤销引擎 —— 纯函数, 无 Vue / Pinia, 故可单测。useContainerDraft 薄封装它。
//
// 一条历史条目 = 主图 draft 快照 + (可选) 被某次"批量改"触及的子图快照。普通主图改的
// 条目没有 sgState (加法式: 老行为不变)。批量改 (yt 控制台) 才带子图状态, 让 undo/redo
// 能把主图 + 子图一起还原 —— 详见 specs/2026-06-17-yt-scripting-console.md §撤销机制。
import type { Container, Subgraph } from "@/lib/backend";

export const HISTORY_CAP = 50;

export interface HistoryEntry {
  draft: Container;
  /** 该条目对应的、被批量改触及的子图快照 (sgID → Subgraph)。普通主图改的条目无此字段。 */
  sgState?: Record<string, Subgraph>;
}
export interface HistoryState {
  entries: HistoryEntry[];
  idx: number; // 指向当前状态对应条目; -1 = 空
}

function clone<T>(v: T): T {
  return JSON.parse(JSON.stringify(v)) as T;
}

export function makeEntry(draft: Container, sgState?: Record<string, Subgraph>): HistoryEntry {
  const e: HistoryEntry = { draft: clone(draft) };
  if (sgState && Object.keys(sgState).length > 0) e.sgState = clone(sgState);
  return e;
}

export function initHistory(draft: Container): HistoryState {
  return { entries: [makeEntry(draft)], idx: 0 };
}

export function canUndo(s: HistoryState): boolean {
  return s.idx > 0;
}
export function canRedo(s: HistoryState): boolean {
  return s.idx >= 0 && s.idx < s.entries.length - 1;
}

/** 推一条新条目: 先截断 redo 分支, 再 push; 超 cap 丢最旧。返回新 state (不改入参)。 */
export function pushEntry(s: HistoryState, entry: HistoryEntry, cap = HISTORY_CAP): HistoryState {
  let entries = s.idx < s.entries.length - 1 ? s.entries.slice(0, s.idx + 1) : s.entries.slice();
  entries.push(entry);
  let idx = entries.length - 1;
  if (entries.length > cap) {
    entries = entries.slice(entries.length - cap);
    idx = entries.length - 1;
  }
  return { entries, idx };
}

/** coalesce: 原地替换当前 top 条目 (burst 合并用, idx 不变)。 */
export function replaceTop(s: HistoryState, entry: HistoryEntry): HistoryState {
  if (s.idx < 0) return pushEntry(s, entry);
  const entries = s.entries.slice();
  entries[s.idx] = entry;
  return { entries, idx: s.idx };
}

/** 批量改前: 把触及子图的"改前"状态 merge 进当前 top 条目的 sgState (让 undo 能回到改前子图)。 */
export function augmentTop(s: HistoryState, sgState: Record<string, Subgraph>): HistoryState {
  if (s.idx < 0) return s;
  const entries = s.entries.slice();
  const cur = entries[s.idx];
  entries[s.idx] = { ...cur, sgState: { ...cur.sgState, ...clone(sgState) } };
  return { entries, idx: s.idx };
}

/** 从条目取要写回的状态 (深拷贝防别名)。subgraphs 空 = 该条目无子图状态。 */
export function restore(entry: HistoryEntry): { draft: Container; subgraphs: Subgraph[] } {
  return {
    draft: clone(entry.draft),
    subgraphs: entry.sgState ? Object.values(entry.sgState).map((sg) => clone(sg)) : [],
  };
}

export function undo(s: HistoryState): { state: HistoryState; entry: HistoryEntry | null } {
  if (!canUndo(s)) return { state: s, entry: null };
  const idx = s.idx - 1;
  return { state: { entries: s.entries, idx }, entry: s.entries[idx] ?? null };
}
export function redo(s: HistoryState): { state: HistoryState; entry: HistoryEntry | null } {
  if (!canRedo(s)) return { state: s, entry: null };
  const idx = s.idx + 1;
  return { state: { entries: s.entries, idx }, entry: s.entries[idx] ?? null };
}
