// 代码片段 store — Script/Expr 放大编辑工具栏「片段」下拉的用户自建片段。
// 持久化在后端 <dataDir>/snippets.json (codesnippet service, 整存整取):
// 首次用到时 load 全量进内存, 每次增删改把全量列表回写。
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend } from '@/lib/backend'

export type CodeSnippetLang = 'script' | 'expr'

export interface CodeSnippet {
  id: string
  lang: CodeSnippetLang
  name: string
  /** 字面插入光标处 — 不支持 ${} 占位 (与 JS 模板字符串语法冲突, 见 spec)。 */
  body: string
}

export const useCodeSnippetsStore = defineStore('codeSnippets', () => {
  const snippets = ref<CodeSnippet[]>([])
  let loaded = false

  async function ensureLoaded() {
    if (loaded) return
    loaded = true
    const list = await backend.codeSnippets.list()
    if (Array.isArray(list)) snippets.value = list as CodeSnippet[]
  }

  function persist() {
    void backend.codeSnippets.saveAll(snippets.value)
  }

  function byLang(lang: CodeSnippetLang): CodeSnippet[] {
    return snippets.value.filter((s) => s.lang === lang)
  }

  function add(lang: CodeSnippetLang, name: string, body: string): CodeSnippet {
    const s: CodeSnippet = { id: crypto.randomUUID(), lang, name, body }
    snippets.value.push(s)
    persist()
    return s
  }

  function update(id: string, patch: Partial<Pick<CodeSnippet, 'name' | 'body'>>): void {
    const idx = snippets.value.findIndex((s) => s.id === id)
    if (idx === -1) return
    snippets.value[idx] = { ...snippets.value[idx], ...patch }
    persist()
  }

  function remove(id: string): void {
    const idx = snippets.value.findIndex((s) => s.id === id)
    if (idx === -1) return
    snippets.value.splice(idx, 1)
    persist()
  }

  return { snippets, ensureLoaded, byLang, add, update, remove }
})
