import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend, type FishStatsEvent } from '@/lib/backend'

export const useFishStore = defineStore('fish', () => {
  const state = ref<'idle' | 'running' | 'paused'>('idle')
  const stats = ref<FishStatsEvent>({
    commonCount: 0,
    purpleCount: 0,
    goldenCount: 0,
    startedAt: '',
  })

  function setState(s: 'idle' | 'running' | 'paused') {
    state.value = s
  }
  function setStats(s: FishStatsEvent) {
    stats.value = s
  }

  // Phase 2 实现
  async function start() {
    return backend.fish.start()
  }
  async function pause() {
    return backend.fish.pause()
  }
  async function resume() {
    return backend.fish.resume()
  }
  async function stop() {
    return backend.fish.stop()
  }

  return { state, stats, setState, setStats, start, pause, resume, stop }
})
