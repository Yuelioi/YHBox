import { createI18n } from 'vue-i18n'
import { createApp, defineComponent, nextTick } from 'vue'
import { createPinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ContainersTab from './ContainersTab.vue'

const routerPush = vi.fn()

const sampleContainers = Array.from({ length: 2 }, (_, idx) => ({
  id: `c-${idx}`,
  name: `Container ${idx + 1}`,
  description: `Desc ${idx + 1}`,
  category: idx === 0 ? 'daily' : 'combat',
  tags: idx === 0 ? ['farm', 'daily'] : ['combat'],
  graph: { nodes: [{ id: 'start' }] },
  createdAt: '2026-07-01T00:00:00Z',
  updatedAt: '2026-07-02T00:00:00Z',
}))

vi.mock('@/lib/backend', () => ({
  backend: {
    containers: {
      list: vi.fn(async () => sampleContainers),
      create: vi.fn(),
      run: vi.fn(),
      stopAll: vi.fn(),
      delete_: vi.fn(),
      deleteMany: vi.fn(),
      exportPackage: vi.fn(),
      pickExportPath: vi.fn(),
    },
  },
}))

vi.mock('@nuxt/ui/composables', () => ({
  useToast: () => ({ add: vi.fn() }),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ path: '/', query: {}, params: {} }),
  useRouter: () => ({ push: routerPush }),
}))

function stub(template = '<div><slot /></div>') {
  return defineComponent({ template })
}

function mountContainersTab() {
  const app = createApp(ContainersTab)
  app.use(createPinia())
  app.use(createI18n({ legacy: false, locale: 'zh', messages: { zh: {} } }))
  app.component('UInput', stub('<input />'))
  app.component('USelect', stub())
  app.component('UInputMenu', stub())
  app.component('USelectMenu', stub())
  app.component('UDropdownMenu', stub('<div><slot /></div>'))
  app.component('UButton', stub('<button><slot /></button>'))
  app.component('UCheckbox', stub('<input type="checkbox" />'))
  app.component('UPagination', stub('<nav />'))
  app.component('UIcon', stub('<span />'))
  app.component('UFormField', stub())
  app.component('UTextarea', stub('<textarea />'))
  app.component('UModal', stub('<div><slot name="content" /></div>'))

  const el = document.createElement('div')
  document.body.appendChild(el)
  app.mount(el)
  return { app, el }
}

describe('ContainersTab layout', () => {
  beforeEach(() => {
    localStorage.clear()
    localStorage.setItem('containers.viewMode', 'list')
    routerPush.mockClear()
  })

  it('keeps filters separate from actions and reserves a stable paginated content area', async () => {
    const { app, el } = mountContainersTab()
    try {
      await nextTick()
      await nextTick()

      const toolbar = el.querySelector('[data-testid="containers-toolbar"]')
      const filterbar = el.querySelector('[data-testid="containers-filterbar"]')
      const content = el.querySelector('[data-testid="containers-content"]')
      const pagination = el.querySelector('[data-testid="containers-pagination"]')
      const tagQuickFilter = el.querySelector('[data-testid="containers-tag-quick-filter"]')
      const columnSelector = el.querySelector('[data-testid="containers-column-selector"]')
      const selectModeButton = el.querySelector('[data-testid="containers-select-mode"]')
      const batchActions = el.querySelector('[data-testid="containers-batch-actions"]')
      const firstCheckbox = el.querySelector('[data-testid="container-checkbox-c-0"]')
      const row = el.querySelector('[data-testid="container-row-c-0"]') as HTMLElement | null
      const listHeader = el.querySelector('[data-testid="containers-list-header"]')
      const rowRun = el.querySelector('[data-testid="container-row-run-c-0"]')
      const rowMore = el.querySelector('[data-testid="container-row-more-c-0"]')
      const rowEdit = el.querySelector('[data-testid="container-row-edit-c-0"]')
      const rowExport = el.querySelector('[data-testid="container-row-export-c-0"]')
      const rowDelete = el.querySelector('[data-testid="container-row-delete-c-0"]')

      expect(toolbar).toBeTruthy()
      expect(filterbar).toBeTruthy()
      expect(content).toBeTruthy()
      expect(pagination).toBeTruthy()
      expect(tagQuickFilter).toBeNull()
      expect(columnSelector).toBeTruthy()
      expect(selectModeButton).toBeNull()
      expect(batchActions).toBeTruthy()
      expect(firstCheckbox).toBeTruthy()
      expect(firstCheckbox!.getAttribute('aria-label')).toBeTruthy()
      expect(listHeader).toBeTruthy()
      expect(listHeader!.className).toContain('sticky')
      expect(listHeader!.className).toContain('top-0')
      expect(rowRun).toBeTruthy()
      expect(rowMore).toBeTruthy()
      expect(rowEdit).toBeNull()
      expect(rowExport).toBeNull()
      expect(rowDelete).toBeNull()
      expect(toolbar!.contains(filterbar)).toBe(false)
      expect(filterbar!.className).toContain('flex-wrap')
      expect(filterbar!.className).toContain('shrink-0')
      expect(content!.className).toContain('flex-1')
      expect(content!.className).toContain('overflow-y-auto')
      expect(content!.contains(pagination)).toBe(false)
      expect(pagination!.className).toContain('shrink-0')
      expect(pagination!.className).not.toContain('sticky')
      expect(row).toBeTruthy()
      row!.dispatchEvent(new MouseEvent('dblclick', { bubbles: true }))
      expect(routerPush).toHaveBeenCalledWith('/containers/c-0/edit')
    } finally {
      app.unmount()
      el.remove()
    }
  })

  it('does not force horizontal overflow after most list columns are hidden', async () => {
    localStorage.setItem('containers.listColumns', JSON.stringify(['category']))
    const { app, el } = mountContainersTab()
    try {
      await nextTick()
      await nextTick()

      const table = el.querySelector('[data-testid="containers-list-table"]')

      expect(table).toBeTruthy()
      expect(table!.className).not.toContain('min-w-[')
      expect(table!.className).toContain('w-full')
    } finally {
      app.unmount()
      el.remove()
    }
  })
})
