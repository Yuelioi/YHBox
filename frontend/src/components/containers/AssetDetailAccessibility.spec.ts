import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const files = ['TemplateDetailPanel.vue', 'LibraryDetailPanel.vue', 'ClipDetailPanel.vue']

describe('asset detail editing', () => {
  it.each(files)('%s exposes click and keyboard reachable edit controls', (file) => {
    const source = readFileSync(join(process.cwd(), 'src/components/containers', file), 'utf8')
    expect(source).toContain('@click="enterEditName"')
    expect(source).toContain('@click="enterEditDesc"')
    expect(source).not.toContain('@dblclick="enterEditName"')
    expect(source).not.toContain('@dblclick="enterEditDesc"')
  })
})
