import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const readSource = (path: string) => readFileSync(join(process.cwd(), path), 'utf8')

describe('Yotta button roles', () => {
  it('uses a token-derived contained primary action without the old CTA treatment', () => {
    const viteConfig = readSource('vite.config.ts')
    const styles = readSource('src/style.css')

    expect(viteConfig).toContain("class: { base: 'btn-primary-contained' }")
    expect(viteConfig).not.toContain('btn-primary-raised')
    expect(styles).toContain('.btn-primary-contained {')
    expect(styles).toContain('.btn-primary-contained:hover')
    expect(styles).toContain('.btn-primary-contained:active')
    expect(styles).toContain('.btn-primary-contained:focus-visible')
    expect(styles).toContain('.btn-primary-contained:disabled')
    expect(styles).toContain(
      '--ui-action-primary-bg: color-mix(in oklab, var(--ui-primary) 7%, var(--ui-bg-elevated))',
    )
    expect(styles).toContain('0 2px 5px var(--ui-action-primary-shadow)')
    expect(styles).not.toContain('.btn-primary-raised')
  })

  it('renders pagination as a selected state instead of a primary submit action', () => {
    for (const path of [
      'src/views/WorkflowsView.vue',
      'src/views/AssetsView.vue',
      'src/app/editor/WorkflowResourceDock.vue',
    ]) {
      expect(readSource(path)).toMatch(/<UPagination[\s\S]*?active-variant="subtle"[\s\S]*?\/>/)
    }
  })

  it('keeps editor-side capture and recording actions at quiet accent emphasis', () => {
    const dock = readSource('src/app/editor/WorkflowResourceDock.vue')
    const createActions = dock.match(
      /data-testid="workflow-resource-create"[\s\S]*?@click="emit\('[^']+'[^>]*\/>/g,
    )

    expect(createActions).toHaveLength(2)
    expect(createActions?.every((source) => source.includes('color="primary"'))).toBe(true)
    expect(createActions?.every((source) => source.includes('variant="soft"'))).toBe(true)
  })

  it('does not reuse a solid primary action surface for compact selected controls', () => {
    for (const path of [
      'src/app/editor/PointValueEditor.vue',
      'src/app/editor/RegionValueEditor.vue',
      'src/components/common/IconPicker.vue',
    ]) {
      const source = readSource(path)
      expect(source).not.toContain('bg-primary text-inverted')
      expect(source).not.toContain('background: var(--ui-primary)')
    }
  })
})
