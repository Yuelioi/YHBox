interface MainWindowCloseRouter {
  push(target: { name: string }): Promise<unknown>
}

export async function requestMainWindowClose(
  routeName: unknown,
  router: MainWindowCloseRouter,
  close: () => void | Promise<void>,
): Promise<boolean> {
  if (routeName === 'workflow-edit') {
    const failure = await router.push({ name: 'workflows' })
    if (failure) return false
  }

  await close()
  return true
}
