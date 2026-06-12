// clips store — 全局 clip 资产 (asset 库 KindClip) 元数据 + 单 clip get/save/update/delete.
// 资产全局化后无容器级/库级两套存储: list 列全局所有 clip (ClipSummary, 无 events);
// get 返 InputClip (events 走 binary chunk + ClipResolver, JSON RPC 仅拿 meta).
// 后端 Save/Delete/Update 均会 emit 'clip:changed' — listen() 订阅后自动 refresh 列表.

import { defineStore } from 'pinia'
import { ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend } from '@/lib/backend'

// ClipMeta 跟 backend inputclip.ClipMeta 对齐 (model.go).
export interface ClipMeta {
  mouseMode: string // 'relative' | 'absolute' | 'mixed'
  baseResolution: [number, number] // [w, h]
  mouseCounts360: number
  filterMode: string // 'precise' | 'simple'
  stopHotkeyVK: number // 默认 0x7B (F12)
}

// ClipSummary 列表项 — 不含 events.
export interface ClipSummary {
  id: string
  label: string
  description?: string
  tags?: string[]
  durationUs: number
  createdAt: string
  meta: ClipMeta
  eventCount: number
}

// InputClip JSON RPC 拿到的元数据. events 走 binary chunk + ClipResolver, JSON 不传.
export interface InputClip {
  id: string
  label: string
  description?: string
  tags?: string[]
  durationUs: number
  createdAt: string
  meta: ClipMeta
}

export const useClipsStore = defineStore('clips', () => {
  const clips = ref<ClipSummary[]>([])
  const loading = ref(false)

  async function refresh() {
    loading.value = true
    try {
      const list = (await backend.clipsContainer.list()) as ClipSummary[] | undefined
      clips.value = list ?? []
    } finally {
      loading.value = false
    }
  }

  async function get(id: string): Promise<InputClip | null> {
    try {
      const clip = (await backend.clipsContainer.get(id)) as InputClip | null | undefined
      return clip ?? null
    } catch (e) {
      console.error('clips.get failed', e)
      return null
    }
  }

  async function save(clip: InputClip): Promise<void> {
    await backend.clipsContainer.save(clip)
    await refresh()
  }

  async function update(
    id: string,
    patch: { label?: string; description?: string; tags?: string[] },
  ): Promise<void> {
    await backend.clipsContainer.update(
      id,
      patch.label ?? '',
      patch.description ?? '',
      patch.tags ?? [],
    )
    await refresh()
  }

  async function remove(id: string): Promise<void> {
    await backend.clipsContainer.delete_(id)
    await refresh()
  }

  // 'clip:changed' 全局订阅 — backend Save/Delete/Update 后 emit, 任意 UI 都重 list.
  let unsubscribe: (() => void) | null = null
  function listen() {
    if (unsubscribe) return
    unsubscribe = Events.On('clip:changed', () => {
      refresh().catch(console.error)
    })
  }
  function unlisten() {
    if (unsubscribe) {
      unsubscribe()
      unsubscribe = null
    }
  }

  return { clips, loading, refresh, get, save, update, remove, listen, unlisten }
})
