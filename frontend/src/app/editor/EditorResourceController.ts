import { computed, ref, type ComputedRef, type Ref } from 'vue'
import type { WorkflowResource } from '../../../../contracts/workflow/current/workflow-source'
import type {
  AssetSummary,
  MacroAsset,
  MacroDocument,
  WorkflowResourceContent,
  WorkflowResourceEdit,
} from '@/lib/backend'
import type { RecordingPreview } from '@/stores/recording'

export interface EditorResourcePort {
  openWorkflow(resource: WorkflowResource): Promise<WorkflowResourceContent>
  rewriteWorkflow(resource: WorkflowResource, edit: WorkflowResourceEdit): Promise<WorkflowResource>
  getMacro(id: string): Promise<MacroAsset | null>
  saveMacro(asset: MacroAsset): Promise<MacroAsset>
}

export type EditorResourceCommand =
  | { kind: 'open-workflow'; resource: WorkflowResource }
  | { kind: 'save-workflow-macro' }
  | { kind: 'save-workflow-clip' }
  | { kind: 'open-global-macro'; asset: AssetSummary }
  | { kind: 'save-global-macro' }

export interface EditorResourceController {
  macroEditing: Ref<MacroAsset | null>
  macroEditBusy: Ref<boolean>
  macroEditValid: Ref<boolean>
  workflowMacroEditing: Ref<{
    resource: WorkflowResource
    document: MacroDocument
  } | null>
  workflowMacroEditValid: Ref<boolean>
  workflowClipEditing: Ref<{
    resource: WorkflowResource
    content: NonNullable<WorkflowResourceContent['inputClip']>
  } | null>
  workflowClipTrimStartUs: Ref<number>
  workflowClipTrimEndUs: Ref<number>
  workflowResourceEditBusy: Ref<boolean>
  workflowClipPreview: ComputedRef<RecordingPreview | null>
  workflowClipTrimChanged: ComputedRef<boolean>
  execute(command: EditorResourceCommand): Promise<boolean>
}

interface EditorResourceControllerOptions {
  port: EditorResourcePort
  replaceWorkflowResource(resourceId: string, resource: WorkflowResource): void
  invalidateAssets(): void
  translate(key: string): string
  showError(title: string, error: unknown): void
}

export function createEditorResourceController(
  options: EditorResourceControllerOptions,
): EditorResourceController {
  const macroEditing = ref<MacroAsset | null>(null)
  const macroEditBusy = ref(false)
  const macroEditValid = ref(true)
  const workflowMacroEditing = ref<{
    resource: WorkflowResource
    document: MacroDocument
  } | null>(null)
  const workflowMacroEditValid = ref(true)
  const workflowClipEditing = ref<{
    resource: WorkflowResource
    content: NonNullable<WorkflowResourceContent['inputClip']>
  } | null>(null)
  const workflowClipTrimStartUs = ref(0)
  const workflowClipTrimEndUs = ref(0)
  const workflowResourceEditBusy = ref(false)

  const workflowClipPreview = computed<RecordingPreview | null>(() => {
    const clip = workflowClipEditing.value?.content
    if (!clip) return null
    const counts = Object.fromEntries(clip.tracks.map((track) => [track.kind, track.count]))
    return {
      mode: 'precise',
      durationUs: clip.durationUs,
      eventCount: clip.eventCount,
      keyActions: counts.keyboard ?? 0,
      clickActions: counts['mouse-buttons'] ?? 0,
      pointerMoves: counts['absolute-motion'] ?? 0,
      rawDeltas: counts['relative-motion'] ?? 0,
      scrollActions: counts.scroll ?? 0,
      steps: [],
      tracks: clip.tracks,
    }
  })
  const workflowClipTrimChanged = computed(() => {
    const clip = workflowClipEditing.value?.content
    return Boolean(
      clip && (workflowClipTrimStartUs.value > 0 || workflowClipTrimEndUs.value < clip.durationUs),
    )
  })

  async function openWorkflow(resource: WorkflowResource): Promise<boolean> {
    try {
      const content = await options.port.openWorkflow(copy(resource))
      if (resource.kind === 'macro' && content.macro) {
        workflowMacroEditing.value = {
          resource: copy(resource),
          document: cloneMacroDocument(content.macro),
        }
        workflowMacroEditValid.value = true
        return true
      }
      if (resource.kind === 'input-clip' && content.inputClip) {
        workflowClipEditing.value = {
          resource: copy(resource),
          content: copy(content.inputClip),
        }
        workflowClipTrimStartUs.value = 0
        workflowClipTrimEndUs.value = content.inputClip.durationUs
        return true
      }
      throw new Error(`resource ${resource.id} has no editable content`)
    } catch (error) {
      options.showError(options.translate('workflow.resources.load_content_failed'), error)
      return false
    }
  }

  async function saveWorkflowMacro(): Promise<boolean> {
    const editing = workflowMacroEditing.value
    if (!editing || !workflowMacroEditValid.value || !editing.resource.name.trim()) return false
    workflowResourceEditBusy.value = true
    try {
      const updated = await options.port.rewriteWorkflow(copy(editing.resource), {
        kind: 'macro-document',
        macro: { document: cloneMacroDocument(editing.document) },
      })
      options.replaceWorkflowResource(editing.resource.id, copy(updated))
      workflowMacroEditing.value = null
      return true
    } catch (error) {
      options.showError(options.translate('workflow.resources.save_content_failed'), error)
      return false
    } finally {
      workflowResourceEditBusy.value = false
    }
  }

  async function saveWorkflowClip(): Promise<boolean> {
    const editing = workflowClipEditing.value
    if (!editing || !workflowClipTrimChanged.value) return false
    workflowResourceEditBusy.value = true
    try {
      const updated = await options.port.rewriteWorkflow(copy(editing.resource), {
        kind: 'input-clip-trim',
        inputClip: {
          trimStartUs: workflowClipTrimStartUs.value,
          trimEndUs: workflowClipTrimEndUs.value,
        },
      })
      options.replaceWorkflowResource(editing.resource.id, copy(updated))
      workflowClipEditing.value = null
      return true
    } catch (error) {
      options.showError(options.translate('workflow.resources.save_content_failed'), error)
      return false
    } finally {
      workflowResourceEditBusy.value = false
    }
  }

  async function openGlobalMacro(asset: AssetSummary): Promise<boolean> {
    try {
      const value = await options.port.getMacro(asset.guid)
      if (!value) throw new Error(`macro ${asset.guid} not found`)
      macroEditing.value = {
        ...value,
        tags: [...(value.tags ?? [])],
        document: cloneMacroDocument(value.document),
        blob: { ...value.blob },
      }
      macroEditValid.value = true
      return true
    } catch (error) {
      options.showError(options.translate('assets.macros.load_failed'), error)
      return false
    }
  }

  async function saveGlobalMacro(): Promise<boolean> {
    if (!macroEditing.value || !macroEditValid.value) return false
    macroEditBusy.value = true
    try {
      await options.port.saveMacro({
        ...macroEditing.value,
        document: cloneMacroDocument(macroEditing.value.document),
      })
      macroEditing.value = null
      options.invalidateAssets()
      return true
    } catch (error) {
      options.showError(options.translate('assets.macros.save_failed'), error)
      return false
    } finally {
      macroEditBusy.value = false
    }
  }

  async function execute(command: EditorResourceCommand): Promise<boolean> {
    switch (command.kind) {
      case 'open-workflow':
        return openWorkflow(command.resource)
      case 'save-workflow-macro':
        return saveWorkflowMacro()
      case 'save-workflow-clip':
        return saveWorkflowClip()
      case 'open-global-macro':
        return openGlobalMacro(command.asset)
      case 'save-global-macro':
        return saveGlobalMacro()
    }
  }

  return {
    macroEditing,
    macroEditBusy,
    macroEditValid,
    workflowMacroEditing,
    workflowMacroEditValid,
    workflowClipEditing,
    workflowClipTrimStartUs,
    workflowClipTrimEndUs,
    workflowResourceEditBusy,
    workflowClipPreview,
    workflowClipTrimChanged,
    execute,
  }
}

function cloneMacroDocument(document: MacroDocument): MacroDocument {
  return {
    ...document,
    baseResolution: [...document.baseResolution] as [number, number],
    actions: document.actions.map((action) => ({
      ...action,
      point: action.point ? { ...action.point } : undefined,
    })),
  }
}

function copy<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}
