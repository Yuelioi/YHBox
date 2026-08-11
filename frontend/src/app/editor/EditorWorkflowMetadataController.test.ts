import { describe, expect, it, vi } from 'vitest'
import {
  createEditorWorkflowMetadataController,
  type EditorWorkflowMetadataControllerOptions,
  type EditorWorkflowMetadataPort,
  type EditorWorkflowMetadataSession,
} from './EditorWorkflowMetadataController'

function harness(
  overrides: {
    session?: Partial<EditorWorkflowMetadataSession>
    port?: Partial<EditorWorkflowMetadataPort>
    saveCurrent?: () => Promise<boolean>
  } = {},
) {
  const session: EditorWorkflowMetadataSession = {
    source: { workflow: { name: 'Workflow' } },
    workflowId: 'workflow-1',
    baseRevision: 7,
    dirty: false,
    load: vi.fn(async () => undefined),
    ...overrides.session,
  }
  const port: EditorWorkflowMetadataPort = {
    getSource: vi.fn(async () => ({
      name: 'Durable name',
      description: 'Description',
      category: 'Automation',
      tags: ['game'],
    })),
    updateSourceMetadata: vi.fn(async () => undefined),
    ...overrides.port,
  }
  const options: EditorWorkflowMetadataControllerOptions = {
    session,
    port,
    saveCurrent: overrides.saveCurrent ?? vi.fn(async () => true),
    translate: (key) => `translated:${key}`,
    describeError: (error) => (error instanceof Error ? error.message : String(error)),
  }
  return { session, port, options, controller: createEditorWorkflowMetadataController(options) }
}

describe('editor workflow metadata controller', () => {
  it('loads durable metadata while opening the dialog', async () => {
    const value = harness()

    await expect(value.controller.open()).resolves.toBe(true)
    expect(value.port.getSource).toHaveBeenCalledWith('workflow-1')
    expect(value.controller.dialogOpen.value).toBe(true)
    expect(value.controller.metadata).toMatchObject({
      name: 'Durable name',
      description: 'Description',
      category: 'Automation',
      tags: ['game'],
    })
    expect(value.controller.busy.value).toBe(false)
  })

  it('saves pending graph edits before publishing metadata at the current revision', async () => {
    const saveCurrent = vi.fn(async () => true)
    const value = harness({ session: { dirty: true }, saveCurrent })
    const draft = { name: 'Renamed', description: '', category: '', tags: [] }

    await value.controller.open()
    await expect(value.controller.save(draft)).resolves.toBe(true)
    expect(saveCurrent).toHaveBeenCalledOnce()
    expect(value.port.updateSourceMetadata).toHaveBeenCalledWith('workflow-1', 7, draft)
    expect(value.session.load).toHaveBeenCalledWith('workflow-1')
    expect(value.controller.dialogOpen.value).toBe(false)
  })

  it('keeps the dialog open when pending graph edits cannot be saved', async () => {
    const value = harness({ session: { dirty: true }, saveCurrent: vi.fn(async () => false) })
    await value.controller.open()

    await expect(
      value.controller.save({ name: 'Renamed', description: '', category: '', tags: [] }),
    ).resolves.toBe(false)
    expect(value.port.updateSourceMetadata).not.toHaveBeenCalled()
    expect(value.controller.error.value).toBe('translated:workflow.editor.metadata_save_blocked')
    expect(value.controller.dialogOpen.value).toBe(true)
  })
})
