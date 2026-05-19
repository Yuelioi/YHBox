// frontend/src/stores/discovery.ts
// Editor v2 B — favorites + recent node kinds backed by localStorage.
// Spec: editor-v2-node-discovery-design.md §6.1

import { defineStore } from 'pinia'
import { ref } from 'vue'

const FAV_KEY = 'yhfish.favorites'
const RECENT_KEY = 'yhfish.recent'
const RECENT_MAX = 8
export const FAVORITES_MAX = 20

export const useDiscoveryStore = defineStore('discovery', () => {
  const favorites = ref<string[]>([])
  const recent = ref<string[]>([])

  function loadFromLocalStorage() {
    try {
      const f = localStorage.getItem(FAV_KEY)
      if (f) favorites.value = JSON.parse(f)
      const r = localStorage.getItem(RECENT_KEY)
      if (r) recent.value = JSON.parse(r)
    } catch (_e) {
      // bad JSON / unavailable storage → keep empties
    }
  }

  function persistFavorites() {
    try {
      localStorage.setItem(FAV_KEY, JSON.stringify(favorites.value))
    } catch (_e) {}
  }

  function persistRecent() {
    try {
      localStorage.setItem(RECENT_KEY, JSON.stringify(recent.value))
    } catch (_e) {}
  }

  function toggleFavorite(kind: string): void {
    const idx = favorites.value.indexOf(kind)
    if (idx >= 0) {
      favorites.value.splice(idx, 1)
    } else {
      if (favorites.value.length >= FAVORITES_MAX) {
        favorites.value.shift()
      }
      favorites.value.push(kind)
    }
    persistFavorites()
  }

  function pushRecent(kind: string): void {
    const idx = recent.value.indexOf(kind)
    if (idx >= 0) recent.value.splice(idx, 1)
    recent.value.unshift(kind)
    if (recent.value.length > RECENT_MAX) recent.value.length = RECENT_MAX
    persistRecent()
  }

  return { favorites, recent, toggleFavorite, pushRecent, loadFromLocalStorage }
})
