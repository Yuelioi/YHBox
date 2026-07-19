import { describe, expect, it } from 'vitest'
import {
  applyBatchMetadata,
  createBatchMetadataDraft,
  hasBatchMetadataChange,
} from './batchMetadata'

describe('batch metadata operations', () => {
  it('preserves fields until the user explicitly changes them', () => {
    const draft = createBatchMetadataDraft()
    expect(hasBatchMetadataChange(draft)).toBe(false)
    expect(applyBatchMetadata({ category: 'Input', tags: ['Stable'] }, draft)).toEqual({
      category: 'Input',
      tags: ['Stable'],
    })
  })

  it('supports independent category and tag operations', () => {
    expect(
      applyBatchMetadata(
        { category: 'Input', tags: ['Stable', 'Daily'] },
        {
          categoryMode: 'set',
          category: 'Production',
          tagMode: 'add',
          tags: ['Reviewed', 'stable'],
        },
      ),
    ).toEqual({ category: 'Production', tags: ['Stable', 'Daily', 'Reviewed'] })

    expect(
      applyBatchMetadata(
        { category: 'Input', tags: ['Stable', 'Daily'] },
        { categoryMode: 'clear', category: '', tagMode: 'remove', tags: ['daily'] },
      ),
    ).toEqual({ category: '', tags: ['Stable'] })
  })

  it('requires values for modes that consume values', () => {
    expect(
      hasBatchMetadataChange({
        categoryMode: 'set',
        category: ' ',
        tagMode: 'keep',
        tags: [],
      }),
    ).toBe(false)
    expect(
      hasBatchMetadataChange({
        categoryMode: 'keep',
        category: '',
        tagMode: 'replace',
        tags: [],
      }),
    ).toBe(false)
    expect(
      hasBatchMetadataChange({
        categoryMode: 'clear',
        category: '',
        tagMode: 'clear',
        tags: [],
      }),
    ).toBe(true)
  })
})
