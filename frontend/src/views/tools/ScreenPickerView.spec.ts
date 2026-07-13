import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/views/tools/ScreenPickerView.vue'), 'utf8')

describe('ScreenPickerView template capture metadata', () => {
  it('collects a category and persists it with the new template', () => {
    expect(source).toContain('v-model="tplCategory"')
    expect(source).toMatch(
      /saveTemplateCapture\(\s*png,\s*tplName\.value\.trim\(\),\s*tplCategory\.value\.trim\(\),/,
    )
  })
})
