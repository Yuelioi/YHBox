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

  it('matches the canonical executable and digest without depending on path casing', () => {
    expect(
      matchingInstalledApplications(installed, {
        executable: 'c:/apps/editor.exe',
        digest: 'sha256:aaaa',
      }),
    ).toEqual(installed)
  })

  it('rejects the same basename or path when exact identity differs', () => {
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
    ).toEqual([])
  })
})
