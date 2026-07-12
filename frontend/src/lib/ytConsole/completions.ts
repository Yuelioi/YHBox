// yt 控制台补全 + hover — 喂给 scriptEditorExtensions (跟 Script 节点编辑器同一套引擎)。
// label 用**带点的成员形式** (yt.nodes / n.set): scriptCompletionSource 按带点词
// (matchBefore /[\w$.]*/) 过滤 label, 故敲 `yt.` / `n.` 才能浮出成员 (裸 'set' 不会)。
// detail/desc 用英文 (语言中性, 避免 i18n residue; 中文说明在 spec / 控制台 hint 文案)。
import type { Completion, CompletionContext, CompletionResult } from '@codemirror/autocomplete'
import type { HoverDoc } from '@/lib/editorHover'

export interface YtEntry {
  label: string
  type: Completion['type']
  sig: string
  desc: string
}

export const YT_ENTRIES: YtEntry[] = [
  {
    label: 'yt.nodes',
    type: 'variable',
    sig: 'yt.nodes: NodeHandle[]',
    desc: 'All nodes in the current container (main graph + all subgraphs), frozen read-only array — use .filter / .map / .forEach.',
  },
  {
    label: 'yt.selected',
    type: 'variable',
    sig: 'yt.selected: NodeHandle[]',
    desc: 'Nodes selected on the canvas when the console was opened.',
  },
  {
    label: 'yt.container',
    type: 'variable',
    sig: 'yt.container: { id, name }',
    desc: 'The current container (read-only).',
  },
  {
    label: 'yt.log',
    type: 'function',
    sig: 'yt.log(...args)',
    desc: 'Print to the output area below.',
  },
  {
    label: 'n.has',
    type: 'method',
    sig: 'n.has(pin): boolean',
    desc: "Whether this node's kind has that input pin (checks the spec, not whether a value is set).",
  },
  {
    label: 'n.get',
    type: 'method',
    sig: 'n.get(pin): value',
    desc: "Current effective value: this run's set value, else the literal, else the spec default; undefined if neither.",
  },
  {
    label: 'n.set',
    type: 'method',
    sig: 'n.set(pin, value)',
    desc: "Set a pin value. Rejected (and reported) if the pin doesn't exist or the value fails coercion.",
  },
  { label: 'n.kind', type: 'property', sig: 'n.kind: string', desc: 'Node kind, e.g. "Sleep".' },
  {
    label: 'n.sgID',
    type: 'property',
    sig: 'n.sgID: string | null',
    desc: 'null = main graph; otherwise the subgraph id.',
  },
  { label: 'n.id', type: 'property', sig: 'n.id: string', desc: 'Node UUID.' },
  { label: 'n.label', type: 'property', sig: 'n.label: string', desc: 'Node display label.' },
]

// hover: 悬停 `nodes` / `set` / `yt.nodes` / `n.set` 都能查到 (词可能含或不含前缀)。
export function ytHoverDoc(word: string): HoverDoc | null {
  const e = YT_ENTRIES.find(
    (x) => x.label === word || x.label === 'yt.' + word || x.label === 'n.' + word,
  )
  return e ? { sig: e.sig, desc: e.desc } : null
}

// ── 上下文感知补全 ──────────────────────────────────────────────────────────
// 扁平前缀匹配做不出 `yt.nodes.` → 数组方法 这种成员补全, 故给控制台一个按"点前文本"
// 分流的源: yt. → 顶层成员; yt.nodes./yt.selected. → 数组方法; 其它 X. → NodeHandle 成员。
const ARRAY_METHODS = [
  'filter',
  'forEach',
  'map',
  'find',
  'some',
  'every',
  'slice',
  'flatMap',
  'length',
]
const NODE_MEMBERS = ['has', 'get', 'set', 'kind', 'sgID', 'id', 'label']
const YT_TOP = ['nodes', 'selected', 'container', 'log']

export type YtCompletionKind = 'array' | 'yt' | 'node'

/** 据光标前文本判断该补哪类成员 (纯函数, 可单测)。null = 不补。 */
export function ytCompletionKind(before: string): YtCompletionKind | null {
  if (/(?:yt\.nodes|yt\.selected)\.\w*$/.test(before)) return 'array'
  if (/\byt\.\w*$/.test(before)) return 'yt'
  if (/[)\]\w]\.\w*$/.test(before)) return 'node'
  return null
}

const WORDS: Record<YtCompletionKind, { words: string[]; type: Completion['type'] }> = {
  array: { words: ARRAY_METHODS, type: 'method' },
  yt: { words: YT_TOP, type: 'property' },
  node: { words: NODE_MEMBERS, type: 'method' },
}

export function ytCompletionSource(ctx: CompletionContext): CompletionResult | null {
  const kind = ytCompletionKind(ctx.state.sliceDoc(0, ctx.pos))
  if (kind) {
    const w = ctx.matchBefore(/\w*$/) // 点后已敲的部分 (用于过滤)
    const { words, type } = WORDS[kind]
    return {
      from: w ? w.from : ctx.pos,
      options: words.map((label) => ({ label, type })),
      validFor: /^\w*$/,
    }
  }
  // 顶层显式触发 (Ctrl+Space) → 提示 yt。
  if (ctx.explicit) {
    const w = ctx.matchBefore(/[\w.]*$/)
    return {
      from: w ? w.from : ctx.pos,
      options: [{ label: 'yt', type: 'variable' }],
      validFor: /^[\w.]*$/,
    }
  }
  return null
}
