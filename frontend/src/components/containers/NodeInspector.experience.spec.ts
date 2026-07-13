import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  join(process.cwd(), 'src/components/containers/NodeInspector.vue'),
  'utf8',
)

describe('NodeInspector experience modes', () => {
  it('keeps node inputs visible while progressively disclosing technical controls', () => {
    expect(source).toContain('v-if="experienceMode === \'pro\'"')
    expect(source).toContain("t('editor.inspector.group_inputs')")
    expect(source).toContain("t('editor.experience.basic_inspector_hint')")
  })
})
