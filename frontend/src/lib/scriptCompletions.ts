// 脚本编辑器补全项: 节点函数 (来自 registry store) + 糖函数 (静态)。
// 节点签名从 Spec.inputs 推导: Kind({Pin1, Pin2, ...}) — 排除 exec pin。
// 末尾另放 CodeInput / CodeEditorModal 共用的 CodeMirror 扩展 (两组件互不 import, 避免环)。
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

// apply 上屏后光标落进末尾括号/引号内 (caretBack = 从串尾回退几格)。
function applyWithCaret(insert: string, caretBack: number) {
  return (v: EditorView, _c: Completion, from: number, to: number) => {
    v.dispatch({
      changes: { from, to, insert },
      selection: { anchor: from + insert.length - caretBack },
    })
  }
}

export const SUGAR_COMPLETIONS: Completion[] = [
  { label: 'vars.get', type: 'function', detail: 'vars.get(name, [scope])', apply: applyWithCaret('vars.get("")', 2) },
  { label: 'vars.set', type: 'function', detail: 'vars.set(name, value, [scope])', apply: applyWithCaret('vars.set("", )', 4) },
  { label: 'vars.inc', type: 'function', detail: 'vars.inc(name, delta, [scope])', apply: applyWithCaret('vars.inc("", 1)', 5) },
  { label: 'params.get', type: 'function', detail: 'params.get(name)', apply: applyWithCaret('params.get("")', 2) },
  { label: 'sleep', type: 'function', detail: 'sleep(ms)', apply: applyWithCaret('sleep()', 1) },
  { label: 'log.info', type: 'function', detail: 'log.info(...args)', apply: applyWithCaret('log.info()', 1) },
  { label: 'log.warn', type: 'function', detail: 'log.warn(...args)', apply: applyWithCaret('log.warn()', 1) },
  { label: 'log.debug', type: 'function', detail: 'log.debug(...args)', apply: applyWithCaret('log.debug()', 1) },
]

// labelOf: kind → i18n 人话名 (node.<Kind>.label), 拼在签名后; 纯函数保持可单测。
export function nodeFnCompletions(
  kinds: string[],
  specs: Map<string, Spec>,
  labelOf?: (kind: string) => string,
): Completion[] {
  return kinds.map((kind) => {
    const spec = specs.get(kind)
    const pins = (spec?.inputs ?? [])
      .filter((i) => i.type !== 'Exec')
      .map((i) => i.name)
      .join(', ')
    const label = labelOf?.(kind) ?? ''
    return {
      label: kind,
      type: 'function',
      detail: label ? `${kind}({${pins}}) · ${label}` : `${kind}({${pins}})`,
      apply: applyWithCaret(`${kind}({})`, 2),
    }
  })
}

// ── CodeInput / CodeEditorModal 共用的 CodeMirror 扩展 ──

// 词匹配含 "." — 让 "vars.g" 能补出 "vars.get" 这类带点糖函数。
function scriptCompletionSource(getOptions: () => Completion[]) {
  return (ctx: CompletionContext): CompletionResult | null => {
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
  placeholder?: string
  onChange?: (doc: string) => void
}): Extension[] {
  const exts: Extension[] = [
    history(),
    javascript(),
    syntaxHighlighting(scriptHighlight),
    autocompletion({ override: [scriptCompletionSource(opts.completions)] }),
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
