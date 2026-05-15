import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend, type GameStatusEvent } from '@/lib/backend'

export const useGameStore = defineStore('game', () => {
  const status = ref<GameStatusEvent | null>(null)

  function setStatus(s: GameStatusEvent) {
    status.value = s
  }

  async function detect() {
    const s = await backend.game.detect()
    if (s) setStatus(s)
  }

  return { status, setStatus, detect }
})
