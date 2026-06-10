// Expr 表达式的 CodeMirror 扩展 (语言/补全/lint/hover) — 供小框 (ExprInput) 和
// 放大编辑 (EditorModal) 共用; 主题与编辑手感在 lib/editorTheme 的共享层。
// i18n 经回调注入, 本文件保持纯函数。
import {
  autocompletion,
  type CompletionContext,
  type CompletionResult,
} from '@codemirror/autocomplete'
import { EditorView, hoverTooltip, placeholder as cmPlaceholder } from '@codemirror/view'
import { StreamLanguage } from '@codemirror/language'
import { linter, type Diagnostic } from '@codemirror/lint'
import type { Extension } from '@codemirror/state'
import { allExprFunctions, exprFnNames, lintExpr, type ExprDiagnostic } from '@/lib/exprFunctions'
import { baseEditorExtensions, type BaseEditorOpts } from '@/lib/editorTheme'
import { fnHoverTooltip } from '@/lib/editorHover'

// ── 语法 token: 手写 stream tokenizer (语法就一行表达式, 不上 Lezer grammar) ──

const exprLanguage = StreamLanguage.define({
  token(stream) {
    if (stream.eatSpace()) return null
    if (stream.match(/^"(?:[^"\\]|\\.)*"?/)) return 'string'
    if (stream.match(/^\d+(\.\d+)?/)) return 'number'
    if (stream.match(/^\$[a-zA-Z_][a-zA-Z0-9_]*/)) return 'variableName.special' // $变量引用
    if (stream.match(/^[a-zA-Z_][a-zA-Z0-9_]*/)) {
      const word = stream.current()
      if (word === 'true' || word === 'false' || word === 'null') return 'atom'
      if (exprFnNames().has(word)) return 'variableName.function' // 内置函数名
      return 'variableName' // 动态输入引用
    }
    if (stream.match(/^[+\-*/%<>=!&|?:,.]+/)) return 'operator'
    if (stream.match(/^[()]/)) return 'bracket'
    stream.next()
    return null
  },
})

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
): { message: string; from: number } | null {
  const first = lintExpr(doc, exprFnNames())[0]
  return first ? { message: diagMessage(first), from: first.from } : null
}

export function exprEditorExtensions(opts: {
  fnDesc: (name: string) => string
  diagMessage: (d: ExprDiagnostic) => string
  inputNames?: () => string[]
  /** 容器变量名 — $ 补全源。 */
  varNames?: () => string[]
  placeholder?: string
  onChange?: (doc: string) => void
} & BaseEditorOpts): Extension[] {
  const lintSource = (v: EditorView): Diagnostic[] =>
    lintExpr(v.state.doc.toString(), exprFnNames()).map(d => ({
      from: d.from,
      to: Math.max(d.to, d.from),
      severity: 'error' as const,
      message: opts.diagMessage(d),
    }))

  const exts: Extension[] = [
    exprLanguage,
    autocompletion({ override: [exprCompletionSource(opts)] }),
    linter(lintSource, { delay: 300 }),
    hoverTooltip(fnHoverTooltip((word) => {
      const f = allExprFunctions().find((x) => x.name === word)
      return f ? { sig: f.sig, desc: opts.fnDesc(f.name) || undefined } : null
    })),
    cmPlaceholder(opts.placeholder ?? ''),
    ...baseEditorExtensions(opts),
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
