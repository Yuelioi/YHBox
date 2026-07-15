<template>
  <aside class="flex h-full w-[340px] shrink-0 flex-col border-l border-default bg-default">
    <div class="flex items-center justify-between border-b border-default px-4 py-3">
      <div class="min-w-0">
        <h2 class="truncate text-sm font-semibold text-highlighted">
          {{ t('workflow31.inspector.title') }}
        </h2>
        <p class="truncate font-mono text-[10px] text-dimmed">
          {{ node?.id || t('workflow31.inspector.no_selection') }}
        </p>
      </div>
      <UButton
        v-if="node"
        icon="i-tabler-trash"
        color="error"
        variant="ghost"
        size="xs"
        :aria-label="t('workflow31.inspector.remove_node')"
        @click="emit('command', { kind: 'remove-node', nodeId: node.id })"
      />
    </div>

    <section class="space-y-3 border-b border-default p-4">
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-xs font-semibold text-highlighted">
            {{ t('workflow31.inspector.state_title') }}
          </h3>
          <p class="mt-1 text-[10px] text-dimmed">
            {{ t('workflow31.inspector.state_hint') }}
          </p>
        </div>
        <UBadge color="neutral" variant="soft" size="sm">{{ variables.length }}</UBadge>
      </div>
      <div class="grid grid-cols-[1fr_1fr_auto] gap-2">
        <UInput
          v-model="newVariableName"
          :placeholder="t('workflow31.inspector.state_name_placeholder')"
          size="sm"
        />
        <USelect
          v-model="newVariableTypeId"
          :items="stateTypeItems"
          value-key="value"
          label-key="label"
          size="sm"
        />
        <UButton
          icon="i-tabler-plus"
          size="sm"
          color="neutral"
          :disabled="!canAddVariable"
          :aria-label="t('workflow31.inspector.state_add')"
          @click="addStateVariable"
        />
      </div>
      <div v-if="variables.length" class="space-y-1.5">
        <div
          v-for="variable in variables"
          :key="variable.name"
          class="flex items-center gap-2 rounded-md bg-elevated/55 px-2.5 py-2"
        >
          <span class="min-w-0 flex-1 truncate font-mono text-[11px] text-toned">{{
            variable.name
          }}</span>
          <span class="max-w-28 truncate text-[10px] text-dimmed">{{
            variableTypeLabel(variable)
          }}</span>
          <UButton
            icon="i-tabler-trash"
            color="error"
            variant="ghost"
            size="xs"
            :aria-label="t('workflow31.inspector.state_remove', { name: variable.name })"
            @click="emit('command', { kind: 'remove-state-variable', name: variable.name })"
          />
        </div>
      </div>
    </section>

    <div
      v-if="!node || !projection"
      class="flex flex-1 items-center justify-center px-8 text-center"
    >
      <div>
        <UIcon name="i-tabler-pointer" class="mx-auto mb-3 size-6 text-dimmed" />
        <p class="text-xs text-muted">{{ t('workflow31.inspector.select_hint') }}</p>
      </div>
    </div>

    <div v-else class="flex-1 space-y-6 overflow-y-auto p-4">
      <section class="space-y-3">
        <label class="block text-xs font-medium text-toned" for="workflow-node-label">
          {{ t('workflow31.inspector.label') }}
        </label>
        <UInput
          id="workflow-node-label"
          :model-value="node.label || ''"
          :placeholder="t('workflow31.inspector.label_placeholder')"
          class="w-full"
          @change="setLabel"
        />
      </section>

      <section v-if="projection.configFields.length" class="space-y-3">
        <h3 class="text-xs font-semibold text-highlighted">
          {{ t('workflow31.inspector.configuration') }}
        </h3>
        <GeneratedFieldEditor
          v-for="field in projection.configFields"
          :key="field.id"
          :field="field"
          :model-value="node.config[field.id]"
          :state-variables="variables.map((variable) => variable.name)"
          @update:model-value="
            emit('command', {
              kind: 'set-config',
              nodeId: node.id,
              fieldId: field.id,
              value: $event,
            })
          "
        />
      </section>

      <section v-if="projection.dataInputs.length" class="space-y-3">
        <h3 class="text-xs font-semibold text-highlighted">
          {{ t('workflow31.inspector.inputs') }}
        </h3>
        <div
          v-for="port in projection.dataInputs"
          :key="port.id"
          class="space-y-2 rounded-lg bg-elevated/55 p-3"
        >
          <div class="flex items-center gap-2">
            <span
              class="size-2 rounded-full"
              :style="{ backgroundColor: port.type.color || '#a1a1aa' }"
              aria-hidden="true"
            />
            <span class="text-xs font-medium text-toned">{{ port.id }}</span>
            <span class="ml-auto font-mono text-[10px] text-dimmed">
              {{ typeLabel(port) }} · {{ port.binding }}
            </span>
          </div>
          <PointValueEditor
            v-if="acceptsInline(port) && port.type.editorAdapter === 'point'"
            :model-value="literalValue(node.bindings[port.id], port.default)"
            @update:model-value="setLiteral(port.id, $event)"
          />
          <USwitch
            v-else-if="acceptsInline(port) && port.type.control === 'toggle'"
            :model-value="literalBoolean(node.bindings[port.id], port.default)"
            @update:model-value="setLiteral(port.id, $event)"
          />
          <UInputNumber
            v-else-if="
              acceptsInline(port) &&
              (port.type.control === 'number' || port.type.control === 'integer')
            "
            :model-value="literalNumber(node.bindings[port.id], port.default)"
            :min="numericConstraint(port.type.constraints.minimum)"
            :max="numericConstraint(port.type.constraints.maximum)"
            :step="port.type.control === 'integer' ? 1 : 'any'"
            class="w-full"
            @update:model-value="setLiteral(port.id, Number($event))"
          />
          <USelect
            v-else-if="acceptsInline(port) && port.type.control === 'select'"
            :model-value="literalValue(node.bindings[port.id], port.default)"
            :items="port.type.constraints.enum.map((value) => ({ label: String(value), value }))"
            class="w-full"
            @update:model-value="setLiteral(port.id, $event)"
          />
          <UInput
            v-else-if="acceptsInline(port) && port.type.control === 'text'"
            :model-value="literalText(node.bindings[port.id])"
            :placeholder="literalPlaceholder(port)"
            class="w-full"
            @change="setLiteralText(port.id, $event)"
          />
          <UTextarea
            v-else-if="acceptsInline(port)"
            :model-value="literalJSON(node.bindings[port.id], port.default)"
            :placeholder="literalPlaceholder(port)"
            class="w-full font-mono text-xs"
            @change="setLiteralJSON(port.id, $event)"
          />
          <p v-else class="text-[11px] leading-5 text-muted">
            {{ t('workflow31.inspector.reference_only', { carrier: port.carrier }) }}
          </p>
          <div class="flex items-center gap-2">
            <UButton
              v-if="port.hasDefault"
              :label="t('workflow31.inspector.use_default')"
              size="xs"
              color="neutral"
              variant="soft"
              @click="emit('command', { kind: 'bind-default', nodeId: node.id, portId: port.id })"
            />
            <UButton
              v-if="node.bindings[port.id]"
              :label="t('workflow31.inspector.clear')"
              size="xs"
              color="neutral"
              variant="ghost"
              @click="emit('command', { kind: 'clear-binding', nodeId: node.id, portId: port.id })"
            />
          </div>
        </div>
      </section>

      <section v-if="projection.capabilities.length" class="space-y-3">
        <h3 class="text-xs font-semibold text-highlighted">
          {{ t('workflow31.inspector.capabilities') }}
        </h3>
        <div
          v-for="capability in projection.capabilities"
          :key="capability.requirementId"
          class="rounded-lg border border-default px-3 py-2.5"
        >
          <p class="text-xs font-medium text-toned">{{ capability.requirementId }}</p>
          <p class="mt-1 text-[11px] text-muted">
            {{ capability.operations.join(', ') }}
          </p>
          <p class="mt-1 font-mono text-[10px] text-dimmed">
            {{ capability.risk }} / {{ capability.consent }}
          </p>
        </div>
      </section>

      <section v-if="projection.statusEvents.length" class="space-y-2">
        <h3 class="text-xs font-semibold text-highlighted">
          {{ t('workflow31.inspector.observed_status') }}
        </h3>
        <p class="text-[11px] leading-5 text-muted">
          {{ t('workflow31.inspector.status_hint') }}
        </p>
        <code class="block text-[10px] text-toned">{{
          projection.statusEvents.map((event) => event.code).join('\n')
        }}</code>
      </section>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { InputBinding, Variable } from '../../../../contracts/workflow/3.1/workflow-source'
import type { PortProjection } from '../../../../contracts/node/3.1/authoring-projection'
import type { TypeProjection } from '../../../../contracts/node/3.1/authoring-projection'
import type { EditorCommand, Node, NodeProjection } from '@/app/editor/EditorSession'
import GeneratedFieldEditor from '@/app/editor/GeneratedFieldEditor.vue'
import PointValueEditor from '@/app/editor/PointValueEditor.vue'

const props = defineProps<{
  node: Node | null
  projection: NodeProjection | null
  variables: Variable[]
  types: TypeProjection[]
}>()
const emit = defineEmits<{ command: [command: EditorCommand] }>()
const { t, te } = useI18n()
const newVariableName = ref('')
const newVariableTypeId = ref('')
const stateTypes = computed(() =>
  props.types.filter((type) =>
    type.representations.some((representation) => representation.kind === 'inline-json'),
  ),
)
const stateTypeItems = computed(() =>
  stateTypes.value.map((type) => ({
    label:
      type.titleKey && te(type.titleKey)
        ? t(type.titleKey)
        : type.typeRef.typeId.split('/').at(-2)!,
    value: type.typeRef.typeId,
  })),
)
const canAddVariable = computed(
  () =>
    /^[A-Za-z0-9_][A-Za-z0-9._-]*$/.test(newVariableName.value) && Boolean(selectedStateType.value),
)
const selectedStateType = computed(() =>
  stateTypes.value.find((type) => type.typeRef.typeId === newVariableTypeId.value),
)

watch(
  stateTypes,
  (values) => {
    if (!values.some((type) => type.typeRef.typeId === newVariableTypeId.value))
      newVariableTypeId.value = values[0]?.typeRef.typeId ?? ''
  },
  { immediate: true },
)

function addStateVariable(): void {
  const type = selectedStateType.value
  if (!type || !canAddVariable.value) return
  emit('command', {
    kind: 'add-state-variable',
    name: newVariableName.value,
    type: { kind: 'ref', ref: { ...type.typeRef } },
    defaultValue: defaultStateValue(type),
  })
  newVariableName.value = ''
}

function defaultStateValue(type: TypeProjection): unknown {
  if (type.examples.length) return structuredClone(type.examples[0])
  switch (type.control) {
    case 'text':
      return ''
    case 'number':
    case 'integer':
      return 0
    case 'toggle':
      return false
    case 'select':
      return type.constraints.enum[0] ?? null
    case 'list':
      return []
    case 'object':
      return {}
    default:
      return null
  }
}

function variableTypeLabel(variable: Variable): string {
  if (variable.type.kind !== 'ref') return variable.type.kind
  const typeId = variable.type.ref.typeId
  const type = props.types.find((candidate) => candidate.typeRef.typeId === typeId)
  if (type?.titleKey && te(type.titleKey)) return t(type.titleKey)
  return typeId.split('/').at(-2) ?? typeId
}

function setLabel(event: Event): void {
  if (!props.node) return
  emit('command', {
    kind: 'set-node-label',
    nodeId: props.node.id,
    label: (event.target as HTMLInputElement).value,
  })
}

function setLiteral(portId: string, value: unknown): void {
  if (!props.node) return
  emit('command', {
    kind: 'bind-value',
    nodeId: props.node.id,
    portId,
    value,
  })
}

function setLiteralText(portId: string, event: Event): void {
  setLiteral(portId, (event.target as HTMLInputElement).value)
}

function setLiteralJSON(portId: string, event: Event): void {
  const raw = (event.target as HTMLTextAreaElement).value
  try {
    setLiteral(portId, JSON.parse(raw))
  } catch {
    return
  }
}

function acceptsInline(port: PortProjection): boolean {
  return port.type.representations.some((representation) => representation.kind === 'inline-json')
}

function literalText(binding: InputBinding | undefined): string {
  return binding?.kind === 'value' && typeof binding.value === 'string' ? binding.value : ''
}

function literalValue(binding: InputBinding | undefined, defaultValue: unknown): unknown {
  return binding?.kind === 'value' ? binding.value : defaultValue
}

function literalBoolean(binding: InputBinding | undefined, defaultValue: unknown): boolean {
  const value = literalValue(binding, defaultValue)
  return typeof value === 'boolean' ? value : false
}

function literalNumber(
  binding: InputBinding | undefined,
  defaultValue: unknown,
): number | undefined {
  const value = literalValue(binding, defaultValue)
  return typeof value === 'number' ? value : undefined
}

function literalJSON(binding: InputBinding | undefined, defaultValue: unknown): string {
  const value = literalValue(binding, defaultValue)
  return value === undefined ? '' : JSON.stringify(value, null, 2)
}

function numericConstraint(value: unknown): number | undefined {
  return typeof value === 'number' ? value : undefined
}

function typeLabel(port: PortProjection): string {
  if (port.type.titleKey && te(port.type.titleKey)) return t(port.type.titleKey)
  return port.type.typeIds.join(' | ') || port.type.label
}

function literalPlaceholder(port: PortProjection): string {
  if (port.hasDefault && typeof port.default === 'string') return port.default
  return t(
    port.binding === 'required'
      ? 'workflow31.inspector.required_value'
      : 'workflow31.inspector.optional_value',
  )
}
</script>
