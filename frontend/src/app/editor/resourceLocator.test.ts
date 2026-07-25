import { describe, expect, it } from 'vitest'
import type { WorkflowResource } from '../../../../contracts/workflow/current/workflow-source'
import { resolveWorkflowResourceBinding, workspaceResourceKind } from './resourceLocator'

const image: WorkflowResource = {
  id: 'template',
  kind: 'image',
  name: '模板',
  image: {
    variants: [
      {
        id: '720p',
        resolution: [1280, 720],
        bbox: [0, 0, 64, 32],
        blob: {
          mediaType: 'image/png',
          digest: `sha256:${'1'.repeat(64)}`,
          size: 128,
        },
      },
    ],
  },
}

describe('workflow resource locator', () => {
  it('resolves an exact image resource variant without falling back to another variant', () => {
    expect(
      resolveWorkflowResourceBinding([image], {
        resourceId: 'template',
        variantId: '720p',
      }),
    ).toMatchObject({
      resource: { id: 'template' },
      resolution: [1280, 720],
      blob: { digest: `sha256:${'1'.repeat(64)}` },
    })
    expect(
      resolveWorkflowResourceBinding([image], {
        resourceId: 'template',
        variantId: 'missing',
      }),
    ).toBeUndefined()
  })

  it('maps source resource kinds to the matching workspace tool', () => {
    expect(workspaceResourceKind(image)).toBe('template')
    expect(
      workspaceResourceKind({
        id: 'clip',
        kind: 'input-clip',
        name: '轨迹',
        inputClip: {
          blob: { mediaType: 'application/octet-stream', digest: 'sha256:clip', size: 1 },
          baseResolution: [1, 1],
          durationUs: 1,
          eventCount: 1,
          mouseCounts360: 0,
          mouseMode: 'relative',
          recordingMode: 'precise',
          stopHotkeyVk: 0,
        },
      }),
    ).toBe('clip')
  })
})
