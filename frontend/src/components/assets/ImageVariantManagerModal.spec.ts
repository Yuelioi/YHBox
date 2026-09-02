import { createApp, defineComponent, h } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

vi.mock('@/composables/useConfirm', () => ({
  useConfirm: () => ({ confirm: vi.fn(async () => true) }),
}))

vi.mock('@/components/common/BaseModal.vue', () => ({
  default: defineComponent({
    setup(_, { slots }) {
      return () => h('div', [slots.default?.(), slots.footer?.()])
    },
  }),
}))

vi.mock('@/components/common/BlobPreview.vue', () => ({
  default: defineComponent({ setup: () => () => h('div') }),
}))

import ImageVariantManagerModal from './ImageVariantManagerModal.vue'

const mounted: Array<ReturnType<typeof createApp>> = []

afterEach(() => {
  for (const app of mounted.splice(0)) app.unmount()
  document.body.innerHTML = ''
})

function mount(events: string[]): HTMLElement {
  const root = document.createElement('div')
  document.body.append(root)
  const app = createApp(ImageVariantManagerModal, {
    open: true,
    title: '图片变体 · 传送',
    variants: [
      {
        id: '1080p',
        resolution: [1920, 1080],
        blob: { digest: 'sha256:image', mediaType: 'image/png', size: 12 },
      },
    ],
    onAdd: () => events.push('add'),
    onRecapture: () => events.push('recapture'),
    'onUpdate:open': (open: boolean) => events.push(`open:${open}`),
  })
  app.component(
    'UButton',
    defineComponent({
      inheritAttrs: false,
      emits: ['click'],
      setup(_, { attrs, emit, slots }) {
        return () => h('button', { ...attrs, onClick: () => emit('click') }, slots.default?.())
      },
    }),
  )
  app.mount(root)
  mounted.push(app)
  return root
}

describe('ImageVariantManagerModal', () => {
  it('closes after starting an add capture so another modal cannot remain behind its overlay', () => {
    const events: string[] = []
    const root = mount(events)

    const add = [...root.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('assets.templates.add_current_resolution'),
    )
    add?.click()

    expect(events).toEqual(['add', 'open:false'])
  })

  it('closes after starting a selected-variant recapture', () => {
    const events: string[] = []
    const root = mount(events)

    const recapture = root.querySelector<HTMLButtonElement>(
      '[aria-label="assets.templates.recapture_variant"]',
    )
    recapture?.click()

    expect(events).toEqual(['recapture', 'open:false'])
  })
})
