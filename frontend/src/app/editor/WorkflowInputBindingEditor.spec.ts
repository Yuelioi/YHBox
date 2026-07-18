import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  join(process.cwd(), 'src/app/editor/WorkflowInputBindingEditor.vue'),
  'utf8',
)

describe('WorkflowInputBindingEditor', () => {
  it('authors List<KeyCode> as a recorded chord instead of raw JSON', () => {
    expect(source).toContain('<KeyChordValueEditor')
    expect(source).toContain('isKeyChordType(port.type.expression)')
  })

  it('opens the shared paged asset picker instead of expanding the full library', () => {
    expect(source).toContain('<AssetPickerModal')
    expect(source).toContain("kind: 'bind-blob'")
    expect(source).not.toContain('templateVariantItems')
    expect(source).not.toContain('clipItems')
  })
})
