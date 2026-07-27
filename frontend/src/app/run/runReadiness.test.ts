import { describe, expect, it } from 'vitest'
import { readinessOutcome, runStartOutcome } from './runReadiness'

describe('run readiness result', () => {
  it('keeps a started Run on the fast path', () => {
    expect(
      runStartOutcome({
        diagnostics: [],
        run: { runId: 'run-1' },
        readiness: { state: 'started' },
      } as never),
    ).toEqual({ state: 'started', runId: 'run-1' })
  })

  it('turns a missing target into a fixable result instead of an exception', () => {
    expect(
      runStartOutcome({
        diagnostics: [],
        readiness: {
          state: 'target-required',
          code: 'admission.target_unavailable',
          slot: 'game-window',
        },
      } as never),
    ).toEqual({
      state: 'target-required',
      code: 'admission.target_unavailable',
      slot: 'game-window',
      nodeId: undefined,
    })
  })

  it('preserves compiler diagnostics for direct repair', () => {
    expect(
      runStartOutcome({
        diagnostics: [{ severity: 'error', code: 'INVALID_CONFIG', nodeId: 'click' }],
      } as never),
    ).toEqual({ state: 'workflow-invalid', code: 'INVALID_CONFIG', nodeId: 'click' })
  })

  it('reuses the same readiness interpretation for schedules', () => {
    expect(readinessOutcome({ state: 'credential-required', slot: 'account' })).toEqual({
      state: 'credential-required',
      code: undefined,
      slot: 'account',
      nodeId: undefined,
    })
  })
})
