import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/components/containers/ContainerEditorToolbar.vue'), 'utf8')

function zone(name: string): string {
  const match = source.match(new RegExp(`<div[^>]+data-zone="${name}"[\\s\\S]*?<!-- /${name} -->`))
  return match?.[0] ?? ''
}

describe('ContainerEditorToolbar structure', () => {
  it('keeps save, validate, debug, and run in the central workflow zone', () => {
    const workflow = zone('workflow')
    const utility = zone('utility')

    expect(workflow).toContain("t('editor.toolbar.save')")
    expect(workflow).toContain("t('editor.toolbar.validate')")
    expect(workflow).toContain("t('editor.toolbar.debug')")
    expect(workflow).toContain("t('editor.toolbar.run_hero')")
    expect(workflow).not.toContain("t('editor.toolbar.record')")
    expect(utility).toContain("t('editor.toolbar.record')")
  })

  it('keeps recording and layout tools out of the workflow zone', () => {
    const workflow = zone('workflow')
    const utility = zone('utility')

    expect(workflow).not.toContain('recordMenuItems')
    expect(workflow).not.toContain('layoutMenuItems')
    expect(utility).toContain('recordMenuItems')
    expect(utility).toContain('layoutMenuItems')
  })
})
