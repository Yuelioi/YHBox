// 脚本编辑器补全项: 节点函数 (来自 registry store) + 糖函数 (静态)。
// 节点签名从 Spec.inputs 推导: Kind({Pin1, Pin2, ...}) — 排除 exec pin。
// 末尾另放 CodeInput / EditorModal 共用的 CodeMirror 扩展 (组件间互不 import, 避免环)。
import {
  autocompletion,
  acceptCompletion,
  completionKeymap,
  type Completion,
  type CompletionContext,
  type CompletionResult,
} from '@codemirror/autocomplete'
import { EditorView, keymap, placeholder as cmPlaceholder } from '@codemirror/view'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { syntaxHighlighting, HighlightStyle } from '@codemirror/language'
import { javascript } from '@codemirror/lang-javascript'
import { tags } from '@lezer/highlight'
import type { Extension } from '@codemirror/state'
import type { Spec } from '@bindings/yotta/internal/node'
import { completionTooltipTheme } from '@/lib/editorTheme'

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
  /** 签名 (如 vars.get(name, [scope]) / ClickAt({XRatio, YRatio})) */
  detail?: string
  /** 人话说明 (节点中文名 / 函数 i18n desc) */
  desc?: string
  insert: string
  caretBack: number
}

function toCompletion(it: InsertItem): Completion {
  return {
    label: it.label,
    type: 'function',
    detail: it.desc ? `${it.detail} · ${it.desc}` : it.detail,
    apply: applyWithCaret(it.insert, it.caretBack),
  }
}

export const SUGAR_ITEMS: InsertItem[] = [
  { label: 'vars.get', detail: 'vars.get(name, [scope])', insert: 'vars.get("")', caretBack: 2 },
  { label: 'vars.set', detail: 'vars.set(name, value, [scope])', insert: 'vars.set("", )', caretBack: 4 },
  { label: 'vars.inc', detail: 'vars.inc(name, delta, [scope])', insert: 'vars.inc("", 1)', caretBack: 5 },
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
    const spec = specs.get(kind)
    const pins = (spec?.inputs ?? [])
      .filter((i) => i.type !== 'Exec')
      .map((i) => i.name)
      .join(', ')
    const label = labelOf?.(kind) ?? ''
    return {
      label: kind,
      detail: `${kind}({${pins}})`,
      desc: label || undefined,
      insert: `${kind}({})`,
      caretBack: 2,
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

// ── CodeInput / EditorModal 共用的 CodeMirror 扩展 ──

// vars.get("…")/set/inc 第一参字符串里 → 补容器变量名 (高频; 比手翻侧栏顺手)。
// 其余位置: 词匹配含 "." — 让 "vars.g" 能补出 "vars.get" 这类带点糖函数。
function scriptCompletionSource(getOptions: () => Completion[], varNames?: () => string[]) {
  return (ctx: CompletionContext): CompletionResult | null => {
    if (varNames) {
      const varCtx = ctx.matchBefore(/vars\.(get|set|inc)\(\s*"[A-Za-z0-9_]*/)
      if (varCtx) {
        const quote = varCtx.text.lastIndexOf('"')
        return {
          from: varCtx.from + quote + 1,
          validFor: /^[A-Za-z0-9_]*$/,
          options: varNames().map((n) => ({ label: n, type: 'variable' as const })),
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

// 配色对齐 ExprInput (pin 类型色: string 绿 / number 蓝 / bool 黄系)。
const scriptHighlight = HighlightStyle.define([
  { tag: tags.string, color: '#4ade80' },
  { tag: tags.number, color: '#60a5fa' },
  { tag: [tags.bool, tags.null], color: '#facc15' },
  { tag: tags.keyword, color: '#c084fc' },
  { tag: tags.comment, color: '#64748b', fontStyle: 'italic' },
  { tag: [tags.propertyName, tags.function(tags.variableName)], color: '#7dd3fc' },
  { tag: tags.variableName, color: '#e2e8f0' },
  { tag: tags.operator, color: '#94a3b8' },
])

// 语法错不在前端 lint (后端 validator SCRIPT_PARSE_ERROR 是权威), 故无 linter 扩展。
export function scriptEditorExtensions(opts: {
  completions: () => Completion[]
  /** 容器变量名 — vars.get("…")/set/inc 第一参字符串内的补全源。 */
  varNames?: () => string[]
  placeholder?: string
  onChange?: (doc: string) => void
}): Extension[] {
  const exts: Extension[] = [
    history(),
    javascript(),
    syntaxHighlighting(scriptHighlight),
    autocompletion({ override: [scriptCompletionSource(opts.completions, opts.varNames)] }),
    completionTooltipTheme,
    cmPlaceholder(opts.placeholder ?? ''),
    keymap.of([{ key: 'Tab', run: acceptCompletion }, ...completionKeymap, ...defaultKeymap, ...historyKeymap]),
    EditorView.lineWrapping,
  ]
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
