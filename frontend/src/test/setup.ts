import { vi } from 'vitest'

const emptyResponse = () =>
  new Response('', {
    status: 204,
    statusText: 'No Content',
  })

const testFetch: typeof fetch = vi.fn(async () => emptyResponse())

vi.stubGlobal('fetch', testFetch)

if (typeof window !== 'undefined') {
  Object.defineProperty(window, 'fetch', {
    configurable: true,
    writable: true,
    value: testFetch,
  })
}
