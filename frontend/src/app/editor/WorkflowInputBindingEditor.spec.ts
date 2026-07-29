import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  join(process.cwd(), 'src/app/editor/WorkflowInputBindingEditor.vue'),
  'utf8',
)

describe('WorkflowInputBindingEditor', () => {
  it('authors List<KeyCode> as a recorded chord instead of raw JSON', () => {
    expect(source).toContain('<WorkflowValueEditor')
    expect(source).toContain('resolvePortAdapter(props.port)')
  })

  it('opens the shared paged asset picker instead of expanding the full library', () => {
    expect(source).toContain('<AssetPickerModal')
    expect(source).toContain("kind: 'bind-blob'")
    expect(source).toContain(':resources="resources"')
    expect(source).toContain('@select-workflow="setWorkflowAsset"')
    expect(source).toContain("kind: 'bind-resource'")
    expect(source).toContain('@capture="emit(\'capture-template\')"')
    expect(source).not.toContain('templateVariantItems')
    expect(source).not.toContain('clipItems')
  })
})
