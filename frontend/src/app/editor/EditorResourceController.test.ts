import { describe, expect, it, vi } from 'vitest'
import type { WorkflowResource } from '../../../../contracts/workflow/current/workflow-source'
import type { MacroAsset } from '@/lib/backend'
import { createEditorResourceController, type EditorResourcePort } from './EditorResourceController'

const macroResource: WorkflowResource = {
  id: 'workflow-macro',
  kind: 'macro',
  name: 'Recorded steps',
  description: '',
  category: '',
  tags: [],
  macro: {
    actionCount: 0,
    baseResolution: [1920, 1080],
    durationUs: 0,
    blob: { mediaType: 'application/json', digest: 'sha256:macro', size: 10 },
  },
}

const macroAsset: MacroAsset = {
  id: 'global-macro',
  label: 'Global steps',
  description: '',
  category: '',
  tags: [],
  createdAt: '2026-07-26T00:00:00Z',
  document: { schemaVersion: 1, baseResolution: [1920, 1080], actions: [] },
  blob: { mediaType: 'application/json', digest: 'sha256:global', size: 10 },
}

function setup(overrides: Partial<EditorResourcePort> = {}) {
  const port: EditorResourcePort = {
    openWorkflow: vi.fn(async () => ({
      kind: 'macro' as const,
      macro: macroAsset.document,
    })),
    rewriteWorkflow: vi.fn(async (resource) => ({ ...resource, name: 'Updated' })),
    getMacro: vi.fn(async () => macroAsset),
    saveMacro: vi.fn(async (asset) => asset),
    ...overrides,
  }
  const replaceWorkflowResource = vi.fn()
  const invalidateAssets = vi.fn()
  const errors: string[] = []
  const controller = createEditorResourceController({
    port,
    replaceWorkflowResource,
    invalidateAssets,
    translate: (key) => key,
    showError: (title) => errors.push(title),
  })
  return { controller, port, replaceWorkflowResource, invalidateAssets, errors }
}

describe('EditorResourceController', () => {
  it('opens and rewrites a workflow macro through one resource seam', async () => {
    const { controller, port, replaceWorkflowResource } = setup()

    expect(await controller.execute({ kind: 'open-workflow', resource: macroResource })).toBe(true)
    controller.workflowMacroEditing.value!.document.baseResolution[0] = 1280
    controller.workflowMacroEditing.value!.resource.name = 'Unified macro'
    controller.workflowMacroEditing.value!.resource.category = 'Automation'
    controller.workflowMacroEditing.value!.resource.tags = ['game']
    expect(await controller.execute({ kind: 'save-workflow-macro' })).toBe(true)

    expect(port.rewriteWorkflow).toHaveBeenCalledWith(
      expect.objectContaining({
        ...macroResource,
        name: 'Unified macro',
        category: 'Automation',
        tags: ['game'],
      }),
      expect.objectContaining({
        kind: 'macro-document',
        macro: { document: expect.objectContaining({ baseResolution: [1280, 1080] }) },
      }),
    )
    expect(replaceWorkflowResource).toHaveBeenCalledWith(
      macroResource.id,
      expect.objectContaining({ name: 'Updated' }),
    )
    expect(controller.workflowMacroEditing.value).toBeNull()
  })

  it('keeps an editor open and reports a stable error when persistence fails', async () => {
    const { controller, errors } = setup({
      rewriteWorkflow: vi.fn(async () => {
        throw new Error('disk unavailable')
      }),
    })

    await controller.execute({ kind: 'open-workflow', resource: macroResource })
    expect(await controller.execute({ kind: 'save-workflow-macro' })).toBe(false)

    expect(controller.workflowMacroEditing.value).not.toBeNull()
    expect(controller.workflowResourceEditBusy.value).toBe(false)
    expect(errors).toEqual(['workflow.resources.save_content_failed'])
  })

  it('invalidates the shared asset list after saving a global macro', async () => {
    const { controller, invalidateAssets } = setup()

    await controller.execute({
      kind: 'open-global-macro',
      asset: { guid: macroAsset.id } as never,
    })
    expect(await controller.execute({ kind: 'save-global-macro' })).toBe(true)

    expect(invalidateAssets).toHaveBeenCalledOnce()
    expect(controller.macroEditing.value).toBeNull()
  })
})
