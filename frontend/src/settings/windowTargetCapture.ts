export interface InstalledApplicationIdentity {
  slot: string
  executable: string
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
  return applications.filter(
    (application) => normalizeWindowsPath(application.executable) === executable,
  )
}

function normalizeWindowsPath(value: string): string {
  return value.trim().replaceAll('/', '\\').toLocaleLowerCase()
}
