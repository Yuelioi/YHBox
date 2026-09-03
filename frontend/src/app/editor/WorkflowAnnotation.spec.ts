import { createApp, defineComponent, h, nextTick } from 'vue'
import { createI18n } from 'vue-i18n'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Annotation } from '../../../../contracts/workflow/current/workflow-source'
import WorkflowAnnotation from './WorkflowAnnotation.vue'

const mounted: ReturnType<typeof createApp>[] = []
const iconStub = defineComponent({ setup: () => () => h('i') })

afterEach(() => {
  for (const app of mounted.splice(0)) app.unmount()
  document.body.replaceChildren()
})

describe('WorkflowAnnotation', () => {
  it('renders bounded markdown in a single internal scrolling region', async () => {
    const root = mountAnnotation(
      '## 说明\n\n支持 **重点** 和列表。\n\n- 第一项\n- 第二项\n\n<img src=x onerror=throw(1)>',
    )
    const surface = root.querySelector<HTMLElement>('[data-testid="workflow-annotation"]')!
    const content = root.querySelector<HTMLElement>('[data-testid="workflow-annotation-content"]')!

    expect(surface.classList).toContain('overflow-hidden')
    expect(content.classList).toContain('min-h-0')
    expect(content.classList).toContain('overflow-y-auto')
    await vi.waitFor(() => expect(content.querySelector('h2')?.textContent).toBe('说明'))
    expect(content.querySelector('strong')?.textContent).toBe('重点')
    expect(content.querySelectorAll('li')).toHaveLength(2)
    expect(content.querySelector('img')).toBeNull()
    expect(root.querySelector('textarea')).toBeNull()
  })

  it('switches to a fixed-height editor and returns to markdown preview', async () => {
    const root = mountAnnotation('Existing note')
    root.querySelector<HTMLButtonElement>('[data-testid="workflow-annotation-edit"]')!.click()
    await nextTick()

    const editor = root.querySelector<HTMLTextAreaElement>(
      '[data-testid="workflow-annotation-editor"]',
    )!
    expect(editor).not.toBeNull()
    expect(editor.classList).toContain('overflow-y-auto')
    expect(editor.classList).toContain('resize-none')

    root.querySelector<HTMLButtonElement>('[data-testid="workflow-annotation-edit"]')!.click()
    await nextTick()
    await vi.waitFor(() =>
      expect(root.querySelector('[data-testid="workflow-annotation-content"]')).not.toBeNull(),
    )
    expect(root.querySelector('textarea')).toBeNull()
    await vi.waitFor(() =>
      expect(
        root.querySelector('[data-testid="workflow-annotation-content"]')?.textContent,
      ).toContain('Existing note'),
    )
  })
})

function mountAnnotation(text: string): HTMLElement {
  const root = document.createElement('div')
  document.body.append(root)
  const annotation: Annotation = {
    id: 'note-a',
    text,
    color: 'amber',
    position: { x: 0, y: 0 },
    size: { width: 300, height: 180 },
  }
  const app = createApp(WorkflowAnnotation, { annotation })
  app.component('UIcon', iconStub)
  app.use(
    createI18n({
      legacy: false,
      locale: 'zh',
      messages: {
        zh: {
          workflow: {
            graphs: {
              comment_label: '注释',
              comment_edit: '编辑注释',
              comment_preview: '预览 Markdown',
              comment_placeholder: '使用 Markdown 添加注释…',
            },
          },
        },
      },
    }),
  )
  mounted.push(app)
  app.mount(root)
  return root
}
