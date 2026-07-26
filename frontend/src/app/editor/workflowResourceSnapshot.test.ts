import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getAsset, getMacro, analyzeMacro, getClipSummary } = vi.hoisted(() => ({
  getAsset: vi.fn(),
  getMacro: vi.fn(),
  analyzeMacro: vi.fn(),
  getClipSummary: vi.fn(),
}))

vi.mock('@/lib/backend', () => ({
  backend: {
    assets: { get: getAsset },
    macros: { get: getMacro, analyze: analyzeMacro },
    clips: { summary: getClipSummary },
  },
}))

import { snapshotGlobalAsset } from './workflowResourceSnapshot'

const blob = {
  mediaType: 'application/octet-stream',
  digest: `sha256:${'a'.repeat(64)}`,
  size: 42,
}

describe('Global Asset Workflow Resource snapshots', () => {
  beforeEach(() => vi.clearAllMocks())

  it('copies every image variant and presentation field', async () => {
    getAsset.mockResolvedValue({
      variants: [
        {
          resolution: [1920, 1080],
          bbox: [10, 20, 40, 60],
          blob: { ...blob, mediaType: 'image/png' },
        },
      ],
    })
    const resource = await snapshotGlobalAsset({
      guid: 'template-1',
      kind: 'template',
      name: 'Submit',
      description: ' Primary ',
      category: ' UI ',
      tags: ['button'],
      variantCount: 1,
      variants: [],
    })
    expect(resource).toMatchObject({
      id: 'asset-template-1',
      kind: 'image',
      name: 'Submit',
      description: 'Primary',
      category: 'UI',
      image: {
        variants: [
          {
            id: 'variant-1920x1080-1',
            resolution: [1920, 1080],
            bbox: [10, 20, 40, 60],
          },
        ],
      },
    })
  })

  it('derives Macro and InputClip summaries from immutable carriers', async () => {
    getMacro.mockResolvedValue({
      blob: { ...blob, mediaType: 'application/vnd.yotta.macro+json' },
      document: { baseResolution: [1280, 720], actions: [{ id: 'wait' }] },
    })
    analyzeMacro.mockResolvedValue({ durationUs: 25_000 })
    const macro = await snapshotGlobalAsset({
      guid: 'macro-1',
      kind: 'macro',
      name: 'Wait',
      variantCount: 0,
      variants: [],
      blob,
    })
    expect(macro.macro).toMatchObject({
      baseResolution: [1280, 720],
      actionCount: 1,
      durationUs: 25_000,
    })

    getClipSummary.mockResolvedValue({
      blob: { ...blob, mediaType: 'application/vnd.yotta.input-clip' },
      durationUs: 30_000,
      eventCount: 3,
      meta: {
        recordingMode: 'precise',
        mouseMode: 'relative',
        baseResolution: [1920, 1080],
        mouseCounts360: 900,
        stopHotkeyVK: 123,
      },
    })
    const clip = await snapshotGlobalAsset({
      guid: 'clip-1',
      kind: 'clip',
      name: 'Turn',
      variantCount: 0,
      variants: [],
      blob,
    })
    expect(clip.inputClip).toMatchObject({
      durationUs: 30_000,
      eventCount: 3,
      recordingMode: 'precise',
      mouseMode: 'relative',
      mouseCounts360: 900,
      stopHotkeyVk: 123,
    })
  })
})
