import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

function source(path: string) {
  return readFileSync(join(process.cwd(), 'src/components', path), 'utf8')
}

describe('editor interaction accessibility', () => {
  it.each([
    'containers/dock/TemplateAssetPanel.vue',
    'containers/dock/LibraryAssetPanel.vue',
    'containers/dock/ClipAssetPanel.vue',
  ])('%s exposes asset selection and details to the keyboard', (file) => {
    const panel = source(file)
    expect(panel).toContain('role="option"')
    expect(panel).toContain(':tabindex="isTabStop(')
    expect(panel).toContain('aria-selected')
    expect(panel).toContain('data-asset-browser-list')
    expect(panel).toContain('data-asset-option')
    expect(panel).toContain('@focus="setActive(')
    expect(panel).toContain('if (move(')
    expect(panel).toMatch(/@keydown="on(?:Cell|Row)Keydown/)
  })

  it.each([
    'containers/dock/TemplateAssetPanel.vue',
    'containers/dock/LibraryAssetPanel.vue',
    'containers/dock/ClipAssetPanel.vue',
  ])('%s keeps details reachable when the workspace collapses', (file) => {
    const panel = source(file)
    expect(panel.match(/<AssetWorkspaceInspector/g)).toHaveLength(1)
  })

  it('moves and traps focus in the compact workspace inspector', () => {
    const inspector = source('containers/dock/AssetWorkspaceInspector.vue')
    expect(inspector).toContain('@container (width < 1040px)')
    expect(inspector).toContain('@keydown.esc.stop')
    expect(inspector).toContain('@keydown.tab="trapFocus"')
    expect(inspector).toContain('sibling.inert = value')
  })

  it('keeps template variant deletion as a named keyboard action', () => {
    const detail = source('containers/TemplateDetailPanel.vue')
    expect(detail).toContain("t('template.picker.del_variant_title'")
    expect(detail).toContain('@click="removeVariant(v.resolution)"')
  })

  it('keeps template variants visible before secondary metadata in every detail entry', () => {
    const detail = source('containers/TemplateDetailPanel.vue')
    expect(detail.match(/template\.picker\.variants_label/g)).toHaveLength(1)
    expect(detail).toContain('v-if="detailLoading"')
    expect(detail).not.toContain('v-if="detailRecord?.variants?.length"')
    expect(detail.indexOf('template.picker.variants_label')).toBeLessThan(
      detail.indexOf('library.detail.description'),
    )
  })

  it('uses a compact preview with an explicit full-image action', () => {
    const detail = source('containers/TemplateDetailPanel.vue')
    expect(detail).toContain('class="relative flex h-40')
    expect(detail).toContain("t('template.detail.view_large')")
    expect(detail).toContain('<BaseModal')
  })

  it('gives low-resolution template thumbnails a quiet inset stage', () => {
    const panel = source('containers/dock/TemplateAssetPanel.vue')
    expect(panel).toContain('bg-sunken p-3')
  })

  it('provides a visible, named edit control for comment boxes', () => {
    const comment = source('containers/CommentBoxNode.vue')
    expect(comment).toContain('class="nodrag cb-edit"')
    expect(comment).toContain(':aria-label="t(\'common.edit\')"')
    expect(comment).toContain('@click.stop="enterEdit"')
  })

  it('lets keyboard users apply and manage snippets', () => {
    const snippets = source('snippets/SnippetsPanel.vue')
    expect(snippets).toContain('role="button"')
    expect(snippets).toContain('@keydown.enter.prevent')
    expect(snippets).toContain('group-focus-within:opacity-100')
  })
})
