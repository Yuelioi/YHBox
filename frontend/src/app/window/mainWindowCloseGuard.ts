export type MainWindowCloseGuard = () => boolean | Promise<boolean>

let activeGuard: MainWindowCloseGuard | undefined

export function registerMainWindowCloseGuard(guard: MainWindowCloseGuard): () => void {
  activeGuard = guard
  return () => {
    if (activeGuard === guard) activeGuard = undefined
  }
}

export async function allowsMainWindowClose(): Promise<boolean> {
  return (await activeGuard?.()) !== false
}
