// 用户片段 → CM 补全项 (Script/Expr 共用): label = 触发词 (VSCode prefix 同义),
// 上屏把触发词整体替换成 body。info 浮层 = 名称 + 描述 + body 预览。
import type { Completion } from '@codemirror/autocomplete'
import type { CodeSnippet } from '@/stores/codeSnippets'

export function snippetCompletions(list: CodeSnippet[]): Completion[] {
  return list.map((s) => ({
    label: s.prefix,
    type: 'snippet',
    detail: s.name,
    info: () => snippetInfo(s),
    apply: s.body,
  }))
}

function snippetInfo(s: CodeSnippet): HTMLElement {
  const root = document.createElement('div')
  root.className = 'cm-yh-doc'
  const sig = root.appendChild(document.createElement('div'))
  sig.className = 'cm-yh-doc-sig'
  sig.textContent = s.name
  if (s.description) {
    const desc = root.appendChild(document.createElement('div'))
    desc.className = 'cm-yh-doc-desc'
    desc.textContent = s.description
  }
  const body = root.appendChild(document.createElement('div'))
  body.className = 'cm-yh-doc-snippet-body'
  body.textContent = s.body
  return root
}
