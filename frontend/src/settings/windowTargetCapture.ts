export interface ConfiguredApplication {
  slot: string
  executable: string
}

export interface CapturedApplication {
  executable: string
}

export function matchingInstalledApplications<T extends ConfiguredApplication>(
  applications: readonly T[],
  captured: CapturedApplication,
): T[] {
  const executable = normalizeWindowsPath(captured.executable)
  return applications.filter(
    (application) => normalizeWindowsPath(application.executable) === executable,
  )
}

function normalizeWindowsPath(value: string): string {
  return value.trim().replaceAll('/', '\\').toLocaleLowerCase()
}
