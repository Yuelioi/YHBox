import { describe, expect, it, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick } from 'vue'
import { createI18n } from 'vue-i18n'
import { createPinia } from 'pinia'
import PinInput from './PinInput.vue'
import { NodeService } from '@bindings/github.com/yottaapp/yotta/internal/node'

vi.mock('@bindings/github.com/yottaapp/yotta/internal/node', () => ({
  NodeService: {
    AsyncOptions: vi.fn(),
  },
}))

const asyncOptionsMock = NodeService.AsyncOptions as unknown as ReturnType<typeof vi.fn>

function flushPromises() {
  return new Promise((resolve) => setTimeout(resolve, 0))
}

function mountAsyncDropdown() {
  const updates: unknown[] = []
  const selected: unknown[] = []

  const Wrapper = defineComponent({
    setup() {
      return () =>
        h(PinInput, {
          type: 'string',
          widgetKind: 'async-dropdown',
          modelValue: '',
          asyncSource: 'androidADBDevices',
          nodeId: 'node-1',
          specKind: 'AndroidTarget',
          currentInputs: { Serial: 'old' },
          'onUpdate:modelValue': (v: unknown) => updates.push(v),
          onAsyncOptionSelected: (payload: unknown) => selected.push(payload),
        })
    },
  })

  const app = createApp(Wrapper)
  app.use(createPinia())
  app.use(createI18n({ legacy: false, locale: 'zh', messages: { zh: {} } }))
  app.component(
    'UCheckbox',
    defineComponent(() => () => null),
  )
  app.component(
    'UInputNumber',
    defineComponent(() => () => null),
  )
  app.component(
    'USelect',
    defineComponent(() => () => null),
  )
  app.component(
    'UTextarea',
    defineComponent(() => () => null),
  )
  app.component(
    'UInput',
    defineComponent(() => () => null),
  )

  const el = document.createElement('div')
  document.body.appendChild(el)
  app.mount(el)
  return { app, el, updates, selected }
}

describe('PinInput async-dropdown', () => {
  it('loads options with node context', async () => {
    asyncOptionsMock.mockResolvedValueOnce([
      { value: 'adb-1', label: 'Pixel 8', meta: { width: 1080, height: 2400 } },
    ])

    const wrapper = mountAsyncDropdown()
    await flushPromises()
    await nextTick()

    expect(asyncOptionsMock).toHaveBeenCalledWith('node-1', 'AndroidTarget', 'androidADBDevices', {
      Serial: 'old',
    })
    expect(wrapper.el.querySelector('[role="combobox"]')).toBeTruthy()
    expect(wrapper.updates).toEqual([])
    expect(wrapper.selected).toEqual([])

    wrapper.app.unmount()
    wrapper.el.remove()
  })
})
