import { createApp, defineComponent, h, nextTick } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'

const {
  push,
  querySources,
  listInstallations,
  getInstallationReadiness,
  createSourceWithMetadata,
  batchUpdateSourceMetadata,
  queryAssets,
  batchUpdateAssetMeta,
} = vi.hoisted(() => ({
  push: vi.fn(),
  querySources: vi.fn(async (query: { page: number }) => ({
    items: [
      {
        workflowId: `workflow-${query.page}`,
        name: 'Daily report',
        description: 'Exports the daily report',
        category: 'Operations',
        tags: ['Daily', 'Report'],
        nodeCount: 8,
        revision: 3,
        sourceHash: 'sha256:source',
        sourceJson: '',
      },
    ],
    total: 40,
    page: query.page,
    pageSize: 20,
    categories: [{ value: 'Operations', count: 20 }],
    tags: [{ value: 'Daily', count: 20 }],
  })),
  listInstallations: vi.fn(async () => [
    {
      installationId: 'installation-1',
      releaseId: 'sha256:release',
      name: 'Installed report',
      lifecycle: 'active',
      createdAt: '2026-07-26T00:00:00Z',
      updatedAt: '2026-07-26T00:00:00Z',
    },
  ]),
  getInstallationReadiness: vi.fn(async () => ({
    installationId: 'installation-1',
    releaseId: 'sha256:release',
    lifecycle: 'active',
    lifecycleAllowsExecution: true,
    runAllowed: true,
    scheduleAllowed: false,
    blockers: [
      {
        kind: 'schedule-consent',
        requirementId: 'scheduled-execution',
        expected: 'sha256:release',
        blocks: ['schedule'],
        action: 'grant-schedule-consent',
      },
    ],
  })),
  createSourceWithMetadata: vi.fn(async () => ({
    workflowId: 'workflow-created',
    name: 'Created workflow',
    description: '',
    category: '',
    tags: [],
    nodeCount: 1,
    revision: 0,
    sourceHash: 'sha256:created',
    sourceJson: '',
  })),
  batchUpdateSourceMetadata: vi.fn(async (requests: Array<{ workflowId: string }>) =>
    requests.map((request) => ({ workflowId: request.workflowId, updated: true, error: '' })),
  ),
  queryAssets: vi.fn(async (query: { kind: string; page: number }) => ({
    items: [
      query.kind === 'macro'
        ? {
            guid: `macro-${query.page}`,
            kind: 'macro',
            name: 'Login macro',
            description: 'Opens the login dialog',
            category: 'Account',
            tags: ['Login', 'Stable'],
            variantCount: 0,
            variants: [],
            blob: {
              digest: 'sha256:macro',
              mediaType: 'application/octet-stream',
              size: 256,
            },
            createdAt: '2026-07-19T00:00:00Z',
          }
        : query.kind === 'clip'
          ? {
              guid: `clip-${query.page}`,
              kind: 'clip',
              name: 'Full camera turn',
              description: 'Preserves the raw trajectory',
              category: 'Combat',
              tags: ['Precise'],
              variantCount: 0,
              variants: [],
              blob: {
                digest: 'sha256:clip',
                mediaType: 'application/octet-stream',
                size: 512,
              },
              createdAt: '2026-07-19T00:00:00Z',
            }
          : {
              guid: `template-${query.page}`,
              kind: 'template',
              name: 'Login button',
              description: 'Primary login button',
              category: 'Account',
              tags: ['Login'],
              variantCount: 1,
              variants: [
                {
                  resolution: [1280, 720],
                  blob: { digest: 'sha256:template', mediaType: 'image/png', size: 128 },
                },
              ],
              thumbnail: { digest: 'sha256:template', mediaType: 'image/png', size: 128 },
              createdAt: '2026-07-19T00:00:00Z',
            },
    ],
    total: 40,
    page: query.page,
    pageSize: 20,
    revision: 1,
    categories: [{ value: 'Account', count: 40 }],
    tags: [{ value: 'Login', count: 40 }],
  })),
  batchUpdateAssetMeta: vi.fn(async (requests: Array<{ guid: string }>) =>
    requests.map((request) => ({ guid: request.guid, updated: true, error: '' })),
  ),
}))

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRouter: () => ({
      push,
      resolve: (to: unknown) => ({ href: String(to), matched: [], meta: {} }),
    }),
    useRoute: () => ({
      path: '/',
      fullPath: '/',
      query: {},
      params: {},
      hash: '',
      name: 'library-test',
      meta: {},
      matched: [],
    }),
  }
})
vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})
vi.mock('@nuxt/ui/composables', () => ({ useToast: () => ({ add: vi.fn() }) }))
vi.mock('@/composables/useConfirm', () => ({
  useConfirm: () => ({ confirm: vi.fn(async () => true) }),
}))
vi.mock('@/app/transport/workflow', () => ({
  workflowTransport: {
    querySources,
    listInstallations,
    getInstallationReadiness,
    listSourceRecoveries: vi.fn(async () => []),
    createSourceWithMetadata,
    batchUpdateSourceMetadata,
    updateSourceMetadata: vi.fn(),
    startRun: vi.fn(),
  },
}))
vi.mock('@/stores/settings', () => ({
  useSettingsStore: () => ({ data: { automation: { targets: [] } } }),
}))
vi.mock('@/stores/assets', () => ({
  useAssetsStore: () => ({ query: queryAssets }),
}))
vi.mock('@/stores/recording', () => ({
  useRecordingStore: () => ({
    state: { phase: 'idle', pending: null },
    invocation: '',
    completionFailure: null,
    reconcile: vi.fn(async () => undefined),
    claimInvocation: vi.fn(),
  }),
}))
vi.mock('@/composables/useRecordingStart', () => ({
  useRecordingStart: () => ({ starting: false, start: vi.fn() }),
}))
vi.mock('@/lib/backend', () => ({
  backend: {
    assets: {
      batchUpdateMeta: batchUpdateAssetMeta,
      batchDelete: vi.fn(),
      previewCleanup: vi.fn(),
      commitCleanup: vi.fn(),
      updateMeta: vi.fn(),
      delete_: vi.fn(),
      removeVariant: vi.fn(),
    },
    tools: { openScreenPicker: vi.fn() },
  },
}))

vi.mock('@/components/common/BaseModal.vue', () => ({
  default: defineComponent({
    props: { open: Boolean },
    setup(props, { slots }) {
      return () =>
        props.open
          ? h('section', { 'data-testid': 'base-modal' }, [slots.default?.(), slots.footer?.()])
          : null
    },
  }),
}))
vi.mock('@/components/common/BlobPreview.vue', () => ({
  default: defineComponent({ setup: () => () => h('div', { 'data-testid': 'blob-preview' }) }),
}))
vi.mock('@/components/common/EmptyState.vue', () => ({
  default: defineComponent({
    setup:
      (_, { slots }) =>
      () =>
        h('div', slots.action?.()),
  }),
}))
vi.mock('@/components/recording/MacroActionEditor.vue', () => ({
  default: defineComponent({ setup: () => () => h('div') }),
}))

import AssetsView from './AssetsView.vue'
import WorkflowsView from './WorkflowsView.vue'

const mounted: Array<ReturnType<typeof createApp>> = []

afterEach(() => {
  vi.useRealTimers()
  for (const app of mounted.splice(0)) app.unmount()
  document.body.innerHTML = ''
  push.mockClear()
  querySources.mockClear()
  createSourceWithMetadata.mockClear()
  batchUpdateSourceMetadata.mockClear()
  queryAssets.mockClear()
  batchUpdateAssetMeta.mockClear()
})

describe('library management views', () => {
  it('renders workflow metadata, creates through a modal, and uses numbered paging', async () => {
    const root = await mountView(WorkflowsView)

    expect(root.textContent).toContain('Daily report')
    expect(root.textContent).toContain('Installed report')
    expect(root.textContent).toContain('workflow.installation.status_ready')
    expect(root.textContent).toContain('Exports the daily report')
    expect(root.textContent).toContain('Operations')
    expect(root.textContent).toContain('Daily')
    expect(root.textContent).toContain('8')

    buttonByText(root, 'workflow.list.new_workflow').click()
    await nextTick()
    const input = root.querySelector('[data-testid="workflow-create-name"]') as HTMLInputElement
    expect(input).toBeTruthy()
    input.value = 'Created workflow'
    input.dispatchEvent(new Event('input', { bubbles: true }))
    await nextTick()
    ;(root.querySelector('[data-testid="workflow-create-submit"]') as HTMLButtonElement).click()
    await flushView()

    expect(createSourceWithMetadata).toHaveBeenCalledWith({
      name: 'Created workflow',
      description: '',
      category: '',
      tags: [],
    })
    expect(push).toHaveBeenCalledWith({ path: '/workflows/workflow-created/edit', query: {} })

    buttonByText(root, '2').click()
    await flushView()
    expect(querySources).toHaveBeenLastCalledWith(
      expect.objectContaining({ page: 2, pageSize: 20 }),
    )
  })

  it('keeps macro, precise recording, and template contexts exclusive in one dense resource list', async () => {
    const root = await mountView(AssetsView)

    expect(root.textContent).toContain('Login macro')
    expect(root.textContent).toContain('Opens the login dialog')
    expect(root.textContent).toContain('Account')
    expect(root.textContent).toContain('Stable')
    expect(root.querySelector('[data-testid="grid-view"]')).toBeNull()

    buttonByText(root, 'assets.tabs.clips').click()
    await flushView()
    expect(queryAssets).toHaveBeenLastCalledWith(
      expect.objectContaining({ kind: 'clip', page: 1, pageSize: 20 }),
    )
    expect(root.textContent).toContain('Full camera turn')
    expect(root.textContent).not.toContain('Login macro')

    buttonByText(root, 'assets.tabs.templates').click()
    await flushView()

    expect(queryAssets).toHaveBeenLastCalledWith(
      expect.objectContaining({ kind: 'template', page: 1, pageSize: 20 }),
    )
    expect(root.textContent).toContain('Login button')
    expect(root.textContent).not.toContain('Login macro')

    buttonByText(root, '2').click()
    await flushView()
    expect(queryAssets).toHaveBeenLastCalledWith(
      expect.objectContaining({ kind: 'template', page: 2, pageSize: 20 }),
    )
  })

  it('replaces workflow filters with a contextual batch toolbar', async () => {
    const root = await mountView(WorkflowsView)

    rowCheckbox(root).click()
    await flushView()

    expect(root.querySelector('[data-testid="library-selection-toolbar"]')).toBeTruthy()
    const batchButton = root.querySelector(
      '[data-testid="workflow-batch-metadata"]',
    ) as HTMLButtonElement
    expect(batchButton).toBeTruthy()
    batchButton.click()
    await flushView()
    expect(root.textContent).toContain('batchMetadata.description')
  })

  it('uses the same contextual batch toolbar for assets', async () => {
    const root = await mountView(AssetsView)

    rowCheckbox(root).click()
    await flushView()

    expect(root.querySelector('[data-testid="library-selection-toolbar"]')).toBeTruthy()
    const batchButton = root.querySelector(
      '[data-testid="asset-batch-metadata"]',
    ) as HTMLButtonElement
    expect(batchButton).toBeTruthy()
    batchButton.click()
    await flushView()
    expect(root.textContent).toContain('batchMetadata.description')
  })

  it('automatically dismisses successful library operation feedback', async () => {
    vi.useFakeTimers()
    const root = await mountView(AssetsView)

    rowCheckbox(root).click()
    await flushView()
    buttonByText(root, 'assets.batch_delete').click()
    await flushView()

    expect(root.textContent).toContain('assets.batch_delete_result')
    vi.advanceTimersByTime(4_001)
    await flushView()
    expect(root.textContent).not.toContain('assets.batch_delete_result')
  })
})

async function mountView(
  component: typeof WorkflowsView | typeof AssetsView,
): Promise<HTMLElement> {
  const root = document.createElement('div')
  document.body.append(root)
  const app = createApp(component)
  registerUI(app)
  mounted.push(app)
  app.mount(root)
  await flushView()
  return root
}

async function flushView(): Promise<void> {
  for (let index = 0; index < 4; index++) {
    await Promise.resolve()
    await nextTick()
  }
}

function buttonByText(root: HTMLElement, value: string): HTMLButtonElement {
  const button = [...root.querySelectorAll('button')].find((candidate) =>
    candidate.textContent?.includes(value),
  )
  if (!button) throw new Error(`button ${value} not found`)
  return button
}

function rowCheckbox(root: HTMLElement): HTMLButtonElement {
  const checkbox = root.querySelectorAll<HTMLButtonElement>('button[role="checkbox"]')[1]
  if (!checkbox) throw new Error('row checkbox not found')
  return checkbox
}

function registerUI(app: ReturnType<typeof createApp>): void {
  app.component(
    'UButton',
    defineComponent({
      inheritAttrs: false,
      props: { label: String, disabled: Boolean },
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
  app.component(
    'UInput',
    defineComponent({
      inheritAttrs: false,
      props: { modelValue: String },
      emits: ['update:modelValue'],
      setup(props, { attrs, emit }) {
        return () =>
          h('input', {
            ...attrs,
            value: props.modelValue,
            onInput: (event: Event) =>
              emit('update:modelValue', (event.target as HTMLInputElement).value),
          })
      },
    }),
  )
  app.component('UTextarea', defineComponent({ setup: () => () => h('textarea') }))
  app.component('UInputMenu', defineComponent({ setup: () => () => h('input') }))
  app.component('USelect', defineComponent({ setup: () => () => h('select') }))
  app.component(
    'UCheckbox',
    defineComponent({
      inheritAttrs: false,
      props: { modelValue: Boolean },
      emits: ['update:modelValue'],
      setup(props, { attrs, emit }) {
        return () =>
          h('input', {
            ...attrs,
            type: 'checkbox',
            checked: props.modelValue,
            onChange: (event: Event) =>
              emit('update:modelValue', (event.target as HTMLInputElement).checked),
          })
      },
    }),
  )
  app.component(
    'UDropdownMenu',
    defineComponent({
      setup:
        (_, { slots }) =>
        () =>
          h('div', slots.default?.()),
    }),
  )
  app.component(
    'UPagination',
    defineComponent({
      emits: ['update:page'],
      setup(_, { emit }) {
        return () =>
          h('button', { 'data-testid': 'next-page', onClick: () => emit('update:page', 2) }, '2')
      },
    }),
  )
  app.component(
    'RouterLink',
    defineComponent({
      setup:
        (_, { slots }) =>
        () =>
          h('a', slots.default?.()),
    }),
  )
  app.component(
    'UFormField',
    defineComponent({
      setup:
        (_, { slots }) =>
        () =>
          h('label', slots.default?.()),
    }),
  )
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
}
