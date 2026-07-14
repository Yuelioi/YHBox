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
  it('keeps maintenance as a utility instead of a fourth asset type', () => {
    expect(dock).not.toContain("{ value: 'maintenance'")
    expect(dock).toContain('<AssetMaintenancePanel')
    expect(dock).toContain("emit('update:tab', 'maintenance')")
    expect(dock).toContain("t('assetBrowser.openWorkspace')")
  })

  it('uses product language for the three reusable asset types', () => {
    expect(dock).toContain("t('assetBrowser.visualTemplates')")
    expect(dock).toContain("t('assetBrowser.automationBlueprints')")
    expect(dock).toContain("t('assetBrowser.actionClips')")
  })

  it('offers cleanup workflows for recordings, blueprints, and templates', () => {
    expect(maintenance).toContain("t('assetMaintenance.recordings.title')")
    expect(maintenance).toContain("t('assetMaintenance.subgraphs.title')")
    expect(maintenance).toContain("t('assetMaintenance.templates.title')")
    expect(maintenance).toContain("openCleanup('recordings')")
    expect(maintenance).toContain("openCleanup('subgraphs')")
    expect(maintenance).toContain("openCleanup('templates')")
    expect(maintenance).not.toContain("emit('navigate'")
  })
})
