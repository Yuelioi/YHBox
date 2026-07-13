import { createI18n } from 'vue-i18n'
import { describe, expect, it } from 'vitest'
import { createApp, defineComponent, nextTick } from 'vue'
import CanvasEmptyState from './CanvasEmptyState.vue'

describe('CanvasEmptyState', () => {
  it('offers direct node and recording actions', async () => {
    const events: string[] = []
    const Root = defineComponent({
      components: { CanvasEmptyState },
      template:
        '<CanvasEmptyState @open-nodes="events.push(\'open-nodes\')" @record="events.push(\'record\')" />',
      setup: () => ({ events }),
    })
    const app = createApp(Root)
    app.use(
      createI18n({
        legacy: false,
        locale: 'en',
        messages: {
          en: {
            editor: {
              inspector: {
                empty: {
                  canvas_title: 'Start building your automation',
                  canvas_desc: 'Add a node, or record an action to create the first step.',
                  add_first_node: 'Add first node',
                  record_actions: 'Record actions',
                  shortcuts_label: 'You can also use shortcuts',
                  tab_explorer: 'Open Node Explorer',
                  right_click_canvas: 'Right-click canvas',
                  command_palette: 'Command palette',
                },
              },
            },
          },
        },
      }),
    )
    app.component('UIcon', defineComponent({ template: '<span />' }))
    app.component(
      'UButton',
      defineComponent({
        emits: ['click'],
        template: '<button type="button" @click="$emit(\'click\')"><slot /></button>',
      }),
    )

    const el = document.createElement('div')
    document.body.appendChild(el)
    try {
      app.mount(el)
      const buttons = el.querySelectorAll('button')
      expect(buttons).toHaveLength(2)
      buttons[0]?.click()
      buttons[1]?.click()
      await nextTick()
      expect(events).toEqual(['open-nodes', 'record'])
    } finally {
      app.unmount()
      el.remove()
    }
  })
})
