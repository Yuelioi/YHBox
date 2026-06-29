import { createApp, defineComponent, nextTick } from 'vue'
import { createI18n } from 'vue-i18n'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import NodeContextMenu from './NodeContextMenu.vue'

const mocks = vi.hoisted(() => ({
  pinsFor: vi.fn(),
  getSpec: vi.fn(),
}))

vi.mock('@/components/containers/pinSpec', () => ({
  pinsFor: mocks.pinsFor,
}))

vi.mock('@/components/containers/nodeRegistry/registry', () => ({
  getSpec: mocks.getSpec,
}))

function mountMenu(nodeKind = 'ClickAt') {
  const i18n = createI18n({
    legacy: false,
    locale: 'en',
    messages: {
      en: {
        node: { ClickAt: { label: 'Click at' } },
        editor: {
          menu: {
            node: {
              copy: 'Copy',
              cut: 'Cut',
              paste: 'Paste',
              duplicate: 'Duplicate',
              delete: 'Delete',
              debug_from_here: 'Debug from here',
              enable: 'Enable',
              disable: 'Disable',
              save_as_snippet: 'Save as snippet',
            },
          },
        },
      },
    },
  })

  const emitted: { action: string[]; updateOpen: boolean[] } = { action: [], updateOpen: [] }
  const app = createApp(NodeContextMenu, {
    open: true,
    position: { x: 10, y: 20 },
    node: {
      id: 'node-1',
      kind: nodeKind,
      label: '',
      config: {},
      x: 0,
      y: 0,
    } as any,
    'onUpdate:open': (value: boolean) => emitted.updateOpen.push(value),
    onAction: (action: string) => emitted.action.push(action),
  })
  app.use(i18n)
  app.component('UIcon', defineComponent({ template: '<span />' }))

  const el = document.createElement('div')
  document.body.appendChild(el)
  app.mount(el)

  return {
    el,
    emitted,
    unmount: () => {
      app.unmount()
      el.remove()
    },
  }
}

describe('NodeContextMenu debug action', () => {
  beforeEach(() => {
    mocks.pinsFor.mockReset()
    mocks.getSpec.mockReset()
    mocks.getSpec.mockReturnValue({
      kind: 'ClickAt',
      labelZh: 'node.ClickAt.label',
      visual: { icon: 'i-tabler-pointer' },
    })
  })

  it('hides Debug from here when the node has no exec input', () => {
    mocks.pinsFor.mockReturnValue({ execIn: [], execOut: [], dataIn: [], dataOut: [] })

    const wrapper = mountMenu('Start')
    try {
      expect(wrapper.el.textContent).not.toContain('Debug from here')
    } finally {
      wrapper.unmount()
    }
  })

  it('emits debug-from-here for executable nodes', async () => {
    mocks.pinsFor.mockReturnValue({ execIn: ['In'], execOut: ['Done'], dataIn: [], dataOut: [] })

    const wrapper = mountMenu()
    try {
      const buttons = [...wrapper.el.querySelectorAll('button')]
      const debugButton = buttons.find((button) => button.textContent?.includes('Debug from here'))

      expect(debugButton).toBeTruthy()
      debugButton!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
      await nextTick()

      expect(wrapper.emitted.action[0]).toBe('debug-from-here')
      expect(wrapper.emitted.updateOpen[0]).toBe(false)
    } finally {
      wrapper.unmount()
    }
  })
})
