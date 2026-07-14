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
    expect(detail).toContain('v-if="recordLoading"')
    expect(detail).not.toContain('v-if="detailRecord?.variants?.length"')
    expect(detail.indexOf('template.picker.variants_label')).toBeLessThan(
      detail.indexOf('library.detail.description'),
    )
  })

  it('keeps the add-variant action compact in dock and workspace details', () => {
    const detail = source('containers/TemplateDetailPanel.vue')
    const marker = detail.indexOf('data-testid="template-variant-capture"')
    const action = detail.slice(
      detail.lastIndexOf('<UButton', marker),
      detail.indexOf('</UButton>', marker),
    )
    expect(action).toContain('class="max-w-full"')
    expect(action).not.toMatch(/\sblock(?:\s|>)/)
  })

  it('does not report the target window as unavailable while resolution detection is pending', () => {
    const detail = source('containers/TemplateDetailPanel.vue')
    expect(detail).toContain('v-if="resolutionLoading"')
    expect(detail.indexOf('v-if="resolutionLoading"')).toBeLessThan(
      detail.indexOf('v-else-if="curRes"'),
    )
    expect(detail).toContain('const resolutionLoading = ref(false)')
    expect(detail).not.toContain('Promise.allSettled([backend.assets.get(guid), refreshCurRes()])')
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
    expect(panel).toContain(':max-upscale="1"')
  })

  it('passes the same explicit container context to every template detail entry', () => {
    const panel = source('containers/dock/TemplateAssetPanel.vue')
    const detail = source('containers/TemplateDetailPanel.vue')
    expect(panel.match(/:container-id="containerId"/g)).toHaveLength(2)
    expect(detail).toContain('containerId: string')
    expect(detail).toContain('const containerId = props.containerId')
    expect(detail).toContain('currentResolution(containerId)')
    expect(detail).toContain("'template_recapture', id, props.containerId")
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
