import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  join(process.cwd(), 'src/components/workflows/WorkflowInstallationUpdateModal.vue'),
  'utf8',
)

describe('WorkflowInstallationUpdateModal', () => {
  it('stages only cached candidates and consumes the opaque preview token', () => {
    expect(source).toContain('workflowTransport.listInstallationUpdateCandidates')
    expect(source).toContain('workflowTransport.previewInstallationUpdate')
    expect(source).toContain('workflowTransport.previewInstallationRollback')
    expect(source).toContain('workflowTransport.applyInstallationUpdate(preview.value.token)')
    expect(source).toContain('result.reconciliationRequired')
    expect(source).toContain("t('workflow.installation.update_reconciliation_title')")
    expect(source).not.toContain('sourceArtifact')
  })

  it('shows exact dependency, local-definition, permission, conflict, and readiness changes', () => {
    for (const field of [
      'addedDependencies',
      'removedDependencies',
      'addedTargets',
      'changedTargets',
      'removedTargets',
      'addedCredentials',
      'changedCredentials',
      'removedCredentials',
      'addedCapabilities',
      'removedCapabilities',
      'preview.conflicts',
      'preview.readiness.blockers',
    ]) {
      expect(source).toContain(field)
    }
    expect(source).toContain('preview.conflicts.length > 0')
    expect(source).not.toContain("t('workflow.installation.update_consent_warning')")
  })
})
