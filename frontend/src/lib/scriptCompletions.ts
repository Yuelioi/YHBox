// 脚本编辑器补全项: 节点函数 (来自 registry store) + 糖函数 (静态)。
// 节点签名从 Spec.inputs 推导: Kind({Pin1, Pin2, ...}) — 排除 exec pin。
// 末尾另放 CodeInput / EditorModal 共用的 CodeMirror 扩展 (组件间互不 import, 避免环)。
import {
  autocompletion,
  snippet,
  type Completion,
  type CompletionContext,
  type CompletionResult,
  type CompletionSource,
} from '@codemirror/autocomplete'
import {
  Decoration,
  type DecorationSet,
  EditorView,
  ViewPlugin,
  type ViewUpdate,
  hoverTooltip,
  placeholder as cmPlaceholder,
} from '@codemirror/view'
import { foldGutter, syntaxTree } from '@codemirror/language'
import { javascript } from '@codemirror/lang-javascript'
import { linter, type Diagnostic } from '@codemirror/lint'
import type { Extension, Range } from '@codemirror/state'
import { indentationMarkers } from '@replit/codemirror-indentation-markers'
import type { Spec } from '@bindings/github.com/yottaapp/yotta/internal/node'
import { baseEditorExtensions, type BaseEditorOpts } from '@/lib/editorTheme'
import { fnHoverTooltip, type HoverDoc } from '@/lib/editorHover'
import {
  scriptExitCompareContext,
  scriptPinValueContext,
  scriptSigContext,
  signatureHelp,
} from '@/lib/editorSignature'

// apply 上屏后光标落进末尾括号/引号内 (caretBack = 从串尾回退几格)。
function applyWithCaret(insert: string, caretBack: number) {
  return (v: EditorView, _c: Completion, from: number, to: number) => {
    v.dispatch({
      changes: { from, to, insert },
      selection: { anchor: from + insert.length - caretBack },
    })
  }
}

// 可插入项的原始数据 — 补全下拉 (toCompletion) 和放大编辑 modal 的参考面板共用一份单源。
export interface InsertItem {
  label: string
  type?: Completion['type']
  /** 签名 (如 params.get(name) / ClickAt({XRatio, YRatio})) */
  detail?: string
  /** 人话说明 (节点中文名 / 函数 i18n desc) */
  desc?: string
  insert: string
  caretBack: number
  /** CodeMirror snippet 模板 (`${Pin}` 占位 Tab 跳格) — 有则优先于 insert/caretBack。 */
  snippet?: string
}

export interface ScriptPinValueItem {
  value: string
  label?: string
  detail?: string
  type?: 'enum' | 'variable' | 'asset'
  insertMode?: 'string' | 'template'
}

export type ScriptTemplateInsertMode = 'bare' | 'array' | 'string'

export interface ScriptTemplateSummary {
  guid: string
  kind: string
  name: string
  category?: string
  tags?: string[]
  variantCount?: number
  createdAt?: string
}

export type ScriptAssetSummary = ScriptTemplateSummary

export interface ScriptAIConnectionSummary {
  id: string
  label: string
  protocol?: string
  baseURL?: string
}

export interface ScriptPointPayload {
  xRatio: number
  yRatio: number
}

export interface ScriptRectPayload {
  region: [number, number, number, number]
}

export interface ScriptColorPayload {
  range: number[]
  hueWrap?: boolean
}

export interface ScriptColorPickTarget {
  colorSpace: 'hsv' | 'rgb'
  shape: 'tuple' | 'object'
}

export interface ScriptAsyncDropdownTarget {
  asyncSource: string
}

function toCompletion(it: InsertItem): Completion {
  return {
    label: it.label,
    type: it.type ?? 'function',
    detail: it.desc ? `${it.detail} · ${it.desc}` : it.detail,
    apply: it.snippet ? snippet(it.snippet) : applyWithCaret(it.insert, it.caretBack),
  }
}

// 变量读写不再走 vars.* 糖 — 用 $hp (读 live 值) 或 GetVar/SetVar/IncVar 节点函数
// (VarName/Scope pin 值位有补全)。糖只留没有节点替身或 $ 捷径的高频项。
// Subgraph 不是节点自动绑定 (RegionRunner 被排除), 是绑定层定制函数, 故在此登记。
export const SUGAR_ITEMS: InsertItem[] = [
  {
    label: 'Subgraph',
    detail: 'Subgraph({SubgraphID, ...params})',
    insert: 'Subgraph({SubgraphID: ""})',
    caretBack: 3,
    snippet: 'Subgraph({SubgraphID: "${SubgraphID}"})',
  },
  { label: 'params.get', detail: 'params.get(name)', insert: 'params.get("")', caretBack: 2 },
  { label: 'sleep', detail: 'sleep(ms)', insert: 'sleep()', caretBack: 1 },
  { label: 'log.info', detail: 'log.info(...args)', insert: 'log.info()', caretBack: 1 },
  { label: 'log.warn', detail: 'log.warn(...args)', insert: 'log.warn()', caretBack: 1 },
  { label: 'log.debug', detail: 'log.debug(...args)', insert: 'log.debug()', caretBack: 1 },
  { label: 'Exit.Done', type: 'constant', detail: '"Done"', insert: 'Exit.Done', caretBack: 0 },
  { label: 'Exit.Fail', type: 'constant', detail: '"Fail"', insert: 'Exit.Fail', caretBack: 0 },
  { label: 'Exit.Found', type: 'constant', detail: '"Found"', insert: 'Exit.Found', caretBack: 0 },
  {
    label: 'Exit.NotFound',
    type: 'constant',
    detail: '"NotFound"',
    insert: 'Exit.NotFound',
    caretBack: 0,
  },
  {
    label: 'Exit.Timeout',
    type: 'constant',
    detail: '"Timeout"',
    insert: 'Exit.Timeout',
    caretBack: 0,
  },
  { label: 'Exit.True', type: 'constant', detail: '"True"', insert: 'Exit.True', caretBack: 0 },
  { label: 'Exit.False', type: 'constant', detail: '"False"', insert: 'Exit.False', caretBack: 0 },
  {
    label: 'Exit.Default',
    type: 'constant',
    detail: '"default"',
    insert: 'Exit.Default',
    caretBack: 0,
  },
  { label: 'Exit.Body', type: 'constant', detail: '"Body"', insert: 'Exit.Body', caretBack: 0 },
  {
    label: 'Exit.Changed',
    type: 'constant',
    detail: '"Changed"',
    insert: 'Exit.Changed',
    caretBack: 0,
  },
  { label: 'Exit.Gone', type: 'constant', detail: '"Gone"', insert: 'Exit.Gone', caretBack: 0 },
  { label: 'Exit.Out', type: 'constant', detail: '"Out"', insert: 'Exit.Out', caretBack: 0 },
  {
    label: 'Exit.Stable',
    type: 'constant',
    detail: '"Stable"',
    insert: 'Exit.Stable',
    caretBack: 0,
  },
]

export const SUGAR_COMPLETIONS: Completion[] = SUGAR_ITEMS.map(toCompletion)

const EXIT_CONSTANT_BY_NAME = new Map<string, string>([
  ['Body', 'Body'],
  ['Changed', 'Changed'],
  ['default', 'Default'],
  ['Done', 'Done'],
  ['Fail', 'Fail'],
  ['False', 'False'],
  ['Found', 'Found'],
  ['Gone', 'Gone'],
  ['NotFound', 'NotFound'],
  ['Out', 'Out'],
  ['Stable', 'Stable'],
  ['Timeout', 'Timeout'],
  ['True', 'True'],
])

export function scriptExitItemsForKind(kind: string, specs: Map<string, Spec>): InsertItem[] {
  const outputs = (specs.get(kind)?.outputs ?? []).filter((o) => o.type === 'Exec')
  return outputs.map((o) => {
    const constant = EXIT_CONSTANT_BY_NAME.get(o.name)
    if (constant) {
      return {
        label: `Exit.${constant}`,
        type: 'constant',
        detail: JSON.stringify(o.name),
        insert: `Exit.${constant}`,
        caretBack: 0,
      }
    }
    return {
      label: JSON.stringify(o.name),
      type: 'constant',
      detail: `${kind}.${o.name}`,
      insert: JSON.stringify(o.name),
      caretBack: 0,
    }
  })
}

export function scriptTemplateItemsForPin(
  kind: string,
  pin: string,
  specs: Map<string, Spec>,
  templates: ScriptTemplateSummary[],
): ScriptPinValueItem[] {
  const input = specs.get(kind)?.inputs?.find((i) => i.name === pin)
  if (input?.semantic !== 'TemplateGUID' && input?.widget?.kind !== 'template-picker') return []
  return templates
    .filter((t) => t.kind === 'template')
    .slice()
    .sort((a, b) => (a.name || a.guid).localeCompare(b.name || b.guid))
    .map((t) => {
      const meta = [t.category, (t.tags ?? []).join(', '), t.guid].filter(Boolean).join(' · ')
      return {
        value: t.guid,
        label: t.name || t.guid,
        detail: meta || undefined,
        type: 'enum' as const,
        insertMode: 'template' as const,
      }
    })
}

export function scriptAssetItemsForPin(
  kind: string,
  pin: string,
  specs: Map<string, Spec>,
  assets: ScriptAssetSummary[],
): ScriptPinValueItem[] {
  const input = specs.get(kind)?.inputs?.find((i) => i.name === pin)
  if (input?.semantic !== 'ClipID') return []
  return assets
    .filter((a) => a.kind === 'clip')
    .slice()
    .sort((a, b) => (a.name || a.guid).localeCompare(b.name || b.guid))
    .map((a) => {
      const meta = [a.category, (a.tags ?? []).join(', '), a.guid].filter(Boolean).join(' · ')
      return {
        value: a.guid,
        label: a.name || a.guid,
        detail: meta || undefined,
        type: 'asset' as const,
        insertMode: 'string' as const,
      }
    })
}

export function scriptAIConnectionItemsForPin(
  kind: string,
  pin: string,
  specs: Map<string, Spec>,
  connections: ScriptAIConnectionSummary[],
): ScriptPinValueItem[] {
  const input = specs.get(kind)?.inputs?.find((i) => i.name === pin)
  if (input?.widget?.kind !== 'ai-connection') return []
  return connections
    .slice()
    .sort((a, b) => (a.label || a.id).localeCompare(b.label || b.id))
    .map((c) => {
      const meta = [c.protocol, c.baseURL, c.id].filter(Boolean).join(' · ')
      return {
        value: c.id,
        label: c.label || c.id,
        detail: meta || undefined,
        type: 'enum' as const,
        insertMode: 'string' as const,
      }
    })
}

export function scriptAsyncDropdownTargetForPin(
  kind: string,
  pin: string,
  specs: Map<string, Spec>,
): ScriptAsyncDropdownTarget | null {
  const input = specs.get(kind)?.inputs?.find((i) => i.name === pin)
  const props = input?.widget?.props as Record<string, unknown> | undefined
  const asyncSource = props?.asyncSource
  if (input?.widget?.kind !== 'async-dropdown' || typeof asyncSource !== 'string' || !asyncSource) {
    return null
  }
  return { asyncSource }
}

export function scriptCurrentCallInputSnapshot(doc: string, pos: number): Record<string, unknown> {
  const bounds = objectBoundsAround(doc, pos)
  if (!bounds) return {}
  const inner = doc.slice(bounds.from + 1, bounds.to)
  const out: Record<string, unknown> = {}
  for (const part of splitTopLevel(inner)) {
    const colon = indexOfTopLevelColon(part)
    if (colon < 0) continue
    const key = parseObjectKey(part.slice(0, colon).trim())
    if (!key) continue
    const value = parseSimpleScriptLiteral(part.slice(colon + 1).trim())
    if (value !== undefined) out[key] = value
  }
  return out
}

function objectBoundsAround(doc: string, pos: number): { from: number; to: number } | null {
  const stack: number[] = []
  for (let i = 0; i < doc.length; i++) {
    const ch = doc[i]
    if (ch === '"' || ch === "'" || ch === '`') {
      i = skipQuoted(doc, i)
      continue
    }
    if (ch === '{') {
      stack.push(i)
      continue
    }
    if (ch !== '}') continue
    const from = stack.pop()
    if (from == null) continue
    if (from < pos && pos <= i) return { from, to: i }
  }
  return null
}

function skipQuoted(text: string, start: number): number {
  const quote = text[start]
  let i = start + 1
  while (i < text.length) {
    if (text[i] === '\\') {
      i += 2
      continue
    }
    if (text[i] === quote) return i
    i++
  }
  return text.length - 1
}

function splitTopLevel(text: string): string[] {
  const parts: string[] = []
  let start = 0
  let depth = 0
  for (let i = 0; i < text.length; i++) {
    const ch = text[i]
    if (ch === '"' || ch === "'" || ch === '`') {
      i = skipQuoted(text, i)
      continue
    }
    if (ch === '{' || ch === '[' || ch === '(') depth++
    else if (ch === '}' || ch === ']' || ch === ')') depth = Math.max(0, depth - 1)
    else if (ch === ',' && depth === 0) {
      parts.push(text.slice(start, i))
      start = i + 1
    }
  }
  parts.push(text.slice(start))
  return parts
}

function indexOfTopLevelColon(text: string): number {
  let depth = 0
  for (let i = 0; i < text.length; i++) {
    const ch = text[i]
    if (ch === '"' || ch === "'" || ch === '`') {
      i = skipQuoted(text, i)
      continue
    }
    if (ch === '{' || ch === '[' || ch === '(') depth++
    else if (ch === '}' || ch === ']' || ch === ')') depth = Math.max(0, depth - 1)
    else if (ch === ':' && depth === 0) return i
  }
  return -1
}

function parseObjectKey(text: string): string | null {
  if (/^[A-Za-z_$][\w$]*$/.test(text)) return text
  const literal = parseSimpleStringLiteral(text)
  return typeof literal === 'string' ? literal : null
}

function parseSimpleScriptLiteral(text: string): unknown {
  if (text === 'true') return true
  if (text === 'false') return false
  if (text === 'null') return null
  if (/^-?(?:\d+|\d*\.\d+)$/.test(text)) return Number(text)
  return parseSimpleStringLiteral(text)
}

function parseSimpleStringLiteral(text: string): string | undefined {
  if (text.length < 2) return undefined
  const quote = text[0]
  if ((quote !== '"' && quote !== "'") || text[text.length - 1] !== quote) return undefined
  const body = text.slice(1, -1)
  if (quote === '"') {
    try {
      return JSON.parse(text)
    } catch {
      return body
    }
  }
  return body.replace(/\\'/g, "'").replace(/\\\\/g, '\\')
}

export function scriptTemplateInsertText(guid: string, mode: ScriptTemplateInsertMode): string {
  if (mode === 'string') return guid
  const quoted = JSON.stringify(guid)
  return mode === 'array' ? quoted : `[${quoted}]`
}

export function scriptPinValueInsertText(
  item: ScriptPinValueItem,
  mode: ScriptTemplateInsertMode,
  inString: boolean,
): string {
  if (item.insertMode === 'template') return scriptTemplateInsertText(item.value, mode)
  return inString ? item.value : JSON.stringify(item.value)
}

function round4(n: number): number {
  return Math.round(n * 1e4) / 1e4
}

export function scriptPointInsertText(payload: ScriptPointPayload): string {
  return `{ x: ${round4(payload.xRatio)}, y: ${round4(payload.yRatio)} }`
}

export function scriptGeometryInsertText(payload: ScriptRectPayload): string {
  const [x, y, w, h] = payload.region
  return `{ pct: { x: ${round4(x)}, y: ${round4(y)}, w: ${round4(w)}, h: ${round4(h)} } }`
}

export function scriptColorInsertText(
  payload: ScriptColorPayload,
  shape: ScriptColorPickTarget['shape'],
): string {
  const range = payload.range.slice(0, 6).map((n) => Math.round(Number(n) || 0))
  while (range.length < 6) range.push(0)
  if (shape === 'tuple') return `[${range.join(', ')}]`
  return `{ hMin: ${range[0]}, hMax: ${range[1]}, sMin: ${range[2]}, sMax: ${range[3]}, vMin: ${range[4]}, vMax: ${range[5]} }`
}

export function scriptStringInsertText(value: string, doc: string, pos: number): string {
  return scriptTemplateInsertMode(doc, pos) === 'string' ? value : JSON.stringify(value)
}

export function scriptColorPickTargetForPin(
  kind: string,
  pin: string,
  specs: Map<string, Spec>,
  doc: string,
  pos: number,
): ScriptColorPickTarget | null {
  const input = specs.get(kind)?.inputs?.find((i) => i.name === pin)
  const schema = input?.schema as { type?: string; widget?: string } | undefined
  if (schema?.widget !== 'colorRange') return null
  if (schema.type === 'tuple') {
    return { colorSpace: scriptColorModeForCall(kind, doc, pos), shape: 'tuple' }
  }
  if (schema.type === 'object') return { colorSpace: 'hsv', shape: 'object' }
  return null
}

function scriptColorModeForCall(kind: string, doc: string, pos: number): 'hsv' | 'rgb' {
  const before = doc.slice(0, pos)
  const callAt = before.lastIndexOf(`${kind}(`)
  if (callAt < 0) return 'hsv'
  const callText = before.slice(callAt)
  const mode = /\bMode\s*:\s*["'](rgb|hsv)["']/.exec(callText)?.[1]
  return mode === 'rgb' ? 'rgb' : 'hsv'
}

export function scriptScreenPickKindForPin(
  kind: string,
  pin: string,
  specs: Map<string, Spec>,
): 'point' | 'rect' | null {
  const input = specs.get(kind)?.inputs?.find((i) => i.name === pin)
  if (input?.type === 'Point') return 'point'
  if (input?.type === 'Geometry' || input?.type === 'Rect') return 'rect'
  return null
}

export function scriptTemplateInsertMode(doc: string, pos: number): ScriptTemplateInsertMode {
  const before = doc.slice(0, pos)
  const quoteCount = (before.match(/(?<!\\)"/g) ?? []).length
  if (quoteCount % 2 === 1) return 'string'

  const stack: string[] = []
  for (let i = 0; i < before.length; i++) {
    const ch = before[i]
    if (ch === '"' || ch === "'" || ch === '`') {
      i++
      while (i < before.length) {
        if (before[i] === '\\') {
          i += 2
          continue
        }
        if (before[i] === ch) break
        i++
      }
      continue
    }
    if (ch === '[' || ch === '{' || ch === '(') stack.push(ch)
    else if (ch === ']' || ch === '}' || ch === ')') stack.pop()
  }
  return stack[stack.length - 1] === '[' ? 'array' : 'bare'
}

// labelOf: kind → i18n 人话名 (node.<Kind>.label); 纯函数保持可单测。
export function nodeFnItems(
  kinds: string[],
  specs: Map<string, Spec>,
  labelOf?: (kind: string) => string,
): InsertItem[] {
  return kinds.map((kind) => {
    const dataPins = (specs.get(kind)?.inputs ?? []).filter((i) => i.type !== 'Exec')
    const pins = dataPins.map((i) => i.name).join(', ')
    // snippet 占位只铺非 advanced pin (捕获框/高级项省略, 走默认值) — Tab 逐个跳格填值。
    const fields = dataPins.filter((i) => !i.advanced).map((i) => i.name)
    const label = labelOf?.(kind) ?? ''
    return {
      label: kind,
      detail: `${kind}({${pins}})`,
      desc: label || undefined,
      insert: fields.length ? `${kind}({})` : `${kind}()`,
      caretBack: fields.length ? 2 : 0,
      snippet: fields.length
        ? `${kind}({${fields.map((f) => `${f}: \${${f}}`).join(', ')}})`
        : undefined,
    }
  })
}

export function nodeFnCompletions(
  kinds: string[],
  specs: Map<string, Spec>,
  labelOf?: (kind: string) => string,
): Completion[] {
  return nodeFnItems(kinds, specs, labelOf).map(toCompletion)
}

// ── 语法快速反馈 (纯函数, 可单测): lezer 容错解析的 error 节点 → 行级诊断。
//    权威仍是后端 validator (SCRIPT_PARSE_ERROR), 这里只是打字时的即时提示。 ──

const jsParser = javascript().language.parser

export interface ScriptSyntaxError {
  from: number
  to: number
  line: number
}

export function scriptSyntaxErrors(doc: string): ScriptSyntaxError[] {
  const out: ScriptSyntaxError[] = []
  const seenLines = new Set<number>()
  jsParser.parse(doc).iterate({
    enter: (n) => {
      if (!n.type.isError) return
      const line = lineOf(doc, n.from)
      if (seenLines.has(line)) return
      seenLines.add(line)
      out.push({ from: n.from, to: Math.max(n.to, n.from), line })
    },
  })
  return out.slice(0, 10)
}

// $变量引用提取 (纯函数, 可单测): 走语法树只取 VariableName 节点,
// 字符串/注释里的 $ 不会命中; 本地 `let $x` 定义记入 defined 免误报。
export interface DollarRefs {
  refs: { name: string; from: number; to: number }[]
  defined: Set<string>
}

export function scriptDollarRefs(doc: string): DollarRefs {
  const refs: DollarRefs['refs'] = []
  const defined = new Set<string>()
  jsParser.parse(doc).iterate({
    enter: (n) => {
      if (n.name !== 'VariableName' && n.name !== 'VariableDefinition') return
      const text = doc.slice(n.from, n.to)
      if (!text.startsWith('$') || text.length < 2) return
      if (n.name === 'VariableDefinition') defined.add(text.slice(1))
      else refs.push({ name: text.slice(1), from: n.from, to: n.to })
    },
  })
  return { refs, defined }
}

function lineOf(doc: string, pos: number): number {
  let line = 1
  for (let i = 0; i < pos && i < doc.length; i++) if (doc[i] === '\n') line++
  return line
}

// ── $变量徽标: 视口内 VariableName 且以 $ 开头 → 橙色 mark (样式在 editorTheme) ──

const dollarMark = Decoration.mark({ class: 'cm-yh-dollar' })

const dollarDecorations = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet
    constructor(view: EditorView) {
      this.decorations = buildDollarDecorations(view)
    }
    update(u: ViewUpdate) {
      if (u.docChanged || u.viewportChanged) this.decorations = buildDollarDecorations(u.view)
    }
  },
  { decorations: (v) => v.decorations },
)

function buildDollarDecorations(view: EditorView): DecorationSet {
  const marks: Range<Decoration>[] = []
  for (const { from, to } of view.visibleRanges) {
    syntaxTree(view.state).iterate({
      from,
      to,
      enter: (n) => {
        if (n.name !== 'VariableName') return
        if (view.state.sliceDoc(n.from, n.from + 1) === '$' && n.to > n.from + 1) {
          marks.push(dollarMark.range(n.from, n.to))
        }
      },
    })
  }
  return Decoration.set(marks)
}

// ── CodeInput / EditorModal 共用的 CodeMirror 扩展 ──

// pin 值位置 (`Kind({Pin: ▮})`): 枚举 pin → 候选值, varname pin → 容器变量名。
// 其余位置: 词匹配含 "." — 让 "log.i" 能补出 "log.info" 这类带点糖函数。
function scriptCompletionSource(
  getOptions: () => Completion[],
  pinValues?: (kind: string, pin: string) => ScriptPinValueItem[],
  exitValues?: (kind: string) => InsertItem[],
) {
  return (ctx: CompletionContext): CompletionResult | null => {
    if (exitValues) {
      const ec = scriptExitCompareContext(ctx.state.doc.toString(), ctx.pos)
      if (ec) {
        const options = exitValues(ec.kind).map(toCompletion)
        if (options.length) {
          return {
            from: ec.from,
            validFor: /^[\w$.]*$/,
            options,
          }
        }
      }
    }
    if (pinValues) {
      const pv = scriptPinValueContext(ctx.state.doc.toString(), ctx.pos)
      if (pv) {
        const opts = pinValues(pv.kind, pv.pin)
        if (opts.length) {
          const inStr = ctx.matchBefore(/"[^"\n\r]*$/)
          const mode = scriptTemplateInsertMode(ctx.state.doc.toString(), ctx.pos)
          return {
            from: inStr ? inStr.from + 1 : ctx.pos,
            validFor: /^[^"',\]}]*$/,
            options: opts.map((o) => ({
              label: o.label ?? o.value,
              type: o.type ?? 'enum',
              detail: o.detail ?? o.value,
              // 串内: 只补值 (引号已在); 裸值位: 按 pin 语义决定字符串/模板数组。
              apply: scriptPinValueInsertText(o, mode, !!inStr),
            })),
          }
        }
      }
    }
    const word = ctx.matchBefore(/[A-Za-z_$][\w$.]*/)
    if (!word && !ctx.explicit) return null
    return {
      from: word ? word.from : ctx.pos,
      validFor: /^[\w$.]*$/,
      options: getOptions(),
    }
  }
}

export function scriptEditorExtensions(
  opts: {
    /** 扁平补全项 (按词前缀过滤); 传了 completionSource 则忽略本项。 */
    completions?: () => Completion[]
    /** 自定义补全源 (上下文感知, 如 yt 控制台的成员补全) — 传了则**取代** completions 的默认源。 */
    completionSource?: CompletionSource
    /** 容器变量名 — $引用未声明提醒 (lint)。 */
    varNames?: () => string[]
    /** pin 值候选: (kind, pin) → 选项; 用于 `Kind({Pin: ▮})` 值位置补全 (枚举值 / 变量名)。缺省不补。 */
    pinValues?: (kind: string, pin: string) => ScriptPinValueItem[]
    /** exit 比较候选: 光标位于 `r.exit === ▮` 时, 根据 `r = Kind(...)` 反查 kind 后提供出口。 */
    exitValues?: (kind: string) => InsertItem[]
    /** 悬停函数名的文档数据 (节点/糖函数), 缺省不出 hover。 */
    hoverDoc?: (word: string) => HoverDoc | null
    /** 函数签名查找 (节点/糖函数), 缺省不出 signature help。 */
    signatureLookup?: (name: string) => { sig: string } | null
    /** lint 文案 (i18n 注入): 语法错 / 未声明 $变量。缺省不挂 linter。 */
    lintMessages?: {
      syntaxError: (line: number) => string
      unknownVar: (name: string) => string
    }
    placeholder?: string
    onChange?: (doc: string) => void
  } & BaseEditorOpts,
): Extension[] {
  const exts: Extension[] = [
    javascript(),
    dollarDecorations,
    indentationMarkers({
      highlightActiveBlock: true,
      colors: { dark: '#404040', activeDark: '#6a6a6a', light: '#404040', activeLight: '#6a6a6a' },
    }),
    autocompletion({
      override: [
        opts.completionSource ??
          scriptCompletionSource(opts.completions ?? (() => []), opts.pinValues, opts.exitValues),
      ],
    }),
    cmPlaceholder(opts.placeholder ?? ''),
    ...baseEditorExtensions(opts),
    // 在共享层之后追加 → 折叠 gutter 落在行号右侧 (VSCode 布局)
    ...(opts.modal ? [foldGutter()] : []),
  ]
  if (opts.hoverDoc) {
    exts.push(hoverTooltip(fnHoverTooltip(opts.hoverDoc)))
  }
  if (opts.signatureLookup) {
    exts.push(
      signatureHelp({
        context: (s, p) => scriptSigContext(s.doc.toString(), p),
        lookup: opts.signatureLookup,
      }),
    )
  }
  const lintMessages = opts.lintMessages
  if (lintMessages) {
    exts.push(
      linter(
        (v) => {
          const doc = v.state.doc.toString()
          const diags: Diagnostic[] = scriptSyntaxErrors(doc).map((e) => ({
            from: e.from,
            to: e.to,
            severity: 'error' as const,
            message: lintMessages.syntaxError(e.line),
          }))
          if (opts.varNames) {
            const known = new Set(opts.varNames())
            const { refs, defined } = scriptDollarRefs(doc)
            for (const r of refs) {
              if (known.has(r.name) || defined.has(r.name)) continue
              diags.push({
                from: r.from,
                to: r.to,
                severity: 'warning' as const,
                message: lintMessages.unknownVar(r.name),
              })
            }
          }
          return diags
        },
        { delay: 300 },
      ),
    )
  }
  const onChange = opts.onChange
  if (onChange) {
    exts.push(
      EditorView.updateListener.of((u) => {
        if (u.docChanged) onChange(u.state.doc.toString())
      }),
    )
  }
  return exts
}
