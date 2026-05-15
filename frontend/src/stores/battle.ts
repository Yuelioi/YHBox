import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend, type BattleStateEvent } from '@/lib/backend'

export const useBattleStore = defineStore('battle', () => {
  const status = ref<BattleStateEvent>({ enabled: false, mods: 'Ctrl+Shift' })

  function setStatus(s: BattleStateEvent) {
    status.value = s
  }

  async function enable() {
    return backend.battle.enable()
  }
  async function disable() {
    return backend.battle.disable()
  }
  async function setMods(label: string) {
    return backend.battle.setMods(label)
  }

  return { status, setStatus, enable, disable, setMods }
})
