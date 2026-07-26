import { describe, expect, it } from 'vitest'
import { workflowNodeVisualState } from './workflowNodeVisualState'

describe('workflow node visual state', () => {
  it('returns successful nodes to their neutral resting state', () => {
    const state = workflowNodeVisualState({ selected: true, runStatus: 'succeeded' })

    expect(state.surfaceClasses).toContain('ring-primary')
    expect(state.surfaceClasses).not.toContain('border-success')
    expect(state.executionTone).toBeNull()
    expect(state.executionStripeClasses).toBe('')
    expect(state.showRunStatus).toBe(false)
  })

  it('keeps debug and validation separate from selection and execution', () => {
    const state = workflowNodeVisualState({
      selected: true,
      runStatus: 'failed',
      debugCurrent: true,
      diagnosticSeverity: 'warning',
    })

    expect(state.surfaceClasses).toContain('ring-primary')
    expect(state.executionTone).toBe('error')
    expect(state.showRunStatus).toBe(true)
    expect(state.debugCurrent).toBe(true)
    expect(state.diagnosticTone).toBe('warning')
  })
})
