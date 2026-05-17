<template>
  <div v-if="!subgraph" class="text-sm text-dimmed">未选中节点</div>
  <div v-else class="space-y-4">
    <header class="flex items-center gap-2 pb-3 border-b border-default">
      <UIcon name="i-tabler-package" class="size-4 text-fuchsia-300" />
      <h3 class="text-sm font-medium text-highlighted">{{ subgraph.label }}</h3>
      <UBadge size="xs" color="neutral" variant="soft">{{ subgraph.outputPins?.length ?? 0 }} 出口</UBadge>
    </header>

    <section class="space-y-2">
      <label class="text-xs text-toned">标签</label>
      <UInput
        :model-value="subgraph.label"
        size="sm"
        @update:model-value="(v: string) => $emit('update', { label: v })"
      />
    </section>

    <section class="space-y-2">
      <label class="text-xs text-toned">描述</label>
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
        <span class="text-xs text-toned font-medium">录制元数据</span>
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-refresh"
          title="重置为录制时的值（要重新录制才能改回）"
          class="ml-auto"
          @click="onResetRecording"
        />
      </div>
      <div class="space-y-2">
        <div class="space-y-1">
          <label class="text-[11px] text-dimmed">源 360° HID counts</label>
          <UInputNumber
            :model-value="subgraph.recordingContext.mouseCounts360"
            size="sm"
            :min="0"
            @update:model-value="onPatchRecording('mouseCounts360', $event)"
          />
        </div>
        <div class="space-y-1">
          <label class="text-[11px] text-dimmed">源分辨率</label>
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
          录制时间：{{ subgraph.recordingContext.recordedAt || '—' }}
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
import { computed } from 'vue'

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
