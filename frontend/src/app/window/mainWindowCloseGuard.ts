export interface MainWindowCloseRequest {
  close: () => void | Promise<void>
  setStage: (stage: MainWindowCloseStage) => void
}

export type MainWindowCloseStage = 'checking' | 'stopping' | 'restoring' | 'closing'

export type MainWindowCloseGuardResult = boolean | 'handled'
export type MainWindowCloseGuard = (
  request: MainWindowCloseRequest,
) => MainWindowCloseGuardResult | Promise<MainWindowCloseGuardResult>

let activeGuard: MainWindowCloseGuard | undefined

export function registerMainWindowCloseGuard(guard: MainWindowCloseGuard): () => void {
  activeGuard = guard
  return () => {
    if (activeGuard === guard) activeGuard = undefined
  }
}

export async function allowsMainWindowClose(
  request: MainWindowCloseRequest,
): Promise<MainWindowCloseGuardResult> {
  return (await activeGuard?.(request)) ?? true
}
