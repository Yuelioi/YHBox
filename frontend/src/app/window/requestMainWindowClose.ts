import { allowsMainWindowClose } from './mainWindowCloseGuard'

export async function requestMainWindowClose(close: () => void | Promise<void>): Promise<boolean> {
  if (!(await allowsMainWindowClose())) return false
  await close()
  return true
}
