import { describe, expect, it } from 'vitest'
import { createApp, defineComponent, h, nextTick } from 'vue'
import { createI18n } from 'vue-i18n'
import en from '@/i18n/en'
import { builtinNodeProjections31 } from '@/contracts/node31'
import NodeAuthoringPanel from './NodeAuthoringPanel.vue'

function mountPanel() {
  const projection = structuredClone(
    builtinNodeProjections31.get('https://schemas.yotta.dev/nodes/conversion/stream-to-blob/v1')!,
  )
  projection.configFields[0].hasDefault = true
  projection.configFields[0].default = 'application/octet-stream'

  const updates: Record<string, unknown>[] = []
  const app = createApp(
    defineComponent(
      () => () =>
        h(NodeAuthoringPanel, {
          nodeId: 'node/1',
          projection,
          modelValue: {},
          'onUpdate:modelValue': (config: Record<string, unknown>) => updates.push(config),
        }),
    ),
  )
  app.use(createI18n({ legacy: false, locale: 'en', messages: { en } }))
  const el = document.createElement('div')
  document.body.appendChild(el)
  app.mount(el)
  return { app, el, updates }
}

describe('NodeAuthoringPanel', () => {
  it('associates visible hints with the field and writes only explicit edits', async () => {
    const wrapper = mountPanel()
    await nextTick()

    const input = wrapper.el.querySelector('input[type="text"]') as HTMLInputElement
    const describedBy = input.getAttribute('aria-describedby')
    expect(input.value).toBe('')
    expect(input.required).toBe(true)
    expect(describedBy).toBeTruthy()
    expect(wrapper.el.querySelector(`#${describedBy}`)?.textContent).toContain(
      'Default hint: application/octet-stream',
    )
    expect(wrapper.updates).toEqual([])

    input.value = 'image/png'
    input.dispatchEvent(new Event('input', { bubbles: true }))
    await nextTick()
    expect(wrapper.updates).toEqual([{ mediaType: 'image/png' }])

    wrapper.app.unmount()
    wrapper.el.remove()
  })
})
