import { describe, expect, it } from 'vitest'
import { matchingInstalledApplications } from './windowTargetCapture'

describe('window target capture installation matching', () => {
  const installed = [
    {
      slot: 'editor',
      executable: String.raw`C:\Apps\Editor.exe`,
    },
  ]

  it('matches a configured application path without depending on path casing', () => {
    expect(
      matchingInstalledApplications(installed, {
        executable: 'c:/apps/editor.exe',
      }),
    ).toEqual(installed)
  })

  it('does not match a different configured path', () => {
    expect(
      matchingInstalledApplications(installed, {
        executable: String.raw`D:\Apps\Editor.exe`,
      }),
    ).toEqual([])
  })
})
