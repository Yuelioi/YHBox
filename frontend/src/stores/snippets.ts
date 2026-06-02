// Snippets store — 用户资产系统. 替代旧 Favorites/Recents.
//
// Snippet 是"带配置的可复用节点模板", 不是"收藏节点 kind". 跟 UE Blueprint 的 Preset
// 或 Figma 的 Component 类似. 举例: 用户在画布配好"目标窗口节点" 含 title=异环 + class=UnrealWindow,
// 保存为 snippet, 下次新容器直接拖出来 = 无需再点选窗口.
//
// 字段设计:
//   - name/description/tags: 基础元数据
//   - category: 单值 — sidebar 树状导航维度 (按游戏/项目分: '异环'/'原神'/'通用')
//   - tags: 多值 — 搜索 / filter 维度 (跨 category)
//   - color/icon: 视觉 identity (红=危险/蓝=窗口/绿=OCR)
//   - shortcut: 全局快捷键字符串 (normalize 成 lowercase), 触发在 mouse 位置生成
//   - usageCount/lastUsedAt: 预埋, 后续智能排序用 (零成本)
//   - payload: discriminated union, 当前只 'node' (single-node), 后续扩 'subgraph'/'multi-node'
//
// 存储: localStorage (现在). 等 schema 稳定后迁 backend file (snippets.json + workspace-scoped).
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { i18n } from '@/i18n'

export interface Snippet {
  id: string
  name: string
  description?: string
  category?: string
  tags: string[]
  color?: string
  icon?: string
  shortcut?: string
  createdAt: string
  updatedAt: string
  usageCount: number
  lastUsedAt?: string
  payload: SnippetPayload
}

export type SnippetPayload = {
  type: 'node'
  kind: string
  config: Record<string, unknown>
}

const STORAGE_KEY = 'yotta.snippets'

export const useSnippetsStore = defineStore('snippets', () => {
  const snippets = ref<Snippet[]>([])
  const loaded = ref(false)

  function load() {
    if (loaded.value) return
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      console.log('[snippets store] load(): localStorage raw =', raw)
      if (raw) {
        const parsed = JSON.parse(raw) as Snippet[]
        if (Array.isArray(parsed)) {
          snippets.value = parsed
          console.log('[snippets store] loaded', parsed.length, 'snippets')
        }
      }
    } catch (e) {
      console.error('[snippets store] load error:', e)
    }
    loaded.value = true
  }

  function persist() {
    try {
      const json = JSON.stringify(snippets.value)
      localStorage.setItem(STORAGE_KEY, json)
      console.log('[snippets store] persist', snippets.value.length, 'snippets (', json.length, 'bytes)')
    } catch (e) {
      console.error('[snippets store] persist error (' + i18n.global.t('snippet.quota_unavailable') + '):', e)
    }
  }

  function create(input: Omit<Snippet, 'id' | 'createdAt' | 'updatedAt' | 'usageCount'>): Snippet {
    const now = new Date().toISOString()
    const s: Snippet = {
      ...input,
      id: crypto.randomUUID(),
      createdAt: now,
      updatedAt: now,
      usageCount: 0,
    }
    snippets.value.push(s)
    persist()
    return s
  }

  function update(id: string, patch: Partial<Omit<Snippet, 'id' | 'createdAt'>>): Snippet | null {
    const idx = snippets.value.findIndex((s) => s.id === id)
    if (idx < 0) return null
    const updated: Snippet = {
      ...snippets.value[idx],
      ...patch,
      updatedAt: new Date().toISOString(),
    }
    snippets.value[idx] = updated
    persist()
    return updated
  }

  function remove(id: string) {
    const idx = snippets.value.findIndex((s) => s.id === id)
    if (idx < 0) return
    snippets.value.splice(idx, 1)
    persist()
  }

  function getById(id: string): Snippet | undefined {
    return snippets.value.find((s) => s.id === id)
  }

  /** 记录 snippet 被使用 (拖拽到画布或快捷键触发). 后续用 lastUsedAt 智能排序. */
  function markUsed(id: string) {
    const s = getById(id)
    if (!s) return
    s.usageCount += 1
    s.lastUsedAt = new Date().toISOString()
    persist()
  }

  /** 按 category 分组. 没填 category 的归 '通用'. 每组内按 usageCount desc + name asc 排序. */
  const byCategory = computed(() => {
    const generalCat = i18n.global.t('snippet.general')
    const groups = new Map<string, Snippet[]>()
    for (const s of snippets.value) {
      const cat = s.category?.trim() || generalCat
      if (!groups.has(cat)) groups.set(cat, [])
      groups.get(cat)!.push(s)
    }
    for (const list of groups.values()) {
      list.sort((a, b) => b.usageCount - a.usageCount || a.name.localeCompare(b.name))
    }
    // 通用 排最后, 其他按 name asc
    const cats = [...groups.keys()].sort((a, b) => {
      if (a === generalCat) return 1
      if (b === generalCat) return -1
      return a.localeCompare(b)
    })
    return cats.map((cat) => ({ category: cat, list: groups.get(cat)! }))
  })

  /** 全 tag 去重 (供 filter chip). */
  const allTags = computed(() => {
    const set = new Set<string>()
    for (const s of snippets.value) for (const t of s.tags) set.add(t)
    return [...set].sort()
  })

  /** 全 category 去重 (供 SaveSnippetDrawer category dropdown). */
  const allCategories = computed(() => {
    const set = new Set<string>()
    for (const s of snippets.value) {
      if (s.category?.trim()) set.add(s.category.trim())
    }
    return [...set].sort()
  })

  /** 按 shortcut 索引 (供 global keydown 监听快查). normalize lowercase 后做 key. */
  const byShortcut = computed(() => {
    const map = new Map<string, Snippet>()
    for (const s of snippets.value) {
      if (s.shortcut) map.set(normalizeShortcut(s.shortcut), s)
    }
    return map
  })

  return {
    snippets,
    loaded,
    load,
    create,
    update,
    remove,
    getById,
    markUsed,
    byCategory,
    allTags,
    allCategories,
    byShortcut,
  }
})

/**
 * normalize 快捷键字符串成 stable key: lowercase + 修饰键固定顺序 (ctrl→shift→alt→meta→key).
 *   'Ctrl+Shift+F' / 'shift+ctrl+f' / 'CTRL + SHIFT + F' → 'ctrl+shift+f'
 */
export function normalizeShortcut(s: string): string {
  const parts = s
    .toLowerCase()
    .split('+')
    .map((p) => p.trim())
    .filter(Boolean)
  const mods = new Set<string>()
  let key = ''
  for (const p of parts) {
    if (p === 'ctrl' || p === 'control') mods.add('ctrl')
    else if (p === 'shift') mods.add('shift')
    else if (p === 'alt' || p === 'option') mods.add('alt')
    else if (p === 'meta' || p === 'cmd' || p === 'command' || p === 'win') mods.add('meta')
    else key = p
  }
  const order = ['ctrl', 'shift', 'alt', 'meta']
  const sortedMods = order.filter((m) => mods.has(m))
  return [...sortedMods, key].filter(Boolean).join('+')
}

/** 系统/编辑器保留快捷键, 用户不能覆盖 (会破坏 copy/paste/undo 等). */
export const RESERVED_SHORTCUTS = new Set([
  'ctrl+c', 'ctrl+v', 'ctrl+x', 'ctrl+z', 'ctrl+y', 'ctrl+s', 'ctrl+a', 'ctrl+d',
  'ctrl+shift+z', 'delete', 'backspace', 'escape', 'enter', 'space', 'tab',
])

export function isReservedShortcut(s: string): boolean {
  return RESERVED_SHORTCUTS.has(normalizeShortcut(s))
}

/** 从 KeyboardEvent 构造 normalize key, 供 global keydown 监听匹配 byShortcut. */
export function eventToShortcutKey(e: KeyboardEvent): string {
  const parts: string[] = []
  if (e.ctrlKey) parts.push('ctrl')
  if (e.shiftKey) parts.push('shift')
  if (e.altKey) parts.push('alt')
  if (e.metaKey) parts.push('meta')
  // e.key 是 'F' 或 'a' 等; 修饰键单独不算
  const k = e.key.toLowerCase()
  if (!['control', 'shift', 'alt', 'meta'].includes(k)) {
    parts.push(k)
  }
  return parts.join('+')
}
