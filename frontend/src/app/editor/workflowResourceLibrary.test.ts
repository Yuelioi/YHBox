import { describe, expect, it } from 'vitest'
import type {
  WorkflowResource,
  YottaWorkflowSource,
} from '../../../../contracts/workflow/current/workflow-source'
import {
  projectWorkflowResourcePage,
  workflowResourceReferenceCount,
} from './workflowResourceLibrary'

describe('workflow resource library projection', () => {
  it('filters, sorts, and paginates one thousand workflow resources deterministically', () => {
    const resources = Array.from({ length: 1_000 }, (_, index): WorkflowResource => {
      const number = String(index).padStart(4, '0')
      return {
        id: `template-${number}`,
        kind: 'image',
        name: `Template ${number}`,
        category: index % 2 === 0 ? 'Fishing' : 'UI',
        tags: index % 10 === 0 ? ['favorite', 'shared'] : ['shared'],
        image: {
          variants: [
            {
              id: 'default',
              resolution: [1, 1],
              bbox: [0, 0, 1, 1],
              blob: {
                mediaType: 'image/png',
                digest: `sha256:${index.toString(16).padStart(64, '0')}`,
                size: 1,
              },
            },
          ],
        },
      }
    })

    const page = projectWorkflowResourcePage(resources, {
      search: 'template',
      category: 'Fishing',
      allCategoriesValue: '__all__',
      tags: ['favorite'],
      sort: 'name_desc',
      page: 2,
      pageSize: 20,
    })

    expect(page.total).toBe(100)
    expect(page.items).toHaveLength(20)
    expect(page.items[0]?.id).toBe('template-0790')
    expect(page.items.at(-1)?.id).toBe('template-0600')
    expect(page.categories).toEqual([
      { value: 'Fishing', count: 500 },
      { value: 'UI', count: 500 },
    ])
    expect(page.tags).toEqual([
      { value: 'favorite', count: 100 },
      { value: 'shared', count: 1_000 },
    ])
  })

  it('counts references in nodes and graph calls', () => {
    const source = {
      graphs: [
        {
          nodes: [
            {
              bindings: {
                template: {
                  kind: 'resource',
                  resource: { resourceId: 'template', variantId: 'default' },
                },
              },
            },
          ],
          calls: [
            {
              bindings: {
                input: {
                  kind: 'resource',
                  resource: { resourceId: 'template', variantId: 'default' },
                },
              },
            },
          ],
        },
      ],
    } as unknown as Pick<YottaWorkflowSource, 'graphs'>

    expect(workflowResourceReferenceCount(source, 'template')).toBe(2)
    expect(workflowResourceReferenceCount(source, 'other')).toBe(0)
  })
})
