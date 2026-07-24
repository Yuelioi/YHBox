import { createApp, defineComponent, h, nextTick } from 'vue'
import { createI18n } from 'vue-i18n'
import { afterEach, describe, expect, it, vi } from 'vitest'
import authoringDocument from '../../../../contracts/node/current/builtin-authoring'
import type { YottaNodeAuthoringProjection } from '../../../../contracts/node/current/authoring-projection'
import type { Variable } from '../../../../contracts/workflow/current/workflow-source'
import zh from '../../i18n/zh'
import type { EditorCommand } from './EditorSession'
import { buildStateTypeChoices } from './stateVariableTypes'
import WorkflowStatePanel from './WorkflowStatePanel.vue'

vi.mock('@/components/common/AdaptiveSelect.vue', async () => {
  const { defineComponent, h } = await import('vue')
  return {
    default: defineComponent({
      inheritAttrs: true,
      props: {
        modelValue: { type: [String, Number, Boolean], default: '' },
        items: { type: Array, default: () => [] },
        labelKey: { type: String, default: 'label' },
        valueKey: { type: String, default: 'value' },
      },
      emits: ['update:modelValue'],
      setup(props, { attrs, emit }) {
        return () =>
          h(
            'select',
            {
              ...attrs,
              value: props.modelValue,
              onChange: (event: Event) =>
                emit('update:modelValue', (event.target as HTMLSelectElement).value),
            },
            (props.items as Array<Record<string, unknown>>).map((item) =>
              h('option', { value: item[props.valueKey] as string }, String(item[props.labelKey])),
            ),
          )
      },
    }),
  }
})

vi.mock('@/app/editor/StateDefaultValueEditor.vue', async () => {
  const { defineComponent, h } = await import('vue')
  return {
    default: defineComponent({
      props: { modelValue: { default: null } },
      emits: ['update:modelValue'],
      setup:
        (props, { emit }) =>
        () =>
          h('textarea', {
            value: JSON.stringify(props.modelValue),
            onChange: (event: Event) =>
              emit('update:modelValue', JSON.parse((event.target as HTMLTextAreaElement).value)),
          }),
    }),
  }
})

const authoring = authoringDocument as unknown as YottaNodeAuthoringProjection
const stateTypeChoices = buildStateTypeChoices(authoring.body.types)

afterEach(() => {
  document.body.replaceChildren()
})

describe('WorkflowStatePanel', () => {
  it.each(stateTypeChoices)('adds the $id state type from the visible form', async (choice) => {
    const commands: EditorCommand[] = []
    const root = mount((command) => commands.push(command))
    const name = root.querySelector<HTMLInputElement>('input')
    const type = root.querySelector<HTMLSelectElement>('select')
    const add = root.querySelector<HTMLButtonElement>('[aria-label="添加状态变量"]')

    expect(name).not.toBeNull()
    expect(type).not.toBeNull()
    expect(add).not.toBeNull()
    name!.value = 'state'
    name!.dispatchEvent(new Event('input', { bubbles: true }))
    type!.value = choice.id
    type!.dispatchEvent(new Event('change', { bubbles: true }))
    await nextTick()
    add!.click()
    await nextTick()

    expect(commands).toHaveLength(1)
    expect(commands[0]).toMatchObject({
      kind: 'add-state-variable',
      name: 'state',
      type: choice.expression,
      defaultValue: choice.defaultValue,
    })
  })

  it('updates an existing compound state default from the visible editor', async () => {
    const choice = stateTypeChoices.find((candidate) =>
      candidate.id.endsWith('/filesystem/metadata/v1'),
    )!
    const variable: Variable = {
      name: 'metadata',
      type: structuredClone(choice.expression),
      default: structuredClone(choice.defaultValue),
    }
    const commands: EditorCommand[] = []
    const root = mount((command) => commands.push(command), [variable])
    const editors = root.querySelectorAll<HTMLTextAreaElement>('textarea')
    expect(editors).toHaveLength(2)
    const next = { ...(choice.defaultValue as Record<string, unknown>), path: 'updated.txt' }
    editors[1]!.value = JSON.stringify(next)
    editors[1]!.dispatchEvent(new Event('change', { bubbles: true }))
    await nextTick()

    expect(commands).toEqual([
      {
        kind: 'update-state-variable',
        name: 'metadata',
        type: choice.expression,
        defaultValue: next,
      },
    ])
  })
})

function mount(
  onCommand: (command: EditorCommand) => void,
  variables: Variable[] = [],
): HTMLElement {
  const root = document.createElement('div')
  document.body.append(root)
  const app = createApp(WorkflowStatePanel, {
    variables,
    types: authoring.body.types,
    references: {},
    typeChangeImpact: () => ({ references: [], issues: [] }),
    onCommand,
  })
  app.component('UIcon', defineComponent({ setup: () => () => h('i') }))
  app.component(
    'UFormField',
    defineComponent({
      setup(_, { slots }) {
        return () => h('label', slots.default?.())
      },
    }),
  )
  app.component(
    'UBadge',
    defineComponent({
      setup(_, { slots }) {
        return () => h('span', slots.default?.())
      },
    }),
  )
  app.component(
    'UButton',
    defineComponent({
      inheritAttrs: true,
      props: {
        label: { type: String, default: '' },
        disabled: { type: Boolean, default: false },
      },
      setup(props, { attrs }) {
        return () => h('button', { ...attrs, disabled: props.disabled }, props.label)
      },
    }),
  )
  app.component(
    'UInput',
    defineComponent({
      inheritAttrs: true,
      props: { modelValue: { type: String, default: '' } },
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
  app.component(
    'USelect',
    defineComponent({
      inheritAttrs: true,
      props: {
        modelValue: { type: [String, Number, Boolean], default: '' },
        items: { type: Array, default: () => [] },
        labelKey: { type: String, default: 'label' },
        valueKey: { type: String, default: 'value' },
      },
      emits: ['update:modelValue'],
      setup(props, { attrs, emit }) {
        return () =>
          h(
            'select',
            {
              ...attrs,
              value: props.modelValue,
              onChange: (event: Event) =>
                emit('update:modelValue', (event.target as HTMLSelectElement).value),
            },
            (props.items as Array<Record<string, unknown>>).map((item) =>
              h('option', { value: item[props.valueKey] as string }, String(item[props.labelKey])),
            ),
          )
      },
    }),
  )
  app.component(
    'UTextarea',
    defineComponent({
      props: { modelValue: { type: String, default: '' } },
      setup: (props) => () => h('textarea', { value: props.modelValue }),
    }),
  )
  app.use(createI18n({ legacy: false, locale: 'zh', messages: { zh } }))
  app.mount(root)
  return root
}
