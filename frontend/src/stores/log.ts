import { defineStore } from 'pinia'
import { ref } from 'vue'
import { parseLine, type LogLine } from '@/lib/logFormat'

const RING_CAP = 500

export const useLogStore = defineStore('log', () => {
  const lines = ref<LogLine[]>([])
  const lastSeq = ref<number>(0)
  const dropDetected = ref<boolean>(false)

  // wireEvents 调：(seq, rawLines)
  function append(seq: number, raw: string[]) {
    if (lastSeq.value !== 0 && seq !== lastSeq.value + 1) {
      // 丢包检测：注入一条 warning 行
      dropDetected.value = true
      lines.value.push({
        time: new Date().toISOString(),
        level: 'warn',
        message: `[log:lines] sequence gap: expected ${lastSeq.value + 1}, got ${seq}`,
      })
    }
    lastSeq.value = seq
    for (const r of raw) {
      lines.value.push(parseLine(r))
      if (lines.value.length > RING_CAP) {
        lines.value.splice(0, lines.value.length - RING_CAP)
      }
    }
  }

  function clear() {
    lines.value = []
    lastSeq.value = 0
    dropDetected.value = false
  }

  return { lines, lastSeq, dropDetected, append, clear }
})
