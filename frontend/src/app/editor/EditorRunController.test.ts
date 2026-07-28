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
    check: vi.fn(async () => ({ diagnostics: [] }) as CompileView),
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
  const showSuccess = vi.fn()
  const focusDebugNode = vi.fn(async () => undefined)
  const dependencies: EditorRunControllerDependencies = {
    session,
    translate: (key) => `translated:${key}`,
    showError,
    showSuccess,
    openWorkbench,
    focusDebugNode,
  }
  return {
    session,
    openWorkbench,
    showError,
    showSuccess,
    focusDebugNode,
    controller: createEditorRunController(dependencies),
  }
}

describe('editor run controller', () => {
  it('owns compile, start, and result-panel routing behind one command interface', async () => {
    const run = harness()

    await expect(run.controller.execute({ kind: 'check-workflow' })).resolves.toEqual({ ok: true })
    expect(run.showSuccess).toHaveBeenCalledWith('translated:workflow.toast.check_succeeded')
    await expect(run.controller.execute({ kind: 'start' })).resolves.toEqual({ ok: true })
    expect(run.openWorkbench).toHaveBeenLastCalledWith('timeline')
  })

  it('routes check errors without pretending a Run started', async () => {
    const run = harness({
      diagnostics: [{}],
      run: vi.fn(async () => null),
    })

    await expect(run.controller.execute({ kind: 'start' })).resolves.toEqual({ ok: false })
    expect(run.openWorkbench).toHaveBeenCalledWith('diagnostics')
    expect(run.showError).not.toHaveBeenCalled()
  })

  it('opens workflow issues when checking finds a non-blocking warning', async () => {
    const run = harness({
      check: vi.fn(
        async () =>
          ({
            diagnostics: [{ severity: 'warning', code: 'MISSING_INPUT_BINDING' }],
          }) as CompileView,
      ),
    })

    await expect(run.controller.execute({ kind: 'check-workflow' })).resolves.toEqual({ ok: true })
    expect(run.openWorkbench).toHaveBeenCalledWith('diagnostics')
    expect(run.showSuccess).not.toHaveBeenCalled()
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

  it('does not duplicate a persistent save error with a toast', async () => {
    const failure = new Error('INVALID_FIELD')
    const run = harness({
      saveError: '请检查节点参数或连线',
      save: vi.fn(async () => {
        throw failure
      }),
      run: vi.fn(async () => {
        throw failure
      }),
    })

    await expect(run.controller.execute({ kind: 'save' })).resolves.toEqual({ ok: false })
    await expect(run.controller.execute({ kind: 'start' })).resolves.toEqual({ ok: false })
    expect(run.showError).not.toHaveBeenCalled()
  })
})
