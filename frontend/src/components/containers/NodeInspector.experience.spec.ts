import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  join(process.cwd(), 'src/components/containers/NodeInspector.vue'),
  'utf8',
)

describe('NodeInspector experience modes', () => {
  it('keeps task capabilities visible while progressively disclosing technical controls', () => {
    expect(source).toContain('v-if="experienceMode === \'pro\'"')
    expect(source).toContain("t('editor.inspector.group_inputs')")
    expect(source).not.toContain("t('editor.experience.basic_inspector_hint')")
    expect(source).toContain("t('inspector.log_enabled_label')")
    expect(source).toContain('v-if="exprChainHint"')
    expect(source).not.toContain("experienceMode === 'pro' && exprChainHint")
    expect(source).not.toContain("experienceMode === 'pro' || danglingCaptures.length > 0")
  })

  it('keeps node identity visible and names icon-only actions', () => {
    expect(source).toContain('data-inspector-header')
    expect(source).toContain('sticky top-0')
    expect(source).toContain(':aria-label="t(\'inspector.help_tooltip\')"')
    expect(source).toContain(':aria-label="t(\'inspector.copy_menu_tooltip\')"')
    expect(source).toContain(':aria-label="t(\'inspector.delete_node_tooltip\')"')
  })
})
