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
    <PointValueEditor
      v-if="acceptsInline && port.type.editorAdapter === 'point'"
      :model-value="literalValue"
      @update:model-value="setLiteral"
    />
    <ColorRangeValueEditor
      v-else-if="acceptsInline && port.type.editorAdapter === 'color-range'"
      :model-value="literalValue"
      @update:model-value="setLiteral"
    />
    <KeyChordValueEditor
      v-else-if="acceptsInline && isKeyChordType(port.type.expression)"
      :model-value="literalKeyChord"
      @update:model-value="setLiteral"
    />
    <USwitch
      v-else-if="acceptsInline && port.type.control === 'toggle'"
      :model-value="literalBoolean"
      @update:model-value="setLiteral"
    />
    <UInputNumber
      v-else-if="
        acceptsInline && (port.type.control === 'number' || port.type.control === 'integer')
      "
      :model-value="literalNumber"
      :min="numericConstraint(port.type.constraints.minimum)"
      :max="numericConstraint(port.type.constraints.maximum)"
      :step="port.type.control === 'integer' ? 1 : 'any'"
      class="w-full"
      @update:model-value="setLiteral(Number($event))"
    />
    <USelect
      v-else-if="acceptsInline && port.type.control === 'select'"
      :model-value="literalValue"
      :items="port.type.constraints.enum.map((value) => ({ label: String(value), value }))"
      class="w-full"
      @update:model-value="setLiteral"
    />
    <UInput
      v-else-if="acceptsInline && port.type.control === 'text'"
      :model-value="literalText"
      :placeholder="literalPlaceholder"
      class="w-full"
      @change="setLiteralText"
    />
    <UTextarea
      v-else-if="acceptsInline"
      :model-value="literalJSON"
      :placeholder="literalPlaceholder"
      class="w-full font-mono text-xs"
      @change="setLiteralJSON"
    />
    <UButton
      v-else-if="usesAssetPicker"
      class="w-full justify-start"
      color="neutral"
      variant="outline"
      :icon="assetKind === 'template' ? 'i-tabler-photo-search' : 'i-tabler-movie'"
      :label="bindingBlob ? t('assetPicker.replace') : pickerPlaceholder"
      trailing-icon="i-tabler-chevron-right"
      @click="pickerOpen = true"
    />
    <div
      v-if="port.editorAdapter === 'template-image' && binding?.kind === 'blob'"
      class="flex items-center gap-3 rounded-lg border border-default bg-default p-2"
    >
      <BlobPreview
        v-if="binding.blob"
        :blob="binding.blob"
        :alt="resourceLabel"
        class="size-14 shrink-0"
        @state="templatePreviewState = $event"
      />
      <div
        v-else
        class="flex size-14 shrink-0 items-center justify-center rounded-lg bg-error/10 text-error"
      >
        <UIcon name="i-tabler-photo-off" class="size-5" />
      </div>
      <div class="min-w-0 flex-1">
        <p class="truncate text-xs font-medium text-toned">
          {{ resourceLabel }}
        </p>
        <p class="truncate font-mono text-[10px] text-dimmed">{{ binding.blob?.digest }}</p>
      </div>
      <UBadge
        v-if="resourceStale || templatePreviewState === 'unavailable'"
        color="error"
        variant="soft"
        size="sm"
      >
        {{ t('workflow.inspector.resource_stale') }}
      </UBadge>
    </div>
    <div
      v-if="isInputClip && binding?.kind === 'blob'"
      class="flex items-center gap-2 rounded-lg border border-default bg-default px-3 py-2"
    >
      <UIcon name="i-tabler-movie" class="size-4 shrink-0 text-primary" />
      <span class="min-w-0 flex-1 truncate text-xs text-toned">
        {{ resourceLabel }}
      </span>
      <UBadge v-if="resourceStale" color="error" variant="soft" size="sm">
        {{ t('workflow.inspector.resource_stale') }}
      </UBadge>
    </div>
    <p v-if="!usesAssetPicker" class="text-[11px] leading-5 text-muted">
      {{ t('workflow.inspector.reference_only', { carrier: port.carrier }) }}
    </p>
    <div class="flex items-center gap-2">
      <UButton
        v-if="port.hasDefault"
        :label="t('workflow.inspector.use_default')"
        size="xs"
        color="neutral"
        variant="soft"
        @click="emit('command', { kind: 'bind-default', nodeId: node.id, portId: port.id })"
      />
      <UButton
        v-if="binding"
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
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PortProjection } from '../../../../contracts/node/3.1/authoring-projection'
import type { EditorCommand, Node } from '@/app/editor/EditorSession'
import PointValueEditor from '@/app/editor/PointValueEditor.vue'
import ColorRangeValueEditor from '@/app/editor/ColorRangeValueEditor.vue'
import KeyChordValueEditor from '@/app/editor/KeyChordValueEditor.vue'
import { isKeyChordType } from '@/app/editor/keyChord'
import BlobPreview from '@/components/common/BlobPreview.vue'
import AssetPickerModal from '@/components/assets/AssetPickerModal.vue'
import { useAssetsStore, type AssetPickerSelection } from '@/stores/assets'
import type { AssetBinding } from '@/lib/backend'

const inputClipTypeId = 'https://schemas.yotta.dev/types/automation/input-clip/v1'
const props = defineProps<{
  node: Node
  port: PortProjection
}>()
const emit = defineEmits<{ command: [command: EditorCommand] }>()
const { t, te } = useI18n()
const assets = useAssetsStore()
const templatePreviewState = ref<'loading' | 'ready' | 'unavailable'>('loading')
const pickerOpen = ref(false)
const resolvedBinding = ref<AssetBinding | null>(null)
const bindingResolved = ref(false)
const immediateSelection = ref<AssetPickerSelection | null>(null)
let resolveGeneration = 0

const binding = computed(() => props.node.bindings[props.port.id])
const acceptsInline = computed(() =>
  props.port.type.representations.some((item) => item.kind === 'inline-json'),
)
const isInputClip = computed(() => props.port.type.typeIds.includes(inputClipTypeId))
const assetKind = computed<'template' | 'clip' | null>(() => {
  if (props.port.editorAdapter === 'template-image') return 'template'
  if (isInputClip.value) return 'clip'
  return null
})
const usesAssetPicker = computed(() => assetKind.value !== null)
const bindingBlob = computed(() =>
  binding.value?.kind === 'blob' ? binding.value.blob : undefined,
)
const pickerPlaceholder = computed(() =>
  t(
    assetKind.value === 'template'
      ? 'workflow.inspector.select_template'
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
const portTitle = computed(() =>
  props.port.titleKey && te(props.port.titleKey) ? t(props.port.titleKey) : props.port.id,
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
const literalBoolean = computed(() =>
  typeof literalValue.value === 'boolean' ? literalValue.value : false,
)
const literalNumber = computed(() =>
  typeof literalValue.value === 'number' ? literalValue.value : undefined,
)
const literalText = computed(() =>
  binding.value?.kind === 'value' && typeof binding.value.value === 'string'
    ? binding.value.value
    : '',
)
const literalKeyChord = computed(() =>
  Array.isArray(literalValue.value)
    ? literalValue.value.filter((value): value is string => typeof value === 'string')
    : [],
)
const literalJSON = computed(() =>
  literalValue.value === undefined ? '' : JSON.stringify(literalValue.value, null, 2),
)
const literalPlaceholder = computed(() =>
  props.port.hasDefault && typeof props.port.default === 'string'
    ? props.port.default
    : t(
        props.port.binding === 'required'
          ? 'workflow.inspector.required_value'
          : 'workflow.inspector.optional_value',
      ),
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

function numericConstraint(value: unknown): number | undefined {
  return typeof value === 'number' ? value : undefined
}
function setLiteral(value: unknown): void {
  emit('command', { kind: 'bind-value', nodeId: props.node.id, portId: props.port.id, value })
}
function setLiteralText(event: Event): void {
  setLiteral((event.target as HTMLInputElement).value)
}
function setLiteralJSON(event: Event): void {
  try {
    setLiteral(JSON.parse((event.target as HTMLTextAreaElement).value))
  } catch {
    return
  }
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
