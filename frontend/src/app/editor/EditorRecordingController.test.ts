import { describe, expect, it, vi } from 'vitest'
import type { WorkflowResource } from '../../../../contracts/workflow/current/workflow-source'
import type { RecordingStopPayload } from '@/stores/recording'
import {
  createEditorRecordingController,
  type EditorRecordingPort,
  type EditorRecordingSnapshot,
} from './EditorRecordingController'

const resource: WorkflowResource = {
  id: 'recorded-macro',
  kind: 'macro',
  name: 'Recorded steps',
  description: '',
  category: '',
  tags: [],
  macro: {
    actionCount: 1,
    baseResolution: [1920, 1080],
    durationUs: 10,
    blob: { mediaType: 'application/json', digest: 'sha256:macro', size: 10 },
  },
}

const pending: RecordingStopPayload = {
  pendingID: 'pending-1',
  targetSlot: 'desktop',
  mode: 'simple',
  durationUs: 1_000_000,
  eventCount: 1,
  preview: {
    mode: 'simple',
    durationUs: 1_000_000,
    eventCount: 1,
    keyActions: 0,
    clickActions: 1,
    pointerMoves: 0,
    rawDeltas: 0,
    scrollActions: 0,
    steps: [],
    tracks: [],
  },
  document: {
    schemaVersion: 2,
    baseResolution: [1920, 1080],
    meta: { autoMove: { enabled: true, mode: 'linear', durationMs: 300 } },
    actions: [
      {
        id: 'click-1',
        kind: 'click',
        durationUs: 1,
        button: 'left',
        point: { x: 0.5, y: 0.5, unit: 'ratio' },
      },
    ],
  },
  environment: {
    baseResolution: [1920, 1080],
    mouseMode: 'absolute',
    mouseCounts360: 0,
  },
}

function setup(overrides: Partial<EditorRecordingPort> = {}) {
  let snapshot: EditorRecordingSnapshot = {
    phase: 'idle',
    pending: null,
    invocation: null,
  }
  const port: EditorRecordingPort = {
    start: vi.fn(async () => true),
    pause: vi.fn(async () => undefined),
    resume: vi.fn(async () => undefined),
    stop: vi.fn(async () => pending),
    cancel: vi.fn(async () => undefined),
    discard: vi.fn(async () => undefined),
    finalize: vi.fn(async () => ({
      destination: 'workflow-resource' as const,
      targetSlot: 'desktop',
      resource,
    })),
    claimInvocation: vi.fn(),
    queryFacets: vi.fn(async () => ({
      categories: ['Automation'],
      tags: ['Recorded'],
    })),
    ...overrides,
  }
  const imported: WorkflowResource[] = []
  const errors: string[] = []
  const startErrors: string[] = []
  const controller = createEditorRecordingController({
    port,
    snapshot: () => snapshot,
    targets: () => [
      { label: 'Desktop', value: 'desktop' },
      { label: 'Browser', value: 'browser' },
    ],
    selectedTargetSlot: () => 'browser',
    importResource: (value) => imported.push(value),
    translate: (key) => key,
    showError: (title) => errors.push(title),
    showStartError: (title) => startErrors.push(title),
  })
  return {
    controller,
    port,
    imported,
    errors,
    startErrors,
    setSnapshot(value: EditorRecordingSnapshot) {
      snapshot = value
    },
  }
}

describe('EditorRecordingController', () => {
  it('selects the current target and starts through one recording seam', async () => {
    const { controller, port } = setup()

    expect(await controller.execute({ kind: 'open-start', mode: 'precise' })).toBe(true)
    expect(controller.state.targetSlot).toBe('browser')
    expect(controller.state.startOpen).toBe(true)
    expect(await controller.execute({ kind: 'start' })).toBe(true)

    expect(port.start).toHaveBeenCalledWith('precise', 'browser')
    expect(controller.state.startOpen).toBe(false)
    expect(controller.state.controlBusy).toBe(false)
  })

  it('claims an editor recording once and imports the finalized workflow resource', async () => {
    const { controller, port, imported, setSnapshot } = setup()
    setSnapshot({ phase: 'pending', pending, invocation: null })

    expect(
      await controller.execute({
        kind: 'sync-pending',
        editorActive: true,
        editorRoute: true,
      }),
    ).toBe(true)
    expect(port.claimInvocation).toHaveBeenCalledWith('editor')
    expect(controller.state.facetCategories).toEqual(['Automation'])

    controller.state.draft.name = '  Login steps  '
    controller.state.draft.tags = [' Recorded ', 'recorded', 'smoke']
    expect(await controller.execute({ kind: 'finalize' })).toBe(true)

    expect(port.finalize).toHaveBeenCalledWith(
      expect.objectContaining({
        destination: 'workflow-resource',
        label: 'Login steps',
        tags: ['Recorded', 'smoke'],
        document: pending.document,
      }),
    )
    expect(imported).toEqual([resource])
    expect(controller.state.pending).toBeNull()
  })

  it('preserves the pending preview when persistence fails so saving can be retried', async () => {
    const { controller, errors, setSnapshot } = setup({
      finalize: vi.fn(async () => {
        throw new Error('blob store unavailable')
      }),
    })
    setSnapshot({ phase: 'pending', pending, invocation: 'editor' })
    await controller.execute({ kind: 'sync-pending', editorActive: true, editorRoute: true })
    controller.state.draft.name = 'Login steps'

    expect(await controller.execute({ kind: 'finalize' })).toBe(false)

    expect(controller.state.pending?.pendingID).toBe(pending.pendingID)
    expect(controller.state.draft.name).toBe('Login steps')
    expect(controller.state.saveBusy).toBe(false)
    expect(errors).toEqual(['recordingSave.save_failed'])
  })

  it('keeps the pending preview when discard fails', async () => {
    const { controller, errors, setSnapshot } = setup({
      discard: vi.fn(async () => {
        throw new Error('disk unavailable')
      }),
    })
    setSnapshot({ phase: 'pending', pending, invocation: 'editor' })
    await controller.execute({ kind: 'sync-pending', editorActive: true, editorRoute: true })

    expect(await controller.execute({ kind: 'discard' })).toBe(false)
    expect(controller.state.pending?.pendingID).toBe(pending.pendingID)
    expect(errors).toEqual(['recordingSave.discard_failed'])
  })
})
