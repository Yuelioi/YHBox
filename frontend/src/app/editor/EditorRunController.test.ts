import { describe, expect, it, vi } from 'vitest'
import type { CompileView, DebugSnapshot, RunView } from '@/app/transport/workflow'
import {
  createEditorRunController,
  type EditorRunControllerDependencies,
  type EditorRunSession,
} from './EditorRunController'

function harness(overrides: Partial<EditorRunSession> = {}) {
  const session: EditorRunSession = {
    diagnostics: [],
    validate: vi.fn(async () => ({ diagnostics: [] }) as CompileView),
    save: vi.fn(async () => undefined),
    run: vi.fn(async () => ({ runId: 'run-1' }) as RunView),
    startDebug: vi.fn(async () => ({ runId: 'debug-1' }) as RunView),
    controlDebug: vi.fn(async () => null),
    cancelRun: vi.fn(async () => null),
    refreshRun: vi.fn(async () => null),
    loadTimelinePage: vi.fn(async () => null),
    ...overrides,
  }
  const openWorkbench = vi.fn()
  const showError = vi.fn()
  const focusDebugNode = vi.fn(async () => undefined)
  const dependencies: EditorRunControllerDependencies = {
    session,
    translate: (key) => `translated:${key}`,
    showError,
    openWorkbench,
    focusDebugNode,
  }
  return {
    session,
    openWorkbench,
    showError,
    focusDebugNode,
    controller: createEditorRunController(dependencies),
  }
}

describe('editor run controller', () => {
  it('owns compile, start, and result-panel routing behind one command interface', async () => {
    const run = harness()

    await expect(run.controller.execute({ kind: 'compile' })).resolves.toEqual({ ok: true })
    expect(run.controller.compileSucceeded.value).toBe(true)
    await expect(run.controller.execute({ kind: 'start' })).resolves.toEqual({ ok: true })
    expect(run.openWorkbench).toHaveBeenLastCalledWith('timeline')
  })

  it('routes compiler-declared diagnostics without pretending a Run started', async () => {
    const run = harness({
      diagnostics: [{}],
      run: vi.fn(async () => null),
    })

    await expect(run.controller.execute({ kind: 'start' })).resolves.toEqual({ ok: false })
    expect(run.openWorkbench).toHaveBeenCalledWith('diagnostics')
    expect(run.showError).not.toHaveBeenCalled()
  })

  it('opens debug context and focuses the paused node from the authoritative snapshot', async () => {
    const snapshot = {
      status: 'paused',
      graphPath: ['main', 'child'],
      nodeId: 'node-2',
    } as DebugSnapshot
    const run = harness({ debugSnapshot: snapshot })

    await expect(
      run.controller.execute({
        kind: 'start-debug',
        breakpoints: [{ graphId: 'child', nodeId: 'node-2' }],
      }),
    ).resolves.toEqual({ ok: true })
    expect(run.openWorkbench).toHaveBeenCalledWith('debug')
    expect(run.focusDebugNode).toHaveBeenCalledWith(['main', 'child'], 'node-2')
  })

  it('normalizes command failures and keeps save success observable', async () => {
    const failure = new Error('disk unavailable')
    const run = harness({
      save: vi.fn(async () => {
        throw failure
      }),
    })

    await expect(run.controller.execute({ kind: 'save' })).resolves.toEqual({ ok: false })
    expect(run.controller.saveSucceeded.value).toBe(false)
    expect(run.showError).toHaveBeenCalledWith('translated:workflow.toast.save_failed', failure)
  })
})
