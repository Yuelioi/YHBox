import { describe, expect, it } from 'vitest'
import { workflowNodeVisualState } from './workflowNodeVisualState'

describe('workflow node visual state', () => {
  it('keeps selection visually dominant after a successful run', () => {
    const state = workflowNodeVisualState({ selected: true, runStatus: 'succeeded' })

    expect(state.surfaceClasses).toContain('ring-primary')
    expect(state.surfaceClasses).not.toContain('border-success')
    expect(state.executionTone).toBe('success')
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
    expect(state.debugCurrent).toBe(true)
    expect(state.diagnosticTone).toBe('warning')
  })
})
