import { createI18n } from 'vue-i18n'
import { describe, expect, it, vi } from 'vitest'
import { createApp, defineComponent } from 'vue'
import NodeLibraryPanel from './NodeLibraryPanel.vue'

vi.mock('@/components/containers/nodeRegistry/registry', () => ({
  allSpecs: () => [],
}))

vi.mock('@/composables/editor/useEditorDragDrop', () => ({
  startEditorDrag: vi.fn(),
}))

describe('NodeLibraryPanel', () => {
  it('keeps search outside the scrollable node list', () => {
    const app = createApp(NodeLibraryPanel)
    app.use(
      createI18n({
        legacy: false,
        locale: 'zh',
        messages: {
          zh: {
            nodeExplorer: {
              title: '节点库',
              search_placeholder: '搜索节点',
              no_match: '没有匹配',
            },
            nodeTarget: {
              windows: 'Windows',
              android: 'Android',
            },
          },
        },
      }),
    )
    app.component('UIcon', defineComponent({ template: '<span />' }))
    app.component('UInput', defineComponent({ template: '<input />' }))

    const el = document.createElement('div')
    document.body.appendChild(el)
    try {
      app.mount(el)
      const search = el.querySelector('[data-testid="node-library-search"]')
      const scroll = el.querySelector('[data-testid="node-library-scroll"]')
      expect(search).toBeTruthy()
      expect(scroll).toBeTruthy()
      expect(scroll!.contains(search)).toBe(false)
    } finally {
      app.unmount()
      el.remove()
    }
  })
})
