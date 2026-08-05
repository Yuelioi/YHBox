import { createApp, defineComponent, h, nextTick } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  pause: vi.fn(async () => undefined),
  resume: vi.fn(async () => undefined),
}))

vi.mock('@/lib/backend', () => ({
  backend: { hotkeys: { pause: mocks.pause, resume: mocks.resume } },
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

import KeyChordValueEditor from './KeyChordValueEditor.vue'

afterEach(() => {
  document.body.replaceChildren()
  mocks.pause.mockClear()
  mocks.resume.mockClear()
})

describe('KeyChordValueEditor', () => {
  it('records Alt alone on keyup without breaking modifier-led chords', async () => {
    const updates: string[][] = []
    const root = document.createElement('div')
    document.body.append(root)
    const app = createApp(KeyChordValueEditor, {
      modelValue: [],
      'onUpdate:modelValue': (value: string[]) => updates.push(value),
    })
    app.component(
      'UButton',
      defineComponent({
        setup:
          (_, { attrs, slots }) =>
          () =>
            h('button', attrs, slots.default?.()),
      }),
    )
    app.component(
      'UKbd',
      defineComponent({
        setup:
          (_, { slots }) =>
          () =>
            h('kbd', slots.default?.()),
      }),
    )
    app.mount(root)

    root.querySelector<HTMLButtonElement>('button')!.click()
    await vi.waitFor(() => expect(mocks.pause).toHaveBeenCalledOnce())
    await nextTick()

    const listener = root.querySelector<HTMLButtonElement>('button[class*="border-primary"]')!
    listener.dispatchEvent(
      new KeyboardEvent('keydown', { bubbles: true, code: 'AltLeft', altKey: true }),
    )
    expect(updates).toEqual([])

    listener.dispatchEvent(
      new KeyboardEvent('keyup', { bubbles: true, code: 'AltLeft', altKey: false }),
    )
    await vi.waitFor(() => expect(mocks.resume).toHaveBeenCalledOnce())
    expect(updates).toEqual([['ALT']])

    app.unmount()
  })
})
