import { allowsMainWindowClose } from './mainWindowCloseGuard'

export async function requestMainWindowClose(close: () => void | Promise<void>): Promise<boolean> {
  const guardResult = await allowsMainWindowClose({ close })
  if (guardResult === false) return false
  if (guardResult === 'handled') return true
  await close()
  return true
}
