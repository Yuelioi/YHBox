import { describe, expect, it } from 'vitest'
import { matchingInstalledApplications } from './windowTargetCapture'

describe('window target capture installation matching', () => {
  const installed = [
    {
      slot: 'editor',
      executable: String.raw`C:\Apps\Editor.exe`,
      executableDigest: 'sha256:AAAA',
    },
  ]

  it('matches the authorized installation path without depending on path casing or app version', () => {
    expect(
      matchingInstalledApplications(installed, {
        executable: 'c:/apps/editor.exe',
        digest: 'sha256:aaaa',
      }),
    ).toEqual(installed)
  })

  it('rejects a different path but keeps a normal update at the authorized path', () => {
    expect(
      matchingInstalledApplications(installed, {
        executable: String.raw`D:\Apps\Editor.exe`,
        digest: 'sha256:AAAA',
      }),
    ).toEqual([])
    expect(
      matchingInstalledApplications(installed, {
        executable: String.raw`C:\Apps\Editor.exe`,
        digest: 'sha256:BBBB',
      }),
    ).toEqual(installed)
  })
})
