import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  join(process.cwd(), 'src/components/containers/ContainerEditorToolbar.vue'),
  'utf8',
)

function zone(name: string): string {
  const match = source.match(new RegExp(`<div[^>]+data-zone="${name}"[\\s\\S]*?<!-- /${name} -->`))
  return match?.[0] ?? ''
}

describe('ContainerEditorToolbar structure', () => {
  it('uses independent identity, workflow, and utility grid zones', () => {
    expect(zone('identity')).toContain('toolbar-breadcrumb')
    expect(zone('workflow')).toContain('toolbar-workflow')
    expect(zone('utility')).toContain('toolbar-utility')
    expect(source).toContain('grid-template-columns: minmax(180px, 1fr) auto minmax(80px, 1fr)')
  })

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

  it('progressively hides secondary labels without removing their actions', () => {
    expect(source).toContain('@container editor-toolbar (max-width: 1450px)')
    expect(source).toContain('.toolbar-secondary-label')
    expect(source).toContain('.toolbar-state-label')
    expect(source).toContain('.toolbar-utility-label')
  })
})
