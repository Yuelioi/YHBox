// signature help: 光标在函数调用括号内时, 浮层显示该函数签名 + 高亮当前参数。
// Expr (单行 DSL) 走字符串扫描; Script (JS) 走 lezer 语法树。
import { javascript } from '@codemirror/lang-javascript'
import { showTooltip, type Tooltip } from '@codemirror/view'
import { StateField, type EditorState, type Extension } from '@codemirror/state'
import { renderSignature } from '@/lib/editorHover'

// ── Expr: 字符串扫描, 跳过双引号串 (含 \" 转义), 找 pos 处最内层未闭合的括号 ──

export function exprSigContext(
  text: string,
  pos: number,
): { name: string; argIndex: number } | null {
  const stack: { name: string; argIndex: number }[] = []
  let i = 0
  while (i < pos) {
    const c = text[i]
    if (c === '"') {
      i++
      while (i < text.length && !(text[i] === '"' && text[i - 1] !== '\\')) i++
      i++
      continue
    }
    if (c === '(') {
      // 往前找紧邻的标识符作为函数名 (跳过空格)
      let k = i
      while (k > 0 && text[k - 1] === ' ') k--
      let j = k
      while (j > 0 && /[a-zA-Z0-9_]/.test(text[j - 1])) j--
      const name = text.slice(j, k)
      stack.push({ name, argIndex: 0 })
      i++
      continue
    }
    if (c === ')') {
      stack.pop()
      i++
      continue
    }
    if (c === ',') {
      if (stack.length) stack[stack.length - 1]!.argIndex++
      i++
      continue
    }
    i++
  }
  const top = stack[stack.length - 1]
  if (!top || !top.name) return null
  return { name: top.name, argIndex: top.argIndex }
}

// ── Script: lezer 语法树, 找包裹 pos 的最内层 ArgList → 其 CallExpression ──

const jsParser = javascript().language.parser

export function scriptSigContext(
  doc: string,
  pos: number,
): { name: string; argIndex: number } | null {
  const tree = jsParser.parse(doc)
  let node = tree.resolveInner(pos, -1) as ReturnType<typeof tree.resolveInner> | null
  while (node && node.name !== 'ArgList') node = node.parent
  if (!node) return null
  const argList = node
  const call = argList.parent
  if (!call || call.name !== 'CallExpression') return null
  const callee = call.firstChild
  if (!callee) return null
  const name = doc.slice(callee.from, callee.to)
  let argIndex = 0
  const c = argList.cursor()
  if (c.firstChild()) {
    do {
      if (c.name === ',' && c.to <= pos) argIndex++
    } while (c.nextSibling())
  }
  return { name, argIndex }
}

// ── Script: 光标落在节点调用对象字面量的某个 pin 值位置 → {kind, pin} ──
//    用于在值位置补全: 枚举 pin → 候选值 (Scope→auto/local/global); varname pin → 容器变量名。
//    走 lezer 树: Property → ObjectExpression → ArgList → CallExpression。
export function scriptPinValueContext(
  doc: string,
  pos: number,
): { kind: string; pin: string } | null {
  const tree = jsParser.parse(doc)
  let prop = tree.resolveInner(pos, -1) as ReturnType<typeof tree.resolveInner> | null
  while (prop && prop.name !== 'Property') prop = prop.parent
  if (!prop) return null
  // key = 属性名 (PropertyDefinition); 光标须在 key 之后 (值位, 不是在编辑 key 本身)
  const key = prop.getChild('PropertyDefinition') ?? prop.getChild('PropertyName')
  if (!key || pos <= key.to) return null
  const obj = prop.parent
  if (!obj || obj.name !== 'ObjectExpression') return null
  const argList = obj.parent
  if (!argList || argList.name !== 'ArgList') return null
  const call = argList.parent
  if (!call || call.name !== 'CallExpression') return null
  const callee = call.firstChild
  if (!callee) return null
  return { kind: doc.slice(callee.from, callee.to), pin: doc.slice(key.from, key.to) }
}

// ── Script: 光标落在 `r.exit === ▮` / `r.exit !== ▮` 右侧时, 反查 `r` 来自哪种节点调用。
//    v1 只做常见局部写法: `const/let/var r = NodeKind(...)`; 最近一次声明优先。
export function scriptExitCompareContext(
  doc: string,
  pos: number,
): { varName: string; kind: string; from: number } | null {
  const before = doc.slice(0, pos)
  const m = /([A-Za-z_$][\w$]*)\.exit\s*(?:={2,3}|!==?)\s*([A-Za-z_$][\w$.]*)?$/.exec(before)
  if (!m) return null
  const varName = m[1]
  const partial = m[2] ?? ''
  const compareAt = before.lastIndexOf(`${varName}.exit`)
  if (compareAt < 0) return null
  const decls = before.slice(0, compareAt)
  const re = new RegExp(
    `\\b(?:const|let|var)\\s+${escapeRegExp(varName)}\\s*=\\s*([A-Za-z_$][\\w$]*)\\s*\\(`,
    'g',
  )
  let kind = ''
  let d: RegExpExecArray | null
  while ((d = re.exec(decls))) kind = d[1]
  if (!kind) return null
  return { varName, kind, from: pos - partial.length }
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

// ── signature help 扩展: showTooltip StateField, 文档/选区变更时重算 ──

export function signatureHelp(opts: {
  context: (state: EditorState, pos: number) => { name: string; argIndex: number } | null
  lookup: (name: string) => { sig: string } | null
}): Extension {
  const field = StateField.define<Tooltip | null>({
    create: (state) => sigTooltip(state, opts),
    update: (value, tr) => (tr.docChanged || tr.selection ? sigTooltip(tr.state, opts) : value),
    provide: (f) => showTooltip.from(f),
  })
  return field
}

function sigTooltip(
  state: EditorState,
  opts: {
    context: (s: EditorState, p: number) => { name: string; argIndex: number } | null
    lookup: (n: string) => { sig: string } | null
  },
): Tooltip | null {
  const pos = state.selection.main.head
  const ctx = opts.context(state, pos)
  if (!ctx) return null
  const fn = opts.lookup(ctx.name)
  if (!fn) return null
  return { pos, above: true, create: () => ({ dom: renderSignature(fn.sig, ctx.argIndex) }) }
}
