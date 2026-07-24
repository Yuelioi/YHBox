import { createApp, nextTick } from 'vue'
import { createI18n } from 'vue-i18n'
import { afterEach, describe, expect, it } from 'vitest'
import authoringDocument from '../../../../contracts/node/current/builtin-authoring'
import type { YottaNodeAuthoringProjection } from '../../../../contracts/node/current/authoring-projection'
import zh from '../../i18n/zh'
import StateDefaultValueEditor from './StateDefaultValueEditor.vue'

const authoring = authoringDocument as unknown as YottaNodeAuthoringProjection

afterEach(() => {
  document.body.replaceChildren()
})

describe('StateDefaultValueEditor', () => {
  it('keeps invalid compound JSON visible and only emits a valid replacement', async () => {
    const type = authoring.body.types.find((candidate) =>
      candidate.typeRef.typeId.endsWith('/filesystem/metadata/v1'),
    )!
    const updates: unknown[] = []
    const root = document.createElement('div')
    document.body.append(root)
    const app = createApp(StateDefaultValueEditor, {
      modelValue: type.stateInitial,
      type,
      'onUpdate:modelValue': (value: unknown) => updates.push(value),
    })
    app.use(createI18n({ legacy: false, locale: 'zh', messages: { zh } }))
    app.mount(root)

    const textarea = root.querySelector<HTMLTextAreaElement>('textarea')!
    textarea.value = '{"path":'
    textarea.dispatchEvent(new Event('input', { bubbles: true }))
    textarea.dispatchEvent(new Event('blur', { bubbles: true }))
    await nextTick()
    expect(root.querySelector('[role="alert"]')?.textContent).toContain('不是有效的 JSON')
    expect(textarea.value).toBe('{"path":')
    expect(updates).toEqual([])

    const next = { ...(type.stateInitial as Record<string, unknown>), path: 'updated.txt' }
    textarea.value = JSON.stringify(next)
    textarea.dispatchEvent(new Event('input', { bubbles: true }))
    textarea.dispatchEvent(new Event('blur', { bubbles: true }))
    await nextTick()
    expect(root.querySelector('[role="alert"]')).toBeNull()
    expect(updates).toEqual([next])
  })
})
