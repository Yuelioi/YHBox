import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend } from '@/lib/backend'

export const useRhythmStore = defineStore('rhythm', () => {
  const state = ref<'idle' | 'running' | 'paused'>('idle')
  // 跨组件/导航/reload 保留 startedAt（FishView 走 backend stats 的 startedAt；rhythm 没 stats，前端自己记）
  const startedAt = ref<number>(0)

  function setState(s: 'idle' | 'running' | 'paused') {
    // idle→running 边沿才更新 startedAt（resume 不重置）；running/paused→idle 清零
    if (s === 'running' && state.value === 'idle') {
      startedAt.value = Date.now()
    } else if (s === 'idle') {
      startedAt.value = 0
    }
    state.value = s
  }

  async function start() {
    return backend.rhythm.start()
  }
  async function pause() {
    return backend.rhythm.pause()
  }
  async function resume() {
    return backend.rhythm.resume()
  }
  async function stop() {
    return backend.rhythm.stop()
  }

  return { state, startedAt, setState, start, pause, resume, stop }
})
