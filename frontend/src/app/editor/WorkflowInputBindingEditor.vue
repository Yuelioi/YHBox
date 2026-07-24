<template>
  <div class="space-y-2 rounded-lg bg-elevated/55 p-3">
    <div class="flex items-center gap-2">
      <span
        class="size-2 rounded-full"
        :style="{ backgroundColor: port.type.color || '#a1a1aa' }"
        aria-hidden="true"
      />
      <span class="text-xs font-medium text-toned">{{ portTitle }}</span>
      <span class="ml-auto font-mono text-[10px] text-dimmed"
        >{{ typeLabel }} · {{ port.binding }}</span
      >
    </div>
    <p v-if="portDescription" class="text-[11px] leading-5 text-muted">{{ portDescription }}</p>
    <p
      v-if="missingRequired"
      class="rounded-md border border-warning/30 bg-warning/10 px-2.5 py-2 text-[11px] text-warning"
    >
      {{ t('workflow.inspector.required_value') }}
    </p>
    <WorkflowValueEditor
      v-if="acceptsInline"
      :adapter="editorAdapter"
      :port="port"
      :model-value="literalValue"
      :target-slot="targetSlot"
      @update:model-value="setLiteral"
    />
    <AssetReferenceField
      v-else-if="usesAssetPicker"
      :kind="assetKind ?? 'clip'"
      :bound="binding?.kind === 'blob'"
      :blob="bindingBlob"
      :label="resourceLabel"
      :placeholder="pickerPlaceholder"
      :stale="resourceStale"
      :clearable="bindingActions.clear"
      @change="pickerOpen = true"
      @clear="emit('command', { kind: 'clear-binding', nodeId: node.id, portId: port.id })"
    />
    <p v-if="needsPickerTarget" class="text-[11px] leading-5 text-warning">
      {{ t('workflow.inspector.picker_target_required') }}
    </p>
    <p v-if="!acceptsInline && !usesAssetPicker" class="text-[11px] leading-5 text-muted">
      {{ t('workflow.inspector.reference_only', { carrier: port.carrier }) }}
    </p>
    <div class="flex items-center gap-2">
      <UButton
        v-if="bindingActions.resetToDefault"
        :label="t('workflow.inspector.use_default')"
        size="xs"
        color="neutral"
        variant="soft"
        @click="emit('command', { kind: 'bind-default', nodeId: node.id, portId: port.id })"
      />
      <UButton
        v-if="bindingActions.clear && !usesAssetPicker"
        :label="t('workflow.inspector.clear')"
        size="xs"
        color="neutral"
        variant="ghost"
        @click="emit('command', { kind: 'clear-binding', nodeId: node.id, portId: port.id })"
      />
    </div>
    <AssetPickerModal
      v-if="assetKind"
      v-model:open="pickerOpen"
      :kind="assetKind"
      :selected-blob="bindingBlob"
      @select="setAsset"
      @capture="emit('capture-template')"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PortProjection } from '../../../../contracts/node/current/authoring-projection'
import type { EditorCommand, Node } from '@/app/editor/EditorSession'
import { resolvePortAdapter } from '@/app/editor/authoringSurface'
import { bindingActionPolicy } from '@/app/editor/bindingActionPolicy'
import AssetReferenceField from '@/app/editor/AssetReferenceField.vue'
import AssetPickerModal from '@/components/assets/AssetPickerModal.vue'
import { useAssetsStore, type AssetPickerSelection } from '@/stores/assets'
import type { AssetBinding } from '@/lib/backend'

const WorkflowValueEditor = defineAsyncComponent(
  () => import('@/app/editor/WorkflowValueEditor.vue'),
)

const inputClipTypeId = 'https://schemas.yotta.dev/types/automation/input-clip/v1'
const macroTypeId = 'https://schemas.yotta.dev/types/automation/macro/v1'
const props = defineProps<{
  node: Node
  port: PortProjection
  title?: string
  targetSlot?: string
  connected?: boolean
}>()
const emit = defineEmits<{
  command: [command: EditorCommand]
  'capture-template': []
}>()
const { t, te } = useI18n()
const assets = useAssetsStore()
const pickerOpen = ref(false)
const resolvedBinding = ref<AssetBinding | null>(null)
const bindingResolved = ref(false)
const immediateSelection = ref<AssetPickerSelection | null>(null)
let resolveGeneration = 0

const binding = computed(() => props.node.bindings[props.port.id])
const bindingActions = computed(() =>
  bindingActionPolicy({
    required: props.port.binding === 'required',
    hasDefault: props.port.hasDefault,
    bound: Boolean(binding.value),
  }),
)
const acceptsInline = computed(() =>
  props.port.type.representations.some((item) => item.kind === 'inline-json'),
)
const editorAdapter = computed(() => resolvePortAdapter(props.port))
const isInputClip = computed(() => props.port.type.typeIds.includes(inputClipTypeId))
const isMacro = computed(() => props.port.type.typeIds.includes(macroTypeId))
const assetKind = computed<'template' | 'macro' | 'clip' | null>(() => {
  if (props.port.editorAdapter === 'template-image') return 'template'
  if (isMacro.value) return 'macro'
  if (isInputClip.value) return 'clip'
  return null
})
const usesAssetPicker = computed(() => assetKind.value !== null)
const missingRequired = computed(
  () =>
    props.port.binding === 'required' &&
    !props.connected &&
    !binding.value &&
    !props.port.hasDefault,
)
const needsPickerTarget = computed(
  () => !props.targetSlot && ['point', 'region', 'color-range'].includes(editorAdapter.value),
)
const bindingBlob = computed(() =>
  binding.value?.kind === 'blob' ? binding.value.blob : undefined,
)
const pickerPlaceholder = computed(() =>
  t(
    assetKind.value === 'template'
      ? 'workflow.inspector.select_template'
      : assetKind.value === 'macro'
        ? 'workflow.inspector.select_macro'
        : 'workflow.inspector.select_clip',
  ),
)
const resourceLabel = computed(() => {
  const selection = immediateSelection.value
  if (selection && sameBlob(selection.blob, bindingBlob.value)) return selectionLabel(selection)
  const resolved = resolvedBinding.value
  if (!resolved?.found) return t('workflow.inspector.resource_missing')
  if (resolved.kind === 'template' && resolved.resolution[0] > 0) {
    return `${resolved.name} · ${resolved.resolution[0]}×${resolved.resolution[1]}`
  }
  return resolved.name
})
const resourceStale = computed(() =>
  Boolean(bindingBlob.value && bindingResolved.value && !resolvedBinding.value?.found),
)
const portTitle = computed(
  () =>
    props.title?.trim() ||
    (props.port.titleKey && te(props.port.titleKey) ? t(props.port.titleKey) : props.port.id),
)
const portDescription = computed(() =>
  props.port.descriptionKey && te(props.port.descriptionKey) ? t(props.port.descriptionKey) : '',
)
const typeLabel = computed(() =>
  props.port.type.titleKey && te(props.port.type.titleKey)
    ? t(props.port.type.titleKey)
    : props.port.type.typeIds.join(' | ') || props.port.type.label,
)
const literalValue = computed(() =>
  binding.value?.kind === 'value' ? binding.value.value : props.port.default,
)
watch(
  () =>
    [
      bindingBlob.value?.mediaType,
      bindingBlob.value?.digest,
      bindingBlob.value?.size,
      assets.epoch,
    ] as const,
  () => void resolveCurrentBinding(),
  { immediate: true },
)

function setLiteral(value: unknown): void {
  emit('command', { kind: 'bind-value', nodeId: props.node.id, portId: props.port.id, value })
}
function setAsset(selection: AssetPickerSelection): void {
  immediateSelection.value = selection
  emit('command', {
    kind: 'bind-blob',
    nodeId: props.node.id,
    portId: props.port.id,
    blob: { ...selection.blob },
  })
}

async function resolveCurrentBinding(): Promise<void> {
  const generation = ++resolveGeneration
  const blob = bindingBlob.value
  resolvedBinding.value = null
  bindingResolved.value = !blob
  if (!blob) {
    immediateSelection.value = null
    return
  }
  try {
    const resolved = await assets.resolveBinding(blob)
    if (generation !== resolveGeneration) return
    resolvedBinding.value = resolved
    bindingResolved.value = true
  } catch {
    if (generation === resolveGeneration) bindingResolved.value = false
  }
}

function sameBlob(
  left: { mediaType: string; digest: string; size: number } | undefined,
  right: { mediaType: string; digest: string; size: number } | undefined,
): boolean {
  return Boolean(
    left &&
    right &&
    left.mediaType === right.mediaType &&
    left.digest === right.digest &&
    left.size === right.size,
  )
}

function selectionLabel(selection: AssetPickerSelection): string {
  return selection.resolution
    ? `${selection.name} · ${selection.resolution[0]}×${selection.resolution[1]}`
    : selection.name
}
</script>
