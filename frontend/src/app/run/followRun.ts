const terminalStatuses = new Set(['succeeded', 'failed', 'cancelled', 'interrupted'])

export async function pollTerminalRunStatus(
  readStatus: () => Promise<string>,
  shouldStop: () => boolean,
  options: {
    attempts?: number
    intervalMs?: number
    wait?: (milliseconds: number) => Promise<void>
  } = {},
): Promise<string | undefined> {
  const attempts = options.attempts ?? 12
  const intervalMs = options.intervalMs ?? 100
  const wait =
    options.wait ??
    ((milliseconds: number) => new Promise<void>((resolve) => setTimeout(resolve, milliseconds)))

  for (let attempt = 0; attempt < attempts && !shouldStop(); attempt += 1) {
    const status = await readStatus()
    if (terminalStatuses.has(status)) return status
    if (attempt + 1 < attempts) await wait(intervalMs)
  }
  return undefined
}
