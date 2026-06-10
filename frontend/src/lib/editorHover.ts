// 悬停函数名 → 文档浮层 (签名 + 人话说明 + 参数表)。Expr / Script 共用;
// 函数数据与 i18n 文案由调用方经 lookup 注入, 本文件只管词提取和 DOM。
import type { EditorView, Tooltip } from '@codemirror/view'

export interface HoverDoc {
  sig: string
  desc?: string
  params?: { name: string; label: string; type: string; required?: boolean }[]
}

/** hoverTooltip 的 source: 光标下提词 → lookup → 浮层。
    链式名先整体查 (vars.get), 不中再查首段 (FindTemplate)。 */
export function fnHoverTooltip(lookup: (word: string) => HoverDoc | null) {
  return (view: EditorView, pos: number): Tooltip | null => {
    const line = view.state.doc.lineAt(pos)
    const re = /[A-Za-z_$][\w$]*(?:\.[A-Za-z_]\w*)*/g
    let m: RegExpExecArray | null
    while ((m = re.exec(line.text))) {
      const start = line.from + m.index
      const end = start + m[0].length
      if (pos < start) return null
      if (pos > end) continue
      const doc = lookup(m[0]) ?? lookup(m[0].split('.')[0]!)
      if (!doc) return null
      return { pos: start, end, above: true, create: () => ({ dom: renderDoc(doc) }) }
    }
    return null
  }
}

export function renderDoc(doc: HoverDoc): HTMLElement {
  const root = document.createElement('div')
  root.className = 'cm-yh-doc'
  const sig = root.appendChild(document.createElement('div'))
  sig.className = 'cm-yh-doc-sig'
  sig.textContent = doc.sig
  if (doc.desc) {
    const desc = root.appendChild(document.createElement('div'))
    desc.className = 'cm-yh-doc-desc'
    desc.textContent = doc.desc
  }
  for (const p of doc.params ?? []) {
    const row = root.appendChild(document.createElement('div'))
    row.className = 'cm-yh-doc-param'
    const name = row.appendChild(document.createElement('span'))
    name.className = 'cm-yh-doc-param-name'
    name.textContent = p.name
    const type = row.appendChild(document.createElement('span'))
    type.className = 'cm-yh-doc-param-type'
    type.textContent = p.required ? `${p.type} *` : p.type
    if (p.label) {
      const label = row.appendChild(document.createElement('span'))
      label.className = 'cm-yh-doc-param-label'
      label.textContent = p.label
    }
  }
  return root
}
