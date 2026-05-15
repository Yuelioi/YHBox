<template>
  <Teleport to="body">
    <div
      v-if="running"
      class="fixed bottom-4 right-4 z-50 flex items-center gap-3 px-3 py-2 rounded-lg bg-elevated/95 backdrop-blur border border-primary/40 shadow-lg shadow-black/30 cursor-pointer"
      @click="goToActions"
    >
      <span class="size-1.5 rounded-full bg-primary animate-pulse" />
      <div class="flex flex-col gap-0.5 min-w-0">
        <div class="flex items-center gap-1.5 text-xs">
          <span class="text-default truncate max-w-[180px]">{{
            runningAction?.name || '(未命名)'
          }}</span>
          <span class="text-dimmed tabular-nums shrink-0">
            {{ store.runningStepIndex + 1 }}/{{ store.runningStepTotal }}
          </span>
        </div>
        <div class="h-0.5 w-full rounded bg-default overflow-hidden">
          <div
            class="h-full bg-primary transition-all duration-150"
            :style="{ width: progressPercent + '%' }"
          />
        </div>
      </div>
      <button
        type="button"
        class="shrink-0 size-6 rounded inline-flex items-center justify-center text-dimmed hover:text-error hover:bg-error/10 cursor-pointer"
        title="停止"
        @click.stop="store.stop()"
      >
        <UIcon name="i-tabler-square" class="size-3.5" />
      </button>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useActionsStore } from '@/stores/actions'

const store = useActionsStore()
const router = useRouter()

// 不在 actions tab 的时候也显这个浮窗，让用户随时知道 runner 在跑什么 + 一键停。
const running = computed(() => !!store.runningID)
const runningAction = computed(() => store.list.find((a) => a.id === store.runningID))
const progressPercent = computed(() => {
  if (store.runningStepTotal === 0) return 0
  return Math.min(100, ((store.runningStepIndex + 1) / store.runningStepTotal) * 100)
})

function goToActions() {
  if (router.currentRoute.value.name !== 'actions') {
    void router.push('/actions')
  }
}
</script>
