// Expr 内置函数元数据 — ExprInput 补全下拉 + 即时未知函数检查用.
// 单一来源是 Go internal/services/expr/builtins.go, 启动时经 NodeService.GetExprFunctions
// RPC 拉取 (populateRegistryFromBackend 后置 setExprFunctions 填). 本模块不再手写函数表 —
// 加函数 = 改 Go 表 (含 Sig) + zh/en 各 1 条 expression.fn.<name>.desc.

export interface ExprFunction {
  name: string
  /** 签名展示串, 参数名用通用英文 (跨 locale 不翻译). */
  sig: string
  minArgs: number
  maxArgs: number
}

let exprFunctions: ExprFunction[] = []
let fnNames = new Set<string>()

/** 启动时由 populateRegistryFromBackend 填; 测试可直接调. */
export function setExprFunctions(list: ExprFunction[]) {
  exprFunctions = list
  fnNames = new Set(list.map(f => f.name))
}

export function allExprFunctions(): ExprFunction[] {
  return exprFunctions
}

export function exprFnNames(): Set<string> {
  return fnNames
}

/** 从签名串取参数名 (snippet 占位 / signature help 共用). 'clamp(x, lo, hi)' → ['x','lo','hi']. */
export function parseSigParams(sig: string): string[] {
  const open = sig.indexOf('(')
  const close = sig.lastIndexOf(')')
  if (open < 0 || close <= open) return []
  const inner = sig.slice(open + 1, close).trim()
  if (!inner) return []
  return inner.split(',').map((s) => s.trim().replace(/^\[|\]$/g, '').trim())
}

/** 光标处 token (往前扫 identifier 字符). 补全替换区间 = [start, caret). */
export function tokenAtCaret(text: string, caret: number): { token: string; start: number } {
  const end = Math.min(Math.max(caret, 0), text.length)
  let i = end
  while (i > 0 && /[a-zA-Z0-9_]/.test(text[i - 1])) i--
  return { token: text.slice(i, end), start: i }
}

/** 扫描"长得像函数调用"的 token (identifier 后跟 '('), 带位置. 跳过双引号字符串 (含 \" 转义). */
function scanCalls(text: string): { name: string; from: number; to: number }[] {
  const out: { name: string; from: number; to: number }[] = []
  let i = 0
  while (i < text.length) {
    const c = text[i]
    if (c === '"') {
      i++
      while (i < text.length && !(text[i] === '"' && text[i - 1] !== '\\')) i++
      i++
      continue
    }
    if (/[a-zA-Z_]/.test(c)) {
      let j = i + 1
      while (j < text.length && /[a-zA-Z0-9_]/.test(text[j])) j++
      let k = j
      while (k < text.length && text[k] === ' ') k++
      if (text[k] === '(') out.push({ name: text.slice(i, j), from: i, to: j })
      i = j
      continue
    }
    i++
  }
  return out
}

/** 文本里调用了但不在 known 集合的函数名 (去重). */
export function unknownFnsIn(text: string, known: Set<string>): string[] {
  const out: string[] = []
  const seen = new Set<string>()
  for (const call of scanCalls(text)) {
    if (!known.has(call.name) && !seen.has(call.name)) {
      seen.add(call.name)
      out.push(call.name)
    }
  }
  return out
}

/** 带位置的启发式诊断 — CodeMirror lint 红线 + 框下红字共用. messageKey 是 i18n key. */
export interface ExprDiagnostic {
  from: number
  to: number
  messageKey: string
  params?: Record<string, unknown>
}

// lintExpr — 括号/引号错误互为级联噪音, 命中即短路只报一条; 其余 (裸词/尾运算符/未知函数) 可并存全收.
export function lintExpr(text: string, known: Set<string>): ExprDiagnostic[] {
  if (text.trim() === '') return []

  let depth = 0
  let inStr = false
  let strStart = -1
  for (let i = 0; i < text.length; i++) {
    const c = text[i]
    if (c === '"' && text[i - 1] !== '\\') {
      if (!inStr) strStart = i
      inStr = !inStr
      continue
    }
    if (inStr) continue
    if (c === '(') depth++
    else if (c === ')') {
      depth--
      if (depth < 0) return [{ from: i, to: i + 1, messageKey: 'expression.error.paren_mismatch' }]
    }
  }
  if (inStr) return [{ from: strStart, to: text.length, messageKey: 'expression.error.string_unclosed' }]
  if (depth > 0) {
    return [{
      from: text.length, to: text.length,
      messageKey: 'expression.error.paren_missing',
      params: { count: depth, char: ')' },
    }]
  }

  const out: ExprDiagnostic[] = []
  const trimmed = text.trim()
  const lead = text.indexOf(trimmed)

  if (/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(trimmed) && !['true', 'false', 'null'].includes(trimmed)) {
    out.push({
      from: lead, to: lead + trimmed.length,
      messageKey: 'expression.error.bare_word',
      params: { var: trimmed },
    })
  }

  if (/[+\-*/%<>=!&|?:,]$/.test(trimmed)) {
    const pos = lead + trimmed.length - 1
    out.push({ from: pos, to: pos + 1, messageKey: 'expression.error.op_end' })
  }

  const seen = new Set<string>()
  for (const call of scanCalls(text)) {
    if (known.has(call.name) || seen.has(call.name)) continue
    seen.add(call.name)
    out.push({
      from: call.from, to: call.to,
      messageKey: 'expression.error.unknown_fn',
      params: { name: call.name },
    })
  }
  return out
}
