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
    <USelect
      v-else-if="port.editorAdapter === 'template-image'"
      :model-value="selectedTemplateVariantId"
      :items="templateVariantItems"
      value-key="value"
      label-key="label"
      :placeholder="t('workflow.inspector.select_template')"
      class="w-full"
      @update:model-value="setTemplateImage"
    />
    <div
      v-if="port.editorAdapter === 'template-image' && binding?.kind === 'blob'"
      class="flex items-center gap-3 rounded-lg border border-default bg-default p-2"
    >
      <BlobPreview
        v-if="selectedTemplateVariant"
        :blob="selectedTemplateVariant.blob"
        :alt="selectedTemplateVariant.label"
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
          {{ selectedTemplateVariant?.label || t('workflow.inspector.resource_missing') }}
        </p>
        <p class="truncate font-mono text-[10px] text-dimmed">{{ binding.blob?.digest }}</p>
      </div>
      <UBadge
        v-if="!selectedTemplateVariant || templatePreviewState === 'unavailable'"
        color="error"
        variant="soft"
        size="sm"
      >
        {{ t('workflow.inspector.resource_stale') }}
      </UBadge>
    </div>
    <USelect
      v-else-if="isInputClip"
      :model-value="selectedClipId"
      :items="clipItems"
      value-key="value"
      label-key="label"
      :placeholder="t('workflow.inspector.select_clip')"
      class="w-full"
      @update:model-value="setClip"
    />
    <div
      v-if="isInputClip && binding?.kind === 'blob'"
      class="flex items-center gap-2 rounded-lg border border-default bg-default px-3 py-2"
    >
      <UIcon name="i-tabler-movie" class="size-4 shrink-0 text-primary" />
      <span class="min-w-0 flex-1 truncate text-xs text-toned">
        {{ selectedClip?.label || t('workflow.inspector.resource_missing') }}
      </span>
      <UBadge v-if="!selectedClip" color="error" variant="soft" size="sm">
        {{ t('workflow.inspector.resource_stale') }}
      </UBadge>
    </div>
    <p v-else class="text-[11px] leading-5 text-muted">
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
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { InputBinding } from '../../../../contracts/workflow/3.1/workflow-source'
import type { PortProjection } from '../../../../contracts/node/3.1/authoring-projection'
import type { EditorCommand, Node } from '@/app/editor/EditorSession'
import PointValueEditor from '@/app/editor/PointValueEditor.vue'
import ColorRangeValueEditor from '@/app/editor/ColorRangeValueEditor.vue'
import BlobPreview from '@/components/common/BlobPreview.vue'

type Blob = { mediaType: string; digest: string; size: number }
type Clip = { id: string; label: string; blob: Blob }
type Item = { label: string; value: string }
type TemplateItem = Item & { blob: Blob }

const inputClipTypeId = 'https://schemas.yotta.dev/types/automation/input-clip/v1'
const props = defineProps<{
  node: Node
  port: PortProjection
  clips: Clip[]
  clipItems: Item[]
  templateVariantItems: TemplateItem[]
}>()
const emit = defineEmits<{ command: [command: EditorCommand] }>()
const { t, te } = useI18n()
const templatePreviewState = ref<'loading' | 'ready' | 'unavailable'>('loading')

const binding = computed(() => props.node.bindings[props.port.id])
const acceptsInline = computed(() =>
  props.port.type.representations.some((item) => item.kind === 'inline-json'),
)
const isInputClip = computed(() => props.port.type.typeIds.includes(inputClipTypeId))
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
const selectedTemplateVariant = computed(() =>
  matchingBlob(binding.value, props.templateVariantItems),
)
const selectedTemplateVariantId = computed(() => selectedTemplateVariant.value?.value)
const selectedClip = computed(() =>
  matchingBlob(
    binding.value,
    props.clips.map((clip) => ({ ...clip, value: clip.id })),
  ),
)
const selectedClipId = computed(() => selectedClip.value?.value)

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
function setClip(value: unknown): void {
  const clip = typeof value === 'string' ? props.clips.find((item) => item.id === value) : undefined
  if (clip)
    emit('command', {
      kind: 'bind-blob',
      nodeId: props.node.id,
      portId: props.port.id,
      blob: { ...clip.blob },
    })
}
function setTemplateImage(value: unknown): void {
  const variant =
    typeof value === 'string'
      ? props.templateVariantItems.find((item) => item.value === value)
      : undefined
  if (variant)
    emit('command', {
      kind: 'bind-blob',
      nodeId: props.node.id,
      portId: props.port.id,
      blob: { ...variant.blob },
    })
}
function matchingBlob<T extends { blob: Blob }>(
  value: InputBinding | undefined,
  items: T[],
): T | undefined {
  if (value?.kind !== 'blob' || !value.blob) return undefined
  return items.find(
    (item) =>
      item.blob.digest === value.blob?.digest &&
      item.blob.mediaType === value.blob.mediaType &&
      item.blob.size === value.blob.size,
  )
}
</script>
