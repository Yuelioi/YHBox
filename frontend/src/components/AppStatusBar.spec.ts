import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/components/AppStatusBar.vue'), 'utf8')

describe('AppStatusBar user language', () => {
  it('tracks every active run while exposing only a localized count and one stop action', () => {
    expect(source).toContain("t('workflow.status.active_count'")
    expect(source).toContain('new Set(activeRunIds.value)')
    expect(source).toContain('workflowTransport.cancelAllRuns()')
    expect(source).not.toContain('workflow.status.program')
    expect(source).not.toContain('{{ event.runId }}')
  })
})
