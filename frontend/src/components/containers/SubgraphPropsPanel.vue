<template>
  <div v-if="!subgraph" class="text-sm text-dimmed">{{ t('subgraphProps.no_selection') }}</div>
  <div v-else class="space-y-4">
    <header class="flex items-center gap-2 pb-3 border-b border-default">
      <UIcon name="i-tabler-package" class="size-4 text-fuchsia-300" />
      <h3 class="text-sm font-medium text-highlighted">{{ subgraph.label }}</h3>
      <UBadge size="xs" color="neutral" variant="soft">{{ t('subgraphProps.outputs_count', { n: subgraph.outputPins?.length ?? 0 }) }}</UBadge>
    </header>

    <section class="space-y-1.5">
      <label class="block text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed">ID</label>
      <button
        type="button"
        class="w-full text-left text-[11px] font-mono bg-elevated/40 rounded px-2 py-1 hover:bg-elevated/60 transition-colors truncate flex items-center gap-1.5"
        :class="copied ? 'text-success' : 'text-dimmed'"
        :title="t('subgraphProps.click_to_copy') + subgraph.id"
        @click="onCopyID"
      >
        <UIcon v-if="copied" name="i-tabler-check" class="size-3 shrink-0" />
        <span class="truncate">{{ copied ? t('common.copied') : subgraph.id }}</span>
      </button>
    </section>

    <section class="space-y-2">
      <label class="text-xs text-toned">{{ t('subgraphProps.name') }}</label>
      <UInput
        :model-value="subgraph.label"
        size="sm"
        @update:model-value="(v: string) => $emit('update', { label: v })"
      />
    </section>

    <section class="space-y-2">
      <label class="text-xs text-toned">{{ t('subgraphProps.description') }}</label>
      <UTextarea
        :model-value="subgraph.description ?? ''"
        :rows="2"
        size="sm"
        @update:model-value="(v: string) => $emit('update', { description: v })"
      />
    </section>

    <!-- 录制元数据 -->
    <section v-if="subgraph.recordingContext" class="space-y-2 rounded-md bg-elevated/30 border border-default/40 p-3">
      <div class="flex items-center gap-1.5">
        <UIcon name="i-tabler-clipboard-data" class="size-3.5 text-toned" />
        <span class="text-xs text-toned font-medium">{{ t('subgraphProps.recording_meta') }}</span>
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-refresh"
          :title="t('subgraphProps.reset_recording_tip')"
          class="ml-auto"
          @click="onResetRecording"
        />
      </div>
      <div class="space-y-2">
        <div class="space-y-1">
          <label class="text-[11px] text-dimmed">{{ t('subgraphProps.source_counts360') }}</label>
          <UInputNumber
            :model-value="subgraph.recordingContext.mouseCounts360"
            size="sm"
            :min="0"
            @update:model-value="onPatchRecording('mouseCounts360', $event)"
          />
        </div>
        <div class="space-y-1">
          <label class="text-[11px] text-dimmed">{{ t('subgraphProps.source_resolution') }}</label>
          <div class="flex items-center gap-1.5">
            <UInputNumber
              :model-value="subgraph.recordingContext.resolution?.[0] ?? 0"
              size="sm"
              :min="0"
              class="w-24"
              @update:model-value="onPatchResolution(0, $event)"
            />
            <span class="text-xs text-dimmed">×</span>
            <UInputNumber
              :model-value="subgraph.recordingContext.resolution?.[1] ?? 0"
              size="sm"
              :min="0"
              class="w-24"
              @update:model-value="onPatchResolution(1, $event)"
            />
          </div>
        </div>
        <p class="text-[10px] text-dimmed">
          {{ t('subgraphProps.recorded_at', { time: subgraph.recordingContext.recordedAt || '—' }) }}
        </p>
      </div>
    </section>

    <!-- 标签 tags -->
    <section class="space-y-2">
      <label class="text-xs text-toned">tags</label>
      <UInputMenu
        :model-value="subgraph.tags ?? []"
        multiple
        creatable
        :items="allTagsList"
        size="sm"
        @update:model-value="(v: string[]) => $emit('update', { tags: v })"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'

const { t } = useI18n()
const toast = useToast()

interface RecordingContext {
  mouseCounts360: number
  resolution: [number, number]
  recordedAt: string
}
interface SubgraphLike {
  id: string
  label: string
  description?: string
  outputPins?: { id: string; name: string }[]
  recordingContext?: RecordingContext
  tags?: string[]
}

const props = defineProps<{
  subgraph: SubgraphLike | null
  allTags?: string[]
}>()
const emit = defineEmits<{ update: [patch: Record<string, any>] }>()

const allTagsList = computed(() => props.allTags ?? [])

const copied = ref(false)
let copiedTimer = 0
async function onCopyID() {
  if (!props.subgraph) return
  try {
    await navigator.clipboard.writeText(props.subgraph.id)
    copied.value = true
    window.clearTimeout(copiedTimer)
    copiedTimer = window.setTimeout(() => { copied.value = false }, 1500)
  } catch {
    toast.add({ title: t('toast.copy_failed'), color: 'error' })
  }
}

function onPatchRecording(key: string, v: any) {
  if (!props.subgraph?.recordingContext) return
  emit('update', {
    recordingContext: { ...props.subgraph.recordingContext, [key]: v },
  })
}

function onPatchResolution(idx: 0 | 1, v: number) {
  if (!props.subgraph?.recordingContext) return
  const r = [...(props.subgraph.recordingContext.resolution ?? [0, 0])] as [number, number]
  r[idx] = v
  emit('update', {
    recordingContext: { ...props.subgraph.recordingContext, resolution: r },
  })
}

function onResetRecording() {
  emit('update', { __resetRecording: true })
}
</script>
