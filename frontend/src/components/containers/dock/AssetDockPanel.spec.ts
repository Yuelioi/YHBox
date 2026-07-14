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

  it('provides one workspace query across all three asset types', () => {
    expect(dock).toContain('v-model="workspaceQuery"')
    expect(dock).toContain("t('assetBrowser.searchAll')")
    expect(dock.match(/:workspace-query="workspaceQuery"/g)).toHaveLength(3)
    expect(dock).toContain('matchingCounts')
  })

  it('caps type navigation width in the expanded workspace', () => {
    expect(dock).toContain('class="asset-type-tabs min-w-0"')
    expect(dock).toContain("[data-workspace='true'] .asset-type-tabs")
    expect(dock).toContain('width: min(100%, 720px)')
  })

  it('forwards the editor container context to template workflows', () => {
    expect(dock).toContain('containerId: string')
    expect(dock).toContain(':container-id="containerId"')
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
