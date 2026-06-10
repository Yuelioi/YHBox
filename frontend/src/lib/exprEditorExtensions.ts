// Expr 表达式的 CodeMirror 扩展 (语言/高亮/补全/lint) — 从 ExprInput.vue 抽出,
// 供小框 (ExprInput) 和放大编辑 (EditorModal) 共用。i18n 经回调注入, 本文件保持纯函数。
import {
  autocompletion,
  acceptCompletion,
  completionKeymap,
  type CompletionContext,
  type CompletionResult,
} from '@codemirror/autocomplete'
import { EditorView, keymap, placeholder as cmPlaceholder } from '@codemirror/view'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { StreamLanguage, syntaxHighlighting, HighlightStyle } from '@codemirror/language'
import { linter, type Diagnostic } from '@codemirror/lint'
import { tags } from '@lezer/highlight'
import type { Extension } from '@codemirror/state'
import { allExprFunctions, exprFnNames, lintExpr, type ExprDiagnostic } from '@/lib/exprFunctions'
import { completionTooltipTheme } from '@/lib/editorTheme'

// ── 语法高亮: 手写 stream tokenizer (语法就一行表达式, 不上 Lezer grammar) ──

const exprLanguage = StreamLanguage.define({
  token(stream) {
    if (stream.eatSpace()) return null
    if (stream.match(/^"(?:[^"\\]|\\.)*"?/)) return 'string'
    if (stream.match(/^\d+(\.\d+)?/)) return 'number'
    if (stream.match(/^\$[a-zA-Z_][a-zA-Z0-9_]*/)) return 'variableName.special' // $变量引用
    if (stream.match(/^[a-zA-Z_][a-zA-Z0-9_]*/)) {
      const word = stream.current()
      if (word === 'true' || word === 'false' || word === 'null') return 'atom'
      if (exprFnNames().has(word)) return 'keyword' // 内置函数名
      return 'variableName' // 动态输入引用
    }
    if (stream.match(/^[+\-*/%<>=!&|?:,.]+/)) return 'operator'
    if (stream.match(/^[()]/)) return 'bracket'
    stream.next()
    return null
  },
})

// 配色对齐 pin 类型色 (types.go: string 绿 / number 蓝 / bool 黄系)。
const exprHighlight = HighlightStyle.define([
  { tag: tags.string, color: '#4ade80' },
  { tag: tags.number, color: '#60a5fa' },
  { tag: tags.atom, color: '#facc15' },
  { tag: tags.keyword, color: '#c084fc' },
  { tag: tags.special(tags.variableName), color: '#fb923c' }, // $变量引用 (橙)
  { tag: tags.variableName, color: '#e2e8f0' },
  { tag: tags.operator, color: '#94a3b8' },
])

// ── 补全: 函数 (sig + i18n 说明, 上屏光标落括号内) + 字面量 + 动态输入名 ──

function exprCompletionSource(opts: {
  fnDesc: (name: string) => string
  inputNames?: () => string[]
  varNames?: () => string[]
}) {
  return (ctx: CompletionContext): CompletionResult | null => {
    // $ 触发容器变量补全 (打一个 $ 就弹全列表)
    const dollar = ctx.matchBefore(/\$[a-zA-Z0-9_]*/)
    if (dollar && opts.varNames) {
      return {
        from: dollar.from,
        validFor: /^\$[a-zA-Z0-9_]*$/,
        options: opts.varNames().map((n) => ({ label: `$${n}`, type: 'variable' as const })),
      }
    }
    const word = ctx.matchBefore(/[a-zA-Z_][a-zA-Z0-9_]*/)
    if (!word && !ctx.explicit) return null
    const from = word ? word.from : ctx.pos
    return {
      from,
      validFor: /^[a-zA-Z0-9_]*$/,
      options: [
        ...allExprFunctions().map(f => ({
          label: f.name,
          displayLabel: f.sig,
          type: 'function',
          detail: opts.fnDesc(f.name),
          apply: (v: EditorView, _c: unknown, applyFrom: number, applyTo: number) => {
            const insert = `${f.name}()`
            const caret = applyFrom + insert.length - (f.maxArgs === 0 ? 0 : 1)
            v.dispatch({ changes: { from: applyFrom, to: applyTo, insert }, selection: { anchor: caret } })
          },
        })),
        ...['true', 'false', 'null'].map(l => ({ label: l, type: 'keyword' as const })),
        ...(opts.inputNames?.() ?? []).map(n => ({ label: n, type: 'variable' as const })),
      ],
    }
  }
}

// ── lint: 启发式诊断 → 红波浪线 + hover 提示 ──

export function firstExprError(
  doc: string,
  diagMessage: (d: ExprDiagnostic) => string,
): string {
  const first = lintExpr(doc, exprFnNames())[0]
  return first ? diagMessage(first) : ''
}

export function exprEditorExtensions(opts: {
  fnDesc: (name: string) => string
  diagMessage: (d: ExprDiagnostic) => string
  inputNames?: () => string[]
  /** 容器变量名 — $ 补全源。 */
  varNames?: () => string[]
  placeholder?: string
  onChange?: (doc: string) => void
}): Extension[] {
  const lintSource = (v: EditorView): Diagnostic[] =>
    lintExpr(v.state.doc.toString(), exprFnNames()).map(d => ({
      from: d.from,
      to: Math.max(d.to, d.from),
      severity: 'error' as const,
      message: opts.diagMessage(d),
    }))

  const exts: Extension[] = [
    history(),
    exprLanguage,
    syntaxHighlighting(exprHighlight),
    autocompletion({ override: [exprCompletionSource(opts)] }),
    completionTooltipTheme,
    linter(lintSource, { delay: 300 }),
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
