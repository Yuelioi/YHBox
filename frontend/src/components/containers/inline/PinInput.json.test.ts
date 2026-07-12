import { describe, expect, it, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick } from 'vue'
import { createI18n } from 'vue-i18n'
import { createPinia } from 'pinia'
import PinInput from './PinInput.vue'

vi.mock('@bindings/github.com/yottaapp/yotta/internal/node', () => ({
  NodeService: {
    AsyncOptions: vi.fn(),
  },
}))

function mountJSONInput(modelValue: unknown) {
  const updates: unknown[] = []
  const Wrapper = defineComponent({
    setup() {
      return () =>
        h(PinInput, {
          type: 'any',
          widgetKind: 'json',
          modelValue,
          'onUpdate:modelValue': (v: unknown) => updates.push(v),
        })
    },
  })

  const app = createApp(Wrapper)
  app.use(createPinia())
  app.use(
    createI18n({
      legacy: false,
      locale: 'zh',
      messages: { zh: { inspector: { pin_input_json_invalid: 'JSON 无效' } } },
    }),
  )
  app.component(
    'UTextarea',
    defineComponent({
      props: ['modelValue', 'color', 'placeholder', 'rows', 'size'],
      emits: ['update:modelValue'],
      setup(props, { emit }) {
        return () =>
          h('textarea', {
            value: props.modelValue,
            onInput: (event: Event) =>
              emit('update:modelValue', (event.target as HTMLTextAreaElement).value),
          })
      },
    }),
  )
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
    'UInput',
    defineComponent(() => () => null),
  )

  const el = document.createElement('div')
  document.body.appendChild(el)
  app.mount(el)
  return { app, el, updates }
}

describe('PinInput JSON editor', () => {
  it('renders string scalar values as valid JSON string literals', async () => {
    const wrapper = mountJSONInput('abc')
    await nextTick()

    const textarea = wrapper.el.querySelector('textarea') as HTMLTextAreaElement
    expect(textarea.value).toBe('"abc"')

    wrapper.app.unmount()
    wrapper.el.remove()
  })
})
