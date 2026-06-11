// 代码片段 store — Script/Expr 放大编辑工具栏「片段」下拉的用户自建部分
// (内置模板仍由各编辑器以 props.snippets 传入, 不进这里)。
// 存储: localStorage (对齐 stores/snippets.ts 先例; schema 稳定后迁 backend file)。
import { defineStore } from 'pinia'
import { ref } from 'vue'

export type CodeSnippetLang = 'script' | 'expr'

export interface CodeSnippet {
  id: string
  lang: CodeSnippetLang
  name: string
  /** 字面插入光标处 — 不支持 ${} 占位 (与 JS 模板字符串语法冲突, 见 spec)。 */
  body: string
  updatedAt: number
}

const STORAGE_KEY = 'yotta.codeSnippets'

export const useCodeSnippetsStore = defineStore('codeSnippets', () => {
  const snippets = ref<CodeSnippet[]>([])

  function load() {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      if (!raw) return
      const parsed = JSON.parse(raw)
      if (Array.isArray(parsed)) snippets.value = parsed
    } catch {
      // 坏数据 / localStorage 不可用 → 空列表起步
    }
  }

  function persist() {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(snippets.value))
    } catch {
      // quota / 不可用 → 静默 (片段仍在内存里可用)
    }
  }

  function byLang(lang: CodeSnippetLang): CodeSnippet[] {
    return snippets.value.filter((s) => s.lang === lang)
  }

  function add(lang: CodeSnippetLang, name: string, body: string): CodeSnippet {
    const s: CodeSnippet = { id: crypto.randomUUID(), lang, name, body, updatedAt: Date.now() }
    snippets.value.push(s)
    persist()
    return s
  }

  function update(id: string, patch: Partial<Pick<CodeSnippet, 'name' | 'body'>>): void {
    const idx = snippets.value.findIndex((s) => s.id === id)
    if (idx === -1) return
    snippets.value[idx] = { ...snippets.value[idx], ...patch, updatedAt: Date.now() }
    persist()
  }

  function remove(id: string): void {
    const idx = snippets.value.findIndex((s) => s.id === id)
    if (idx === -1) return
    snippets.value.splice(idx, 1)
    persist()
  }

  load()
  return { snippets, byLang, add, update, remove }
})
