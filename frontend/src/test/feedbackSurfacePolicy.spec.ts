import { readdirSync, readFileSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

const sourceRoot = join(process.cwd(), 'src')

function vueFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name)
    if (entry.isDirectory()) return vueFiles(path)
    return entry.isFile() && entry.name.endsWith('.vue') ? [path] : []
  })
}

describe('feedback surface policy', () => {
  it('keeps the Nuxt modal lifecycle behind the shared BaseModal boundary', () => {
    const directModalUsers = vueFiles(sourceRoot)
      .filter((path) => readFileSync(path, 'utf8').includes('<UModal'))
      .map((path) => relative(sourceRoot, path).replaceAll('\\', '/'))

    expect(directModalUsers).toEqual(['components/common/BaseModal.vue'])
  })

  it('uses one persistent surface for editor save failures', () => {
    const editor = readFileSync(join(sourceRoot, 'views/WorkflowEditorView.vue'), 'utf8')
    const runController = readFileSync(
      join(sourceRoot, 'app/editor/EditorRunController.ts'),
      'utf8',
    )

    expect(editor).toContain('v-if="session.saveError"')
    expect(editor).toContain('@click="session.dismissSaveError()"')
    expect(editor).toContain('@click="locateSaveError"')
    expect(editor).not.toContain('v-else-if="session.failure"')
    expect(runController).toContain('if (!dependencies.session.saveError)')
  })

  it('keeps persistent workflow resource feedback dismissible', () => {
    const dock = readFileSync(join(sourceRoot, 'app/editor/WorkflowResourceDock.vue'), 'utf8')

    expect(dock).toContain('data-testid="workflow-resource-feedback"')
    expect(dock).toContain('data-testid="workflow-resource-feedback-dismiss"')
    expect(dock).toContain(':aria-label="t(\'common.close\')"')
    expect(dock).toContain('@click="dismissFeedback"')
    expect(dock).toContain('useAutoDismissFeedback(feedback)')
  })
})
