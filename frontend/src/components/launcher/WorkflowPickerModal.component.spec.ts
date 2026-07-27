import ui from '@nuxt/ui/vue-plugin'
import { createApp, defineComponent, h, nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  querySources: vi.fn(),
}))

vi.mock('@/app/transport/workflow', () => ({
  workflowTransport: {
    querySources: mocks.querySources,
  },
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${JSON.stringify(params)}` : key,
    }),
  }
})

vi.mock('@/components/common/BaseModal.vue', async () => {
  const { defineComponent, h } = await import('vue')
  return {
    default: defineComponent({
      props: { open: Boolean },
      setup:
        (props, { slots }) =>
        () =>
          props.open ? h('div', [slots.default?.(), slots.footer?.()]) : null,
    }),
  }
})

vi.mock('@/components/common/AdaptiveSelect.vue', async () => {
  const { defineComponent, h } = await import('vue')
  return {
    default: defineComponent({
      setup: () => () => h('div', { 'data-testid': 'category-filter' }),
    }),
  }
})

import WorkflowPickerModal from './WorkflowPickerModal.vue'

const mounted: Array<ReturnType<typeof createApp>> = []

beforeEach(() => {
  mocks.querySources.mockReset()
  mocks.querySources.mockImplementation(async (query: { page: number }) => ({
    items: [
      {
        workflowId: query.page === 1 ? 'alpha' : 'beta',
        name: query.page === 1 ? '自动钓鱼' : '截图归档',
        description: '',
        category: query.page === 1 ? '游戏' : '工具',
        tags: [],
        createdAt: '',
        updatedAt: '',
        nodeCount: 1,
        revision: 1,
        sourceHash: '',
        sourceJson: '',
      },
    ],
    total: 31,
    page: query.page,
    pageSize: 30,
    categories: [{ value: '游戏', count: 1 }],
    tags: [],
  }))
})

afterEach(() => {
  for (const app of mounted.splice(0)) app.unmount()
  document.body.innerHTML = ''
})

describe('WorkflowPickerModal', () => {
  it('searches by page, preserves cross-page selection, and allows an existing workflow again', async () => {
    const added: unknown[][] = []
    const root = document.createElement('div')
    document.body.append(root)
    const app = createApp(
      defineComponent({
        setup: () => () =>
          h(WorkflowPickerModal, {
            open: true,
            addedCounts: { alpha: 2 },
            onAdd: (workflows: unknown[]) => added.push(workflows),
          }),
      }),
    )
    app.use(ui)
    mounted.push(app)
    app.mount(root)

    await vi.waitFor(() => expect(mocks.querySources).toHaveBeenCalledOnce())
    expect(mocks.querySources).toHaveBeenLastCalledWith(
      expect.objectContaining({ page: 1, pageSize: 30, sort: 'updated_desc' }),
    )
    expect(root.textContent).toContain('settingsLauncher.picker_added_count')

    const search = root.querySelector(
      'input[placeholder="settingsLauncher.picker_search"]',
    ) as HTMLInputElement
    search.value = 'fish'
    search.dispatchEvent(new Event('input', { bubbles: true }))
    await vi.waitFor(() =>
      expect(mocks.querySources).toHaveBeenLastCalledWith(
        expect.objectContaining({ search: 'fish', page: 1 }),
      ),
    )

    const first = [...root.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('自动钓鱼'),
    )
    expect(first).toBeTruthy()
    first?.click()
    await nextTick()

    const next = root.querySelector(
      'button[aria-label="settingsLauncher.picker_next"]',
    ) as HTMLButtonElement
    expect(next).toBeTruthy()
    next.click()
    await vi.waitFor(() =>
      expect(mocks.querySources).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 })),
    )

    const second = [...root.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('截图归档'),
    )
    expect(second).toBeTruthy()
    second?.click()
    await nextTick()

    const add = [...root.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('settingsLauncher.picker_add_selected'),
    )
    expect(add).toBeTruthy()
    add?.click()

    expect(added).toHaveLength(1)
    expect(
      (added[0] as Array<{ workflowId: string }>).map((workflow) => workflow.workflowId),
    ).toEqual(['alpha', 'beta'])
    expect(search).toBeTruthy()
  })
})
