import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const appSource = readFileSync(join(process.cwd(), 'src/App.vue'), 'utf8')
const runtimeWorkbenchSource = readFileSync(
  join(process.cwd(), 'src/app/editor/WorkflowRuntimeWorkbench.vue'),
  'utf8',
)

describe('application shell', () => {
  it('keeps logs in the workflow runtime workbench instead of management pages', () => {
    expect(appSource).not.toContain('<LogPanel')
    expect(appSource).not.toContain("import LogPanel from './components/LogPanel.vue'")
    expect(runtimeWorkbenchSource).toContain('<LogPanel v-else-if="tab === \'logs\'" embedded />')
  })
})
