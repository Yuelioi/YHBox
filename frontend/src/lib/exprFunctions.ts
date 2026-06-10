// Expr 内置函数元数据 — ExprInput 补全下拉 + 即时未知函数检查用.
// 单一来源是 Go internal/services/expr/builtins.go; 两侧测试用同一预期字面量互锁
// (exprFunctions.spec.ts ↔ builtins_test.go), 改函数表必须四处同步.

export interface ExprFunction {
  name: string
  /** 签名展示串, 参数名用通用英文 (跨 locale 不翻译). */
  sig: string
  minArgs: number
  maxArgs: number
}

export const EXPR_FUNCTIONS: ExprFunction[] = [
  { name: 'abs', sig: 'abs(x)', minArgs: 1, maxArgs: 1 },
  { name: 'ceil', sig: 'ceil(x)', minArgs: 1, maxArgs: 1 },
  { name: 'clamp', sig: 'clamp(x, min, max)', minArgs: 3, maxArgs: 3 },
  { name: 'floor', sig: 'floor(x)', minArgs: 1, maxArgs: 1 },
  { name: 'max', sig: 'max(a, b)', minArgs: 2, maxArgs: 2 },
  { name: 'min', sig: 'min(a, b)', minArgs: 2, maxArgs: 2 },
  { name: 'now', sig: 'now()', minArgs: 0, maxArgs: 0 },
  { name: 'pow', sig: 'pow(x, y)', minArgs: 2, maxArgs: 2 },
  { name: 'round', sig: 'round(x, digits?)', minArgs: 1, maxArgs: 2 },
  { name: 'sqrt', sig: 'sqrt(x)', minArgs: 1, maxArgs: 1 },
]

const FN_NAMES = new Set(EXPR_FUNCTIONS.map(f => f.name))

/** 光标处 token (往前扫 identifier 字符). 补全替换区间 = [start, caret). */
export function tokenAtCaret(text: string, caret: number): { token: string; start: number } {
  const end = Math.min(Math.max(caret, 0), text.length)
  let i = end
  while (i > 0 && /[a-zA-Z0-9_]/.test(text[i - 1])) i--
  return { token: text.slice(i, end), start: i }
}

/** 文本里"长得像函数调用"且不在内置表的名字 (去重). 跳过双引号字符串字面量 (含 \" 转义). */
export function unknownFnsIn(text: string): string[] {
  const out: string[] = []
  const seen = new Set<string>()
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
      const name = text.slice(i, j)
      let k = j
      while (k < text.length && text[k] === ' ') k++
      if (text[k] === '(' && !FN_NAMES.has(name) && !seen.has(name)) {
        seen.add(name)
        out.push(name)
      }
      i = j
      continue
    }
    i++
  }
  return out
}
