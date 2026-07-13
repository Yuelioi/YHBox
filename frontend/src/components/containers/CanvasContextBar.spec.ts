import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  join(process.cwd(), 'src/components/containers/CanvasContextBar.vue'),
  'utf8',
)

describe('CanvasContextBar responsive structure', () => {
  it('keeps actions on one line and switches to icon labels from canvas width', () => {
    expect(source).toContain('white-space: nowrap')
    expect(source).toContain('@container editor-canvas (max-width: 540px)')
    expect(source).toContain('.context-action-label')
    expect(source).toContain('context-selection--compact')
  })
})
