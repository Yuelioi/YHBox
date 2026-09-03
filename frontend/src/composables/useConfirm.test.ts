import { describe, expect, it } from 'vitest'
import { useConfirm } from './useConfirm'

describe('useConfirm pending alternate action', () => {
  it('keeps the dialog open and exposes pending copy after resolving the alternate choice', async () => {
    const { confirm, state, resolveActive, finishPending } = useConfirm()
    const result = confirm({
      title: 'Unsaved changes',
      alternateText: 'Discard changes',
      alternateValue: 'discard',
      alternatePendingText: 'Restoring and exiting…',
    })

    resolveActive('discard')

    await expect(result).resolves.toBe('discard')
    expect(state.open).toBe(true)
    expect(state.pending).toBe(true)
    expect(state.pendingText).toBe('Restoring and exiting…')

    finishPending()
    expect(state.open).toBe(false)
    expect(state.pending).toBe(false)
  })
})
