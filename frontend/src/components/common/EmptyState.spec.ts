import { createApp } from 'vue'
import { describe, expect, it } from 'vitest'
import EmptyState from './EmptyState.vue'

function render(props: { inset?: boolean } = {}): { element: HTMLElement; dispose: () => void } {
  const host = document.createElement('div')
  document.body.appendChild(host)
  const app = createApp(EmptyState, {
    icon: 'i-tabler-box-off',
    title: 'No items',
    ...props,
  })
  app.config.warnHandler = () => undefined
  app.mount(host)

  return {
    element: host.firstElementChild as HTMLElement,
    dispose: () => {
      app.unmount()
      host.remove()
    },
  }
}

describe('EmptyState surface spacing', () => {
  it('provides a standard gutter when nested in an edge-to-edge list surface', () => {
    const view = render({ inset: true })
    try {
      expect(view.element.classList).toContain('m-3')
    } finally {
      view.dispose()
    }
  })

  it('stays aligned with an already padded page by default', () => {
    const view = render()
    try {
      expect(view.element.classList).not.toContain('m-3')
    } finally {
      view.dispose()
    }
  })
})
