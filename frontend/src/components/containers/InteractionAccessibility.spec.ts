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
    expect(panel).toContain('tabindex="0"')
    expect(panel).toContain('aria-selected')
    expect(panel).toMatch(/@keydown="on(?:Cell|Row)Keydown/)
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
