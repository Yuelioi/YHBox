import { describe, expect, it } from 'vitest'
import type { FieldProjection } from '../../../contracts/node/3.1/authoring-projection'
import {
  patchProjectedConfig,
  projectedConstraintTokens,
  projectedInitialConfig,
} from './nodeAuthoringUi'

function field(overrides: Partial<FieldProjection> = {}): FieldProjection {
  return {
    constraints: { enum: [] },
    control: 'text',
    deprecated: false,
    examples: [],
    hasDefault: false,
    id: 'mediaType',
    properties: [],
    readOnly: false,
    required: true,
    ...overrides,
  }
}

describe('node authoring UI semantics', () => {
  it('never materializes JSON Schema defaults into node config', () => {
    const source = { existing: true }
    const projected = projectedInitialConfig(source, [
      field({ hasDefault: true, default: 'application/octet-stream' }),
    ])

    expect(projected).toEqual({ existing: true })
    expect(projected).not.toBe(source)
    expect(projected).not.toHaveProperty('mediaType')
  })

  it('applies only explicit edits without mutating the source', () => {
    const source = { mediaType: 'image/png', untouched: 7 }
    const updated = patchProjectedConfig(source, 'mediaType', 'video/mp4')
    const removed = patchProjectedConfig(updated, 'mediaType', undefined)

    expect(source.mediaType).toBe('image/png')
    expect(updated).toEqual({ mediaType: 'video/mp4', untouched: 7 })
    expect(removed).toEqual({ untouched: 7 })
  })

  it('projects visible constraint facts instead of hiding them in a tooltip', () => {
    expect(
      projectedConstraintTokens(
        field({
          constraints: {
            enum: [],
            minLength: 3,
            maxLength: 255,
            pattern: '^[a-z]+/[a-z]+$',
          },
        }),
      ),
    ).toEqual(['minLength: 3', 'maxLength: 255', 'pattern: ^[a-z]+/[a-z]+$'])
  })
})
