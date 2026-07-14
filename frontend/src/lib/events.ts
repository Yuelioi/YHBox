import { backend } from './backend'
import { useHotkeysStore } from '@/stores/hotkeys'
import { useLogStore } from '@/stores/log'

let disposeCurrent: (() => void) | undefined

// wireEvents owns the process-wide subscriptions. Calling it twice replaces
// the previous listeners instead of duplicating every backend log batch.
export function wireEvents() {
  disposeCurrent?.()

  const offLog = backend.events.onLogBatch((batch) =>
    useLogStore().appendBatch(batch.seq, batch.entries ?? [], batch.dropped ?? 0),
  )

  void useHotkeysStore().reload()
  const offHotkeys = backend.events.onHotkeyChanged(() => {
    void useHotkeysStore().reload()
  })

  disposeCurrent = () => {
    offLog?.()
    offHotkeys?.()
    disposeCurrent = undefined
  }
  return disposeCurrent
}
