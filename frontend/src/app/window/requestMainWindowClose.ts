import { allowsMainWindowClose, type MainWindowCloseStage } from './mainWindowCloseGuard'

export async function requestMainWindowClose(
  close: () => void | Promise<void>,
  setStage: (stage: MainWindowCloseStage) => void = () => undefined,
): Promise<boolean> {
  setStage('checking')
  const closeWithFeedback = async () => {
    setStage('closing')
    await allowClosingFeedbackToPaint()
    await close()
  }
  const guardResult = await allowsMainWindowClose({ close: closeWithFeedback, setStage })
  if (guardResult === false) return false
  if (guardResult === 'handled') return true
  await closeWithFeedback()
  return true
}

function allowClosingFeedbackToPaint(): Promise<void> {
  if (typeof requestAnimationFrame !== 'function') return Promise.resolve()
  return new Promise((resolve) => requestAnimationFrame(() => resolve()))
}
