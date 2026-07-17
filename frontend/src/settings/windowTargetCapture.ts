export interface InstalledApplicationIdentity {
  slot: string
  executable: string
  executableDigest: string
}

export interface ExecutableIdentity {
  executable: string
  digest: string
}

export function matchingInstalledApplications<T extends InstalledApplicationIdentity>(
  applications: readonly T[],
  captured: ExecutableIdentity,
): T[] {
  const executable = normalizeWindowsPath(captured.executable)
  const digest = captured.digest.toLocaleLowerCase()
  return applications.filter(
    (application) =>
      normalizeWindowsPath(application.executable) === executable &&
      application.executableDigest.toLocaleLowerCase() === digest,
  )
}

function normalizeWindowsPath(value: string): string {
  return value.trim().replaceAll('/', '\\').toLocaleLowerCase()
}
