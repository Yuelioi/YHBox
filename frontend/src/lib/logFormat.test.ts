import { describe, expect, it } from 'vitest'
import { workflowFailureSummary } from './logFormat'

describe('workflowFailureSummary', () => {
  it('formats routed failure metadata without exposing raw causes', () => {
    expect(
      workflowFailureSummary({
        failure: {
          code: 'ai.generation_failed',
          category: 'provider',
          retryHint: false,
          sourceNodeId: 'ai-generate',
          sourcePortId: 'failed',
          attempt: 2,
        },
      }),
    ).toBe('ai.generation_failed · ai-generate:failed · attempt 2')
  })

  it('ignores unrelated or malformed fields', () => {
    expect(workflowFailureSummary({ durationMs: 12 })).toBe('')
    expect(workflowFailureSummary({ failure: { code: 42 } })).toBe('')
  })
})
