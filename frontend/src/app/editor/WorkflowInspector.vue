<template>
  <aside class="flex h-full w-[340px] shrink-0 flex-col border-l border-default bg-default">
    <div class="flex items-center justify-between border-b border-default px-4 py-3">
      <div class="min-w-0">
        <h2 class="truncate text-sm font-semibold text-highlighted">
          {{ t('workflow.inspector.title') }}
        </h2>
        <p class="truncate font-mono text-[10px] text-dimmed">
          {{ node?.id || t('workflow.inspector.no_selection') }}
        </p>
      </div>
      <UButton
        v-if="node"
        icon="i-tabler-trash"
        color="error"
        variant="ghost"
        size="xs"
        :aria-label="t('workflow.inspector.remove_node')"
        @click="emit('command', { kind: 'remove-node', nodeId: node.id })"
      />
    </div>

    <section class="space-y-3 border-b border-default p-4">
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-xs font-semibold text-highlighted">
            {{ t('workflow.inspector.state_title') }}
          </h3>
          <p class="mt-1 text-[10px] text-dimmed">
            {{ t('workflow.inspector.state_hint') }}
          </p>
        </div>
        <UBadge color="neutral" variant="soft" size="sm">{{ variables.length }}</UBadge>
      </div>
      <div class="grid grid-cols-[1fr_1fr_auto] gap-2">
        <UInput
          v-model="newVariableName"
          :placeholder="t('workflow.inspector.state_name_placeholder')"
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
          :aria-label="t('workflow.inspector.state_add')"
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
            :aria-label="t('workflow.inspector.state_remove', { name: variable.name })"
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
        <p class="text-xs text-muted">{{ t('workflow.inspector.select_hint') }}</p>
      </div>
    </div>

    <div v-else class="flex-1 space-y-6 overflow-y-auto p-4">
      <section class="space-y-3">
        <label class="block text-xs font-medium text-toned" for="workflow-node-label">
          {{ t('workflow.inspector.label') }}
        </label>
        <UInput
          id="workflow-node-label"
          :model-value="node.label || ''"
          :placeholder="t('workflow.inspector.label_placeholder')"
          class="w-full"
          @change="setLabel"
        />
      </section>

      <section v-if="projection.configFields.length" class="space-y-3">
        <h3 class="text-xs font-semibold text-highlighted">
          {{ t('workflow.inspector.configuration') }}
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
          {{ t('workflow.inspector.inputs') }}
        </h3>
        <WorkflowInputBindingEditor
          v-for="port in projection.dataInputs"
          :key="port.id"
          :node="node"
          :port="port"
          :clips="clips"
          :clip-items="clipItems"
          :template-variant-items="templateVariantItems"
          @command="emit('command', $event)"
        />
      </section>

      <section v-if="projection.capabilities.length" class="space-y-3">
        <h3 class="text-xs font-semibold text-highlighted">
          {{ t('workflow.inspector.capabilities') }}
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
          {{ t('workflow.inspector.observed_status') }}
        </h3>
        <p class="text-[11px] leading-5 text-muted">
          {{ t('workflow.inspector.status_hint') }}
        </p>
        <code class="block text-[10px] text-toned">{{
          projection.statusEvents.map((event) => event.code).join('\n')
        }}</code>
      </section>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import type { Variable } from '../../../../contracts/workflow/3.1/workflow-source'
import type { TypeProjection } from '../../../../contracts/node/3.1/authoring-projection'
import type { EditorCommand, Node, NodeProjection } from '@/app/editor/EditorSession'
import GeneratedFieldEditor from '@/app/editor/GeneratedFieldEditor.vue'
import WorkflowInputBindingEditor from '@/app/editor/WorkflowInputBindingEditor.vue'
import { useClipsStore } from '@/stores/clips'
import { useTemplatesStore } from '@/stores/templates'

const props = defineProps<{
  node: Node | null
  projection: NodeProjection | null
  variables: Variable[]
  types: TypeProjection[]
}>()
const emit = defineEmits<{ command: [command: EditorCommand] }>()
const { t, te } = useI18n()
const clipsStore = useClipsStore()
const { clips } = storeToRefs(clipsStore)
const templatesStore = useTemplatesStore()
const { map: templates } = storeToRefs(templatesStore)
const clipItems = computed(() =>
  clips.value.map((clip) => ({
    label: clip.label || clip.id,
    value: clip.id,
  })),
)
const templateVariantItems = computed(() =>
  Object.values(templates.value).flatMap((asset) =>
    asset.variants.map((variant, index) => ({
      label: `${asset.name} · ${variant.resolution[0]}×${variant.resolution[1]}`,
      value: `${asset.guid}:${index}`,
      blob: variant.blob,
    })),
  ),
)
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

onMounted(() => {
  void clipsStore.refresh()
  void templatesStore.reload()
})

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
</script>
