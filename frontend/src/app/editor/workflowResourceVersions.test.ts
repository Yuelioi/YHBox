import { describe, expect, it } from 'vitest'
import { reactive } from 'vue'
import type { WorkflowResource } from '../../../../contracts/workflow/current/workflow-source'
import { applyCapturedImageVersion, WorkflowResourceVersionError } from './workflowResourceVersions'

const blob = (digest: string) => ({ mediaType: 'image/png', digest, size: 12 })
const resource = (): WorkflowResource => ({
  id: 'image-1',
  kind: 'image',
  name: 'Target',
  image: {
    variants: [
      { id: 'a', resolution: [100, 100], bbox: [0, 0, 1, 1], blob: blob('sha256:old') },
      { id: 'z', resolution: [200, 200], bbox: [0, 0, 1, 1], blob: blob('sha256:other') },
    ],
  },
})
const captured = {
  id: 'captured',
  resolution: [300, 300] as [number, number],
  bbox: [0, 0, 1, 1] as [number, number, number, number],
  blob: blob('sha256:new'),
}

describe('workflow image versions', () => {
  it('re-records the current version while preserving its referenced id', () => {
    const result = applyCapturedImageVersion(resource(), captured, 'replace')
    expect(result.image?.variants.map((variant) => variant.id)).toEqual(['a', 'z'])
    expect(result.image?.variants[0]?.blob.digest).toBe('sha256:new')
  })

  it('re-records the selected variant instead of implicitly replacing the first one', () => {
    const result = applyCapturedImageVersion(resource(), captured, 'replace', 'z')
    expect(result.image?.variants.map((variant) => variant.id)).toEqual(['a', 'z'])
    expect(result.image?.variants[0]?.blob.digest).toBe('sha256:old')
    expect(result.image?.variants[1]?.blob.digest).toBe('sha256:new')
  })

  it('appends and sorts a new version without changing existing versions', () => {
    const result = applyCapturedImageVersion(resource(), captured, 'append')
    expect(result.image?.variants.map((variant) => variant.id)).toEqual(['a', 'captured', 'z'])
    expect(result.image?.variants[0]?.blob.digest).toBe('sha256:old')
  })

  it('accepts a captured variant projected through a reactive event boundary', () => {
    const result = applyCapturedImageVersion(resource(), reactive(captured), 'replace')
    expect(result.image?.variants[0]?.blob.digest).toBe('sha256:new')
  })

  it('rejects a missing replacement target', () => {
    expect(() =>
      applyCapturedImageVersion(resource(), captured, 'replace', 'missing'),
    ).toThrowError(
      expect.objectContaining<Partial<WorkflowResourceVersionError>>({
        id: 'workflow.resource.recapture_target_stale',
      }),
    )
  })
})
