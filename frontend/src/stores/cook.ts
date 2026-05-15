import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend } from '@/lib/backend'

export const useCookStore = defineStore('cook', () => {
  const state = ref<'idle' | 'running' | 'paused'>('idle')

  function setState(s: 'idle' | 'running' | 'paused') {
    state.value = s
  }

  async function start() {
    return backend.cook.start()
  }
  async function pause() {
    return backend.cook.pause()
  }
  async function resume() {
    return backend.cook.resume()
  }
  async function stop() {
    return backend.cook.stop()
  }
  async function setIntervalMs(ms: number) {
    return backend.cook.setIntervalMs(ms)
  }

  return { state, setState, start, pause, resume, stop, setIntervalMs }
})
