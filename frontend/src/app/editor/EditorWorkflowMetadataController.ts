import { reactive, ref } from 'vue'

export interface WorkflowMetadataDraft {
  name: string
  description: string
  category: string
  tags: string[]
}

export interface EditorWorkflowMetadataSession {
  source: { workflow: { name: string } } | null
  readonly workflowId: string
  baseRevision: number
  dirty: boolean
  load(workflowId: string): Promise<void>
}

export interface EditorWorkflowMetadataPort {
  getSource(workflowId: string): Promise<Partial<WorkflowMetadataDraft>>
  updateSourceMetadata(
    workflowId: string,
    baseRevision: number,
    draft: WorkflowMetadataDraft,
  ): Promise<unknown>
}

export interface EditorWorkflowMetadataControllerOptions {
  session: EditorWorkflowMetadataSession
  port: EditorWorkflowMetadataPort
  saveCurrent: () => Promise<boolean>
  translate: (key: string) => string
  describeError: (error: unknown) => string
}

export function createEditorWorkflowMetadataController(
  options: EditorWorkflowMetadataControllerOptions,
) {
  const dialogOpen = ref(false)
  const busy = ref(false)
  const error = ref('')
  const metadata = reactive<WorkflowMetadataDraft>({
    name: '',
    description: '',
    category: '',
    tags: [],
  })

  async function open(): Promise<boolean> {
    const source = options.session.source
    if (!source || busy.value) return false
    assignMetadata({ name: source.workflow.name })
    error.value = ''
    dialogOpen.value = true
    busy.value = true
    try {
      const durable = await options.port.getSource(options.session.workflowId)
      assignMetadata({ ...durable, name: durable.name || source.workflow.name })
      return true
    } catch (cause) {
      error.value = options.describeError(cause)
      return false
    } finally {
      busy.value = false
    }
  }

  async function save(draft: WorkflowMetadataDraft): Promise<boolean> {
    if (busy.value || !options.session.workflowId) return false
    busy.value = true
    error.value = ''
    try {
      if (options.session.dirty && !(await options.saveCurrent())) {
        error.value = options.translate('workflow.editor.metadata_save_blocked')
        return false
      }
      const workflowId = options.session.workflowId
      await options.port.updateSourceMetadata(workflowId, options.session.baseRevision, draft)
      await options.session.load(workflowId)
      assignMetadata(draft)
      dialogOpen.value = false
      return true
    } catch (cause) {
      error.value = options.describeError(cause)
      return false
    } finally {
      busy.value = false
    }
  }

  function assignMetadata(value: Partial<WorkflowMetadataDraft>): void {
    metadata.name = value.name ?? ''
    metadata.description = value.description ?? ''
    metadata.category = value.category ?? ''
    metadata.tags = [...(value.tags ?? [])]
  }

  return { dialogOpen, busy, error, metadata, open, save }
}
