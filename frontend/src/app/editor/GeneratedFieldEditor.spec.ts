import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/app/editor/GeneratedFieldEditor.vue'), 'utf8')

describe('GeneratedFieldEditor hierarchy', () => {
  it('keeps helper text below the field label in narrow inspectors', () => {
    expect(source).toContain(':description="description"')
    expect(source).toContain(':required="field.required"')
    expect(source).toContain("label: 'min-w-0 text-xs font-medium text-toned'")
    expect(source).toContain("description: 'mt-1 text-[11px] leading-4 text-muted'")
    expect(source).not.toContain(':hint="hint"')
  })
})
