<template>
  <div
    class="rounded-xl bg-default border border-default p-4 flex flex-col gap-3 transition-colors hover:border-accented"
  >
    <!-- Name + badges -->
    <div class="flex items-start justify-between gap-2 min-w-0">
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2 min-w-0">
          <span v-if="running" class="size-1.5 rounded-full bg-primary animate-pulse shrink-0" />
          <h3 class="text-sm font-medium text-highlighted truncate">
            {{ action.name || '(未命名)' }}
          </h3>
        </div>
        <p
          v-if="action.description"
          class="text-[11px] text-dimmed mt-0.5 line-clamp-2"
          :title="action.description"
        >
          {{ action.description }}
        </p>
        <div class="flex items-center gap-2 mt-1.5 flex-wrap">
          <span class="text-[11px] text-dimmed inline-flex items-center gap-1">
            <UIcon name="i-tabler-list-numbers" class="size-3" />
            {{ action.steps.length }} 步
          </span>
        </div>
      </div>
    </div>

    <!-- 运行进度条（仅 running 时显示） -->
    <div v-if="running && store.runningStepTotal > 0" class="space-y-1">
      <div class="h-1 rounded-full bg-elevated overflow-hidden">
        <div
          class="h-full bg-primary transition-all duration-150"
          :style="{ width: progressPercent + '%' }"
        />
      </div>
      <div class="flex items-center justify-between text-[10px] text-dimmed tabular-nums">
        <span>step {{ store.runningStepIndex + 1 }}/{{ store.runningStepTotal }}</span>
      </div>
    </div>

    <!-- Actions row -->
    <div class="flex items-center gap-1.5 pt-1 border-t border-default/60">
      <UButton
        v-if="!running"
        size="xs"
        color="primary"
        variant="soft"
        icon="i-tabler-player-play"
        @click="emit('run')"
      >
        运行
      </UButton>
      <UButton
        v-else
        size="xs"
        color="error"
        variant="soft"
        icon="i-tabler-square"
        @click="emit('stop')"
      >
        停止
      </UButton>
      <UButton
        size="xs"
        variant="ghost"
        color="neutral"
        icon="i-tabler-pencil"
        @click="emit('edit')"
      >
        编辑
      </UButton>
      <div class="flex-1" />
      <UButton
        size="xs"
        variant="ghost"
        color="error"
        icon="i-tabler-trash"
        @click="emit('delete')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useActionsStore, type Action } from '@/stores/actions'

const props = defineProps<{ action: Action; running: boolean }>()
const emit = defineEmits<{
  run: []
  stop: []
  edit: []
  delete: []
}>()

const store = useActionsStore()

const progressPercent = computed(() => {
  if (store.runningStepTotal === 0) return 0
  // step N 显示 N/total 的进度（step 已开始执行），停在 100% 等下次 loop
  return Math.min(100, ((store.runningStepIndex + 1) / store.runningStepTotal) * 100)
})
</script>
