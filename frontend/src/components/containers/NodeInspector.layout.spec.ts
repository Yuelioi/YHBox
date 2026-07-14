import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const inspector = readFileSync(
  join(process.cwd(), 'src/components/containers/NodeInspector.vue'),
  'utf8',
)
const shell = readFileSync(
  join(process.cwd(), 'src/components/containers/ContainerEditorInspector.vue'),
  'utf8',
)

describe('NodeInspector layout', () => {
  it('keeps scroll-container padding from opening a gap above the sticky header', () => {
    const aside = shell.match(/<aside[^>]+>/)?.[0] ?? ''

    expect(aside).toContain('overflow-y-auto')
    expect(aside).not.toContain('p-4')
    expect(inspector).toContain('<div v-else class="p-4">')
    expect(inspector).toContain('data-inspector-header')
    expect(inspector).toContain('sticky top-0')
    expect(inspector).toContain('-mx-4 -mt-4')
  })

  it('uses the global button alignment baseline for fixed-size header actions', () => {
    expect(inspector).toContain('class="size-8 p-0"')
    expect(inspector).not.toContain('inspectorIconButtonUi')
    expect(inspector).toContain('items-center gap-3')
  })
})
