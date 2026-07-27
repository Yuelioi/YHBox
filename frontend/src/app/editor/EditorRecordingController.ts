import { reactive } from 'vue'
import type { WorkflowResource } from '../../../../contracts/workflow/current/workflow-source'
import type {
  MacroAction,
  RecordingFinalizePayload,
  RecordingInvocation,
  RecordingMode,
  RecordingState,
  RecordingStopPayload,
} from '@/stores/recording'

export interface EditorRecordingPort {
  start(mode: RecordingMode, targetSlot: string): Promise<boolean>
  pause(): Promise<void>
  resume(): Promise<void>
  stop(): Promise<RecordingStopPayload | null>
  cancel(): Promise<void>
  discard(pendingID: string): Promise<void>
  finalize(input: {
    pendingID: string
    destination: 'workflow-resource'
    label: string
    description: string
    category: string
    tags: string[]
    actions?: MacroAction[]
    trimStartUs?: number
    trimEndUs?: number
  }): Promise<RecordingFinalizePayload>
  claimInvocation(origin: 'editor'): void
  queryFacets(kind: 'macro' | 'clip'): Promise<{ categories: string[]; tags: string[] }>
}

export interface EditorRecordingSnapshot {
  phase: RecordingState['phase']
  pending: RecordingStopPayload | null
  invocation: RecordingInvocation | null
}

export interface EditorRecordingTarget {
  label: string
  value: string
}

export type EditorRecordingCommand =
  | { kind: 'open-start'; mode: RecordingMode }
  | { kind: 'start' }
  | { kind: 'pause' }
  | { kind: 'resume' }
  | { kind: 'stop' }
  | { kind: 'cancel' }
  | { kind: 'discard' }
  | { kind: 'finalize' }
  | { kind: 'sync-pending'; editorActive: boolean; editorRoute: boolean }

export interface EditorRecordingState {
  startOpen: boolean
  targetSlot: string
  mode: RecordingMode
  controlBusy: boolean
  pending: RecordingStopPayload | null
  saveBusy: boolean
  actions: MacroAction[]
  actionsValid: boolean
  trimStartUs: number
  trimEndUs: number
  draft: {
    name: string
    description: string
    category: string
    tags: string[]
  }
  facetCategories: string[]
  facetTags: string[]
}

export interface EditorRecordingController {
  state: EditorRecordingState
  execute(command: EditorRecordingCommand): Promise<boolean>
}

interface EditorRecordingControllerOptions {
  port: EditorRecordingPort
  snapshot(): EditorRecordingSnapshot
  targets(): EditorRecordingTarget[]
  selectedTargetSlot(): unknown
  importResource(resource: WorkflowResource): void
  translate(key: string): string
  showError(title: string, error: unknown): void
  showStartError(title: string, error: unknown): void
}

export function createEditorRecordingController(
  options: EditorRecordingControllerOptions,
): EditorRecordingController {
  const state = reactive<EditorRecordingState>({
    startOpen: false,
    targetSlot: '',
    mode: 'simple',
    controlBusy: false,
    pending: null,
    saveBusy: false,
    actions: [],
    actionsValid: true,
    trimStartUs: 0,
    trimEndUs: 0,
    draft: {
      name: '',
      description: '',
      category: '',
      tags: [],
    },
    facetCategories: [],
    facetTags: [],
  })

  async function openStart(mode: RecordingMode): Promise<boolean> {
    const snapshot = options.snapshot()
    if (snapshot.phase !== 'idle') {
      if (snapshot.pending) await openPreview(snapshot.pending)
      return Boolean(snapshot.pending)
    }
    const targets = options.targets()
    if (!targets.length) {
      options.showError(
        options.translate('workflow.recording.start_failed'),
        options.translate('workflow.inspector.no_installed_target'),
      )
      return false
    }
    const selectedSlot = options.selectedTargetSlot()
    state.mode = mode
    state.targetSlot =
      typeof selectedSlot === 'string' && targets.some((target) => target.value === selectedSlot)
        ? selectedSlot
        : state.targetSlot || targets[0]?.value || ''
    state.startOpen = true
    return true
  }

  async function start(): Promise<boolean> {
    if (!state.targetSlot) return false
    state.controlBusy = true
    try {
      const started = await options.port.start(state.mode, state.targetSlot)
      if (started) state.startOpen = false
      return started
    } catch (error) {
      options.showStartError(options.translate('workflow.recording.start_failed'), error)
      return false
    } finally {
      state.controlBusy = false
    }
  }

  async function control(action: 'pause' | 'resume' | 'stop'): Promise<boolean> {
    try {
      await options.port[action]()
      return true
    } catch (error) {
      options.showError(options.translate('workflow.recording.control_failed'), error)
      return false
    }
  }

  async function cancel(): Promise<boolean> {
    try {
      await options.port.cancel()
      return true
    } catch (error) {
      options.showError(options.translate('workflow.recording.control_failed'), error)
      return false
    }
  }

  async function syncPending(editorActive: boolean, editorRoute: boolean): Promise<boolean> {
    const snapshot = options.snapshot()
    if (!snapshot.pending) {
      if (snapshot.phase === 'idle') state.pending = null
      return false
    }
    if (!editorActive || !editorRoute || (snapshot.invocation && snapshot.invocation !== 'editor'))
      return false
    options.port.claimInvocation('editor')
    await openPreview(snapshot.pending)
    return true
  }

  async function openPreview(payload: RecordingStopPayload): Promise<void> {
    if (state.pending?.pendingID === payload.pendingID) return
    state.pending = payload
    state.actions = cloneActions(payload.actions ?? [])
    state.actionsValid = true
    state.trimStartUs = 0
    state.trimEndUs = payload.durationUs
    state.draft.name = ''
    state.draft.description = ''
    state.draft.category = ''
    state.draft.tags = []
    const pendingID = payload.pendingID
    try {
      const facets = await options.port.queryFacets(payload.mode === 'simple' ? 'macro' : 'clip')
      if (state.pending?.pendingID !== pendingID) return
      state.facetCategories = facets.categories
      state.facetTags = facets.tags
    } catch {
      if (state.pending?.pendingID !== pendingID) return
      state.facetCategories = []
      state.facetTags = []
    }
  }

  async function finalize(): Promise<boolean> {
    const pending = state.pending
    const name = state.draft.name.trim()
    if (!pending || !name || (pending.mode === 'simple' && !state.actionsValid)) return false
    state.saveBusy = true
    try {
      const saved = await options.port.finalize({
        pendingID: pending.pendingID,
        destination: 'workflow-resource',
        label: name,
        description: state.draft.description.trim(),
        category: state.draft.category.trim(),
        tags: uniqueStrings(state.draft.tags),
        actions: pending.actions ? cloneActions(state.actions) : undefined,
        trimStartUs: pending.mode === 'precise' ? state.trimStartUs : undefined,
        trimEndUs: pending.mode === 'precise' ? state.trimEndUs : undefined,
      })
      if (saved.destination !== 'workflow-resource') {
        throw new Error('recording finalize returned the wrong destination')
      }
      options.importResource(saved.resource)
      state.pending = null
      return true
    } catch (error) {
      options.showError(options.translate('recordingSave.save_failed'), error)
      return false
    } finally {
      state.saveBusy = false
    }
  }

  async function discard(): Promise<boolean> {
    const pending = state.pending
    if (!pending) return false
    state.saveBusy = true
    try {
      await options.port.discard(pending.pendingID)
      state.pending = null
      return true
    } catch (error) {
      options.showError(options.translate('recordingSave.discard_failed'), error)
      return false
    } finally {
      state.saveBusy = false
    }
  }

  async function execute(command: EditorRecordingCommand): Promise<boolean> {
    switch (command.kind) {
      case 'open-start':
        return openStart(command.mode)
      case 'start':
        return start()
      case 'pause':
      case 'resume':
      case 'stop':
        return control(command.kind)
      case 'cancel':
        return cancel()
      case 'discard':
        return discard()
      case 'finalize':
        return finalize()
      case 'sync-pending':
        return syncPending(command.editorActive, command.editorRoute)
    }
  }

  return { state, execute }
}

export function formatRecordingDuration(durationUs: number): string {
  const seconds = Math.max(0, Math.round(durationUs / 1_000_000))
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`
}

function cloneActions(actions: MacroAction[]): MacroAction[] {
  return actions.map((action) => ({
    ...action,
    point: action.point ? { ...action.point } : undefined,
  }))
}

function uniqueStrings(values: string[]): string[] {
  const seen = new Set<string>()
  return values
    .map((value) => value.trim())
    .filter((value) => {
      const key = value.toLocaleLowerCase()
      if (!key || seen.has(key)) return false
      seen.add(key)
      return true
    })
}
