import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  join(process.cwd(), 'src/components/containers/dock/ContainerEditorDock.vue'),
  'utf8',
)

describe('ContainerEditorDock adaptive structure', () => {
  it('supports an overlay mode that no longer consumes canvas width', () => {
    expect(source).toContain("'dock-shell--overlay': overlay")
    expect(source).toContain('position: absolute')
    expect(source).toContain('v-if="!overlay"')
  })
})
