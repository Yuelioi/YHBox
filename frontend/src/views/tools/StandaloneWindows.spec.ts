import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const toolViews = [
  'MouseHUDView.vue',
  'RecordingHUDView.vue',
  'CalibrationHUDView.vue',
  'FloatingLauncherView.vue',
  'ScreenPickerView.vue',
]

const readSource = (path: string) => readFileSync(join(process.cwd(), path), 'utf8')

describe('standalone window presentation contract', () => {
  it.each(toolViews)('%s uses the shared HUD shell and Nuxt UI buttons', (filename) => {
    const source = readSource(`src/views/tools/${filename}`)

    expect(source).toContain('<HudShell')
    expect(source).not.toMatch(/<button(?:\s|>)/)
  })

  it('keeps window dragging and close accessibility in the shared shell', () => {
    const source = readSource('src/components/tools/HudShell.vue')

    expect(source).toContain('--wails-draggable: drag')
    expect(source).toContain('--wails-draggable: no-drag')
    expect(source).toContain(':aria-label="resolvedCloseTitle"')
  })

  it('shares the semantic state panel between live HUDs', () => {
    for (const filename of ['RecordingHUDView.vue', 'CalibrationHUDView.vue']) {
      expect(readSource(`src/views/tools/${filename}`)).toContain('<HudStatePanel')
    }
  })
})
