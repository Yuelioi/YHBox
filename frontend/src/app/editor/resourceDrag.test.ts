import { describe, expect, it } from 'vitest'
import { parseWorkspaceResource, serializeWorkspaceResource } from './resourceDrag'

describe('workspace resource drag payload', () => {
  it('round-trips a bounded asset selection', () => {
    const selection = {
      guid: 'template-1',
      kind: 'template' as const,
      name: 'Submit button',
      resolution: [1280, 720] as [number, number],
      blob: {
        digest: 'sha256:abc',
        size: 42,
        mediaType: 'image/png',
      },
    }

    expect(parseWorkspaceResource(serializeWorkspaceResource(selection))).toEqual(selection)
  })

  it('rejects unrelated and malformed drop data', () => {
    expect(parseWorkspaceResource('{"kind":"window"}')).toBeNull()
    expect(parseWorkspaceResource('not json')).toBeNull()
  })
})
