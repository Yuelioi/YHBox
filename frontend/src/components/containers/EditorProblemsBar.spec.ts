import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  join(process.cwd(), 'src/components/containers/EditorProblemsBar.vue'),
  'utf8',
)

describe('EditorProblemsBar hierarchy', () => {
  it('presents the readable message before technical metadata', () => {
    expect(source.indexOf('{{ errorText(e) }}')).toBeLessThan(source.indexOf('{{ e.code }}'))
    expect(source).toContain('text-[13px] leading-5 text-default')
  })

  it('uses calm text counts and accessible disclosure state', () => {
    expect(source).not.toContain('<UBadge')
    expect(source).toContain(':aria-expanded="expanded"')
    expect(source).toContain('aria-controls="editor-problems-panel"')
  })

  it('offers the shared safe fix path for stale unknown literal values', () => {
    expect(source).toContain('import { literalValidationFix }')
    expect(source).toContain('return literalValidationFix(e) !== null')
  })
})
