import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/components/common/SectionHeader.vue'), 'utf8')

describe('SectionHeader typography', () => {
  it('uses sentence-case hierarchy instead of compact all-caps metadata styling', () => {
    expect(source).toContain('text-[13px] font-medium text-toned')
    expect(source).not.toContain('uppercase')
    expect(source).not.toContain('tracking-wider')
  })

  it('keeps counts readable and aligned', () => {
    expect(source).toContain('text-xs tabular-nums text-dimmed')
  })
})
