import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const dock = readFileSync(
  join(process.cwd(), 'src/components/containers/dock/AssetDockPanel.vue'),
  'utf8',
)
const maintenance = readFileSync(
  join(process.cwd(), 'src/components/containers/dock/AssetMaintenancePanel.vue'),
  'utf8',
)

describe('AssetDockPanel information architecture', () => {
  it('places maintenance in the asset navigation instead of beside it', () => {
    expect(dock).toContain("value: 'maintenance'")
    expect(dock).toContain('<AssetMaintenancePanel')
    expect(dock).not.toContain("t('recordingCleanup.action')")
  })

  it('offers recording cleanup and routes to existing blueprint and template managers', () => {
    expect(maintenance).toContain("t('assetMaintenance.recordings.title')")
    expect(maintenance).toContain("t('assetMaintenance.subgraphs.title')")
    expect(maintenance).toContain("t('assetMaintenance.templates.title')")
    expect(maintenance).toContain("emit('navigate', 'library')")
    expect(maintenance).toContain("emit('navigate', 'templates')")
  })
})
