import { describe, expect, it } from 'vitest'
import { readinessOutcome, runReadinessMessage, runStartOutcome } from './runReadiness'

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
    const outcome = runStartOutcome({
      diagnostics: [
        {
          severity: 'error',
          code: 'UNKNOWN_PORT',
          nodeId: 'click',
          params: {
            fromNodeId: 'wait',
            fromPortId: 'found',
            toNodeId: 'click',
            toPortId: 'in',
          },
        },
      ],
    } as never)

    expect(outcome).toEqual({
      state: 'workflow-invalid',
      code: 'UNKNOWN_PORT',
      nodeId: 'click',
      fromNodeId: 'wait',
      fromPortId: 'found',
      toNodeId: 'click',
      toPortId: 'in',
    })
    if (outcome.state === 'started') throw new Error('expected blocked run')
    expect(runReadinessMessage(outcome)).toContain('wait / found')
    expect(runReadinessMessage(outcome)).toContain('click / in')
    expect(runReadinessMessage(outcome)).toContain('打开工作流')
  })

  it('reuses the same readiness interpretation for schedules', () => {
    expect(
      readinessOutcome({
        state: 'credential-required',
        slot: 'account',
        fromNodeId: 'source',
        fromPortId: 'done',
        toNodeId: 'target',
        toPortId: 'in',
      }),
    ).toEqual({
      state: 'credential-required',
      code: undefined,
      slot: 'account',
      nodeId: undefined,
      fromNodeId: 'source',
      fromPortId: 'done',
      toNodeId: 'target',
      toPortId: 'in',
    })
  })

  it('explains a missing AI model instead of calling it a generic target', () => {
    const outcome = readinessOutcome({
      state: 'target-required',
      code: 'admission.target_unavailable',
      requirementId: 'model',
      slot: 'model',
    })

    expect(outcome).toMatchObject({ requirementId: 'model', slot: 'model' })
    expect(runReadinessMessage(outcome)).toContain('AI 模型')
    expect(runReadinessMessage(outcome)).toContain('model')
    expect(runReadinessMessage(outcome)).not.toContain('所需目标')
  })
})
