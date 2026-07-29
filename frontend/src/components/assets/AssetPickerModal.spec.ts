import { createApp, defineComponent, h, nextTick } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'

const query = vi.fn(async () => ({
  items: [
    {
      guid: 'template-1',
      kind: 'template',
      name: 'F3',
      description: '',
      category: '异环',
      tags: [],
      variants: [
        {
          resolution: [1280, 720],
          blob: { digest: 'sha256:template', mediaType: 'image/png', size: 128 },
        },
      ],
    },
  ],
  total: 1,
}))

vi.mock('@/stores/assets', () => ({
  useAssetsStore: () => ({
    epoch: 0,
    recentGUIDs: [],
    query,
    markUsed: vi.fn(),
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/components/common/BaseModal.vue', () => ({
  default: defineComponent({
    setup(_, { slots }) {
      return () => h('div', { 'data-testid': 'modal' }, slots.default?.())
    },
  }),
}))

vi.mock('@/components/common/BlobPreview.vue', () => ({
  default: defineComponent({
    setup() {
      return () => h('div')
    },
  }),
}))

vi.mock('@/components/common/EmptyState.vue', () => ({
  default: defineComponent({
    setup() {
      return () => h('div')
    },
  }),
}))

import AssetPickerModal from './AssetPickerModal.vue'

const mounted: Array<ReturnType<typeof createApp>> = []

afterEach(() => {
  for (const app of mounted.splice(0)) app.unmount()
  document.body.innerHTML = ''
  query.mockClear()
})

describe('AssetPickerModal', () => {
  it('selects the preferred template variant when the user clicks its card', async () => {
    const root = document.createElement('div')
    document.body.append(root)
    const app = createApp(AssetPickerModal, { open: true, kind: 'template' })
    app.component(
      'UButton',
      defineComponent({
        inheritAttrs: false,
        props: { disabled: Boolean, label: String },
        emits: ['click'],
        setup(props, { attrs, emit, slots }) {
          return () =>
            h(
              'button',
              {
                ...attrs,
                disabled: props.disabled,
                onClick: () => emit('click'),
              },
              slots.default?.() ?? props.label,
            )
        },
      }),
    )
    for (const name of ['UInput', 'USelect']) {
      app.component(name, defineComponent({ setup: () => () => h('input') }))
    }
    for (const name of ['UBadge', 'UIcon', 'USkeleton']) {
      app.component(
        name,
        defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('span', slots.default?.()),
        }),
      )
    }
    mounted.push(app)
    app.mount(root)
    await Promise.resolve()
    await nextTick()

    const card = root.querySelector('article') as HTMLElement
    expect(card).toBeTruthy()
    card.click()
    await nextTick()

    expect(card.getAttribute('aria-selected')).toBe('true')
    const confirm = [...root.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('assetPicker.use_template'),
    )
    expect(confirm).toBeTruthy()
    expect(confirm?.disabled).toBe(false)
  })

  it('offers in-context template capture and closes before handing control back', async () => {
    const root = document.createElement('div')
    document.body.append(root)
    const onCapture = vi.fn()
    const onUpdateOpen = vi.fn()
    const app = createApp(AssetPickerModal, {
      open: true,
      kind: 'template',
      onCapture,
      'onUpdate:open': onUpdateOpen,
    })
    app.component(
      'UButton',
      defineComponent({
        inheritAttrs: false,
        props: { disabled: Boolean, label: String },
        emits: ['click'],
        setup(props, { attrs, emit, slots }) {
          return () =>
            h(
              'button',
              { ...attrs, disabled: props.disabled, onClick: () => emit('click') },
              slots.default?.() ?? props.label,
            )
        },
      }),
    )
    for (const name of ['UInput', 'USelect']) {
      app.component(name, defineComponent({ setup: () => () => h('input') }))
    }
    for (const name of ['UBadge', 'UIcon', 'USkeleton']) {
      app.component(
        name,
        defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('span', slots.default?.()),
        }),
      )
    }
    mounted.push(app)
    app.mount(root)
    await Promise.resolve()
    await nextTick()

    const capture = [...root.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('assetPicker.capture_template'),
    )
    expect(capture).toBeTruthy()
    capture?.click()
    await nextTick()

    expect(onUpdateOpen).toHaveBeenCalledWith(false)
    expect(onCapture).toHaveBeenCalledTimes(1)
  })

  it('offers compatible resources from the current workflow', async () => {
    const root = document.createElement('div')
    document.body.append(root)
    const onSelectWorkflow = vi.fn()
    const app = createApp(AssetPickerModal, {
      open: true,
      kind: 'template',
      onSelectWorkflow,
      resources: [
        {
          id: 'image-22d96f23-8318-4336-969d-708441b37e64',
          kind: 'image',
          name: 'F1',
          image: {
            variants: [
              {
                id: 'variant-f1',
                resolution: [1920, 1080],
                bbox: [0, 0, 1920, 1080],
                blob: { digest: 'sha256:f1', mediaType: 'image/png', size: 256 },
              },
            ],
          },
        },
      ],
    })
    app.component(
      'UButton',
      defineComponent({
        inheritAttrs: false,
        props: { disabled: Boolean, label: String },
        emits: ['click'],
        setup(props, { attrs, emit, slots }) {
          return () =>
            h(
              'button',
              {
                ...attrs,
                disabled: props.disabled,
                onClick: () => emit('click'),
              },
              slots.default?.() ?? props.label,
            )
        },
      }),
    )
    for (const name of ['UInput', 'USelect']) {
      app.component(name, defineComponent({ setup: () => () => h('input') }))
    }
    for (const name of ['UBadge', 'UIcon', 'USkeleton']) {
      app.component(
        name,
        defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('span', slots.default?.()),
        }),
      )
    }
    mounted.push(app)
    app.mount(root)
    await Promise.resolve()
    await nextTick()

    expect(root.textContent).toContain('F1')
    expect(root.textContent).toContain('1920×1080')

    const card = root.querySelector('article') as HTMLElement
    card.click()
    await nextTick()
    const confirm = [...root.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('assetPicker.use_template'),
    )
    confirm?.click()
    await nextTick()

    expect(onSelectWorkflow).toHaveBeenCalledWith({
      resourceId: 'image-22d96f23-8318-4336-969d-708441b37e64',
      variantId: 'variant-f1',
    })
  })
})
