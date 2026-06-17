// yt 控制台 CodeMirror 补全表 —— 与 executor.ts 的 yt API 配套 (同目录, 一起 review 防漂移)。
// 喂给 scriptEditorExtensions({ completions }) 让控制台敲 `yt.` / 在 forEach 回调里有提示。
import type { Completion } from "@codemirror/autocomplete";

// detail/info 用语言中性签名 (非中文), 避免 i18n residue; 中文说明在 spec / 控制台 hint。
export const YT_COMPLETIONS: Completion[] = [
  { label: "yt.nodes", type: "variable", detail: "NodeHandle[] (main + subgraphs)" },
  { label: "yt.selected", type: "variable", detail: "NodeHandle[] (selected)" },
  { label: "yt.container", type: "variable", detail: "{ id, name }" },
  { label: "yt.log", type: "function", detail: "log(...args)" },
  // NodeHandle 方法/属性 (在 forEach(n => ...) 回调里用)
  { label: "has", type: "method", detail: "has(pin): boolean" },
  { label: "get", type: "method", detail: "get(pin): value" },
  { label: "set", type: "method", detail: "set(pin, value)" },
  { label: "kind", type: "property", detail: "string" },
  { label: "sgID", type: "property", detail: "string | null" },
  { label: "id", type: "property", detail: "string" },
  { label: "label", type: "property", detail: "string" },
];
