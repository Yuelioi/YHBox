import { ref, watch } from 'vue'

interface Options {
  default: number
  min: number
  max: number
}

export function useSplitpane(key: string, opts: Options) {
  const stored = localStorage.getItem(key)
  const initial = parseInitial(stored, opts)
  const width = ref(initial)

  function setWidth(v: number) {
    const clamped = Math.min(opts.max, Math.max(opts.min, v))
    width.value = clamped
    localStorage.setItem(key, String(clamped))
  }

  watch(width, (v) => {
    localStorage.setItem(key, String(v))
  })

  return { width, setWidth }
}

function parseInitial(stored: string | null, opts: Options): number {
  if (!stored) return opts.default
  const n = Number(stored)
  if (!Number.isFinite(n)) return opts.default
  return Math.min(opts.max, Math.max(opts.min, n))
}
