import { createApp, nextTick } from 'vue'
import { createI18n } from 'vue-i18n'
import { describe, expect, it, vi } from 'vitest'
import TemplateDetailPanel from './TemplateDetailPanel.vue'

vi.mock('@/lib/backend', () => ({
  backend: {
    assets: {
      get: vi.fn().mockResolvedValue({
        guid: 'tpl-1',
        variants: [{ resolution: [1920, 1080], blob: 'blob-1' }],
      }),
      currentResolution: vi.fn().mockResolvedValue([1782, 1427]),
      pickVariant: vi.fn().mockResolvedValue({ index: 0, exact: false }),
    },
  },
}))

vi.mock('@/stores/templates', () => ({
  useTemplatesStore: () => ({
    map: {
      'tpl-1': {
        guid: 'tpl-1',
        name: '测试模板',
        description: '',
        category: '',
        tags: [],
        firstBlobSha: 'blob-1',
      },
    },
    readBlobDataURL: vi.fn().mockResolvedValue(undefined),
  }),
}))

vi.mock('@/composables/useConfirm', () => ({
  useConfirm: () => ({ confirm: vi.fn() }),
}))

vi.mock('@nuxt/ui/composables', () => ({
  useToast: () => ({ add: vi.fn() }),
}))

vi.mock('@/components/common/BaseModal.vue', () => ({
  default: { template: '<div><slot /></div>' },
}))

describe('TemplateDetailPanel setup', () => {
  it('mounts immediate detail watchers without temporal-dead-zone errors', async () => {
    const errors: unknown[] = []
    const el = document.createElement('div')
    document.body.appendChild(el)
    const app = createApp(TemplateDetailPanel, {
      guid: 'tpl-1',
      containerId: 'container-1',
    })
    app.use(
      createI18n({
        legacy: false,
        locale: 'zh',
        missingWarn: false,
        fallbackWarn: false,
        messages: {
          zh: {
            common: { loading: '加载中' },
            template: {
              capture: { preview: '预览' },
              detail: { empty: '未选择模板', view_large: '查看大图' },
              picker: {
                variants_label: '分辨率变体',
                current_window: '当前窗口',
                scaled_from: '运行时用 {res} 缩放',
                add_variant: '新增 {res}',
                recapture: '重拍',
                window_not_open: '窗口未开',
              },
            },
          },
        },
      }),
    )
    app.config.errorHandler = (error) => errors.push(error)

    try {
      app.mount(el)
      await nextTick()
      await Promise.resolve()
      await nextTick()
      expect(errors).toEqual([])
      expect(el.textContent).toContain('1920×1080')
    } finally {
      app.unmount()
      el.remove()
    }
  })
})
