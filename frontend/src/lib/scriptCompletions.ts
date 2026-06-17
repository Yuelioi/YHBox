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
import type { Spec } from '@bindings/yotta/internal/node'
import { baseEditorExtensions, type BaseEditorOpts } from '@/lib/editorTheme'
import { fnHoverTooltip, type HoverDoc } from '@/lib/editorHover'
import { scriptPinValueContext, scriptSigContext, signatureHelp } from '@/lib/editorSignature'

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
  /** 签名 (如 params.get(name) / ClickAt({XRatio, YRatio})) */
  detail?: string
  /** 人话说明 (节点中文名 / 函数 i18n desc) */
  desc?: string
  insert: string
  caretBack: number
  /** CodeMirror snippet 模板 (`${Pin}` 占位 Tab 跳格) — 有则优先于 insert/caretBack。 */
  snippet?: string
}

function toCompletion(it: InsertItem): Completion {
  return {
    label: it.label,
    type: 'function',
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
]

export const SUGAR_COMPLETIONS: Completion[] = SUGAR_ITEMS.map(toCompletion)

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
  pinValues?: (kind: string, pin: string) => { value: string; label?: string; type?: 'enum' | 'variable' }[],
) {
  return (ctx: CompletionContext): CompletionResult | null => {
    if (pinValues) {
      const pv = scriptPinValueContext(ctx.state.doc.toString(), ctx.pos)
      if (pv) {
        const opts = pinValues(pv.kind, pv.pin)
        if (opts.length) {
          const inStr = ctx.matchBefore(/"[A-Za-z0-9_]*$/)
          return {
            from: inStr ? inStr.from + 1 : ctx.pos,
            validFor: /^[A-Za-z0-9_]*$/,
            options: opts.map((o) => ({
              label: o.value,
              type: o.type ?? 'enum',
              detail: o.label,
              // 串内: 只补值 (引号已在); 裸值位: 自带引号上屏。
              apply: inStr ? o.value : `"${o.value}"`,
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

export function scriptEditorExtensions(opts: {
  /** 扁平补全项 (按词前缀过滤); 传了 completionSource 则忽略本项。 */
  completions?: () => Completion[]
  /** 自定义补全源 (上下文感知, 如 yt 控制台的成员补全) — 传了则**取代** completions 的默认源。 */
  completionSource?: CompletionSource
  /** 容器变量名 — $引用未声明提醒 (lint)。 */
  varNames?: () => string[]
  /** pin 值候选: (kind, pin) → 选项; 用于 `Kind({Pin: ▮})` 值位置补全 (枚举值 / 变量名)。缺省不补。 */
  pinValues?: (kind: string, pin: string) => { value: string; label?: string; type?: 'enum' | 'variable' }[]
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
} & BaseEditorOpts): Extension[] {
  const exts: Extension[] = [
    javascript(),
    dollarDecorations,
    indentationMarkers({
      highlightActiveBlock: true,
      colors: { dark: '#404040', activeDark: '#6a6a6a', light: '#404040', activeLight: '#6a6a6a' },
    }),
    autocompletion({
      override: [opts.completionSource ?? scriptCompletionSource(opts.completions ?? (() => []), opts.pinValues)],
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
