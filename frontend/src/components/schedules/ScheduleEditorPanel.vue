<template>
  <div class="space-y-5 max-w-2xl">
    <section class="space-y-2">
      <div class="grid grid-cols-[120px_1fr] items-center gap-2 text-xs">
        <label class="text-dimmed">名称</label>
        <UInput v-model="draft.name" />
        <label class="text-dimmed">启用</label>
        <USwitch v-model="draft.enabled" />
      </div>
    </section>

    <section class="space-y-2">
      <div class="text-xs uppercase tracking-wider text-dimmed">Targets（按顺序跑）</div>
      <div v-for="(t, i) in draft.targets" :key="i" class="flex items-center gap-2 text-xs">
        <span class="text-dimmed w-4 tabular-nums">{{ i + 1 }}.</span>
        <USelect v-model="t.id" :items="containerItems" class="flex-1" />
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-arrow-up"
          :disabled="i === 0"
          @click="moveTarget(i, i - 1)"
        />
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-arrow-down"
          :disabled="i === draft.targets.length - 1"
          @click="moveTarget(i, i + 1)"
        />
        <UButton
          size="xs"
          variant="ghost"
          color="error"
          icon="i-tabler-x"
          @click="draft.targets.splice(i, 1)"
        />
      </div>
      <UButton size="xs" variant="soft" color="neutral" icon="i-tabler-plus" @click="addTarget"
        >添加容器</UButton
      >
    </section>

    <section class="space-y-2">
      <div class="text-xs uppercase tracking-wider text-dimmed">Trigger</div>
      <USelect v-model="draft.trigger.kind" :items="triggerKinds" class="w-48" />

      <div
        v-if="draft.trigger.kind === 'cron'"
        class="grid grid-cols-[120px_1fr] gap-2 text-xs items-center"
      >
        <label class="text-dimmed">subKind</label>
        <USelect v-model="draft.trigger.subKind" :items="cronSubKinds" />
        <template v-if="draft.trigger.subKind === 'daily'">
          <label class="text-dimmed">at (HH:MM)</label>
          <UInput v-model="draft.trigger.at" placeholder="05:00" />
        </template>
        <template v-else-if="draft.trigger.subKind === 'interval'">
          <label class="text-dimmed">每 N 分钟</label>
          <UInputNumber
            :model-value="draft.trigger.everyMinutes ?? 30"
            :min="1"
            @update:model-value="draft.trigger.everyMinutes = Number($event)"
          />
        </template>
      </div>

      <div
        v-if="draft.trigger.kind === 'hotkey'"
        class="grid grid-cols-[120px_1fr] gap-2 text-xs items-center"
      >
        <label class="text-dimmed">热键</label>
        <UInput v-model="draft.trigger.hotkey" placeholder="Ctrl+Shift+2" />
      </div>
    </section>

    <section class="space-y-2">
      <div class="text-xs uppercase tracking-wider text-dimmed">限制</div>
      <div class="grid grid-cols-[120px_1fr] gap-2 text-xs items-center">
        <label class="text-dimmed">timeout 分钟</label>
        <UInputNumber
          :model-value="draft.timeoutMinutes"
          :min="0"
          @update:model-value="draft.timeoutMinutes = Number($event)"
        />
        <label class="text-dimmed">onError</label>
        <USelect v-model="draft.onError" :items="onErrorOptions" />
      </div>
    </section>

    <div class="flex justify-end gap-2 pt-3 border-t border-default">
      <UButton variant="ghost" color="neutral" @click="$emit('cancel')">取消</UButton>
      <UButton color="primary" icon="i-tabler-check" @click="$emit('save', draft)">保存</UButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import type { Container, Schedule } from '@/lib/backend'

const props = defineProps<{ schedule: Schedule; containers: Container[] }>()
defineEmits<{ save: [s: Schedule]; cancel: [] }>()

// 深拷贝一次进 reactive；外部 schedule prop 变动也同步
const draft = reactive<Schedule>(JSON.parse(JSON.stringify(props.schedule)))
watch(
  () => props.schedule,
  (s) => Object.assign(draft, JSON.parse(JSON.stringify(s))),
)

const containerItems = computed(() =>
  props.containers.map((c) => ({ label: c.name || '(未命名)', value: c.id })),
)

const triggerKinds = [
  { label: '不绑（仅手动）', value: 'manual' },
  { label: 'Cron', value: 'cron' },
  { label: '启动后一次', value: 'once' },
  { label: '热键', value: 'hotkey' },
]

const cronSubKinds = [
  { label: '每日', value: 'daily' },
  { label: '间隔', value: 'interval' },
]

const onErrorOptions = [
  { label: '任一报错就停', value: 'stop' },
  { label: '继续下一个', value: 'continue' },
]

function addTarget() {
  draft.targets.push({ kind: 'container', id: props.containers[0]?.id ?? '' })
}

function moveTarget(from: number, to: number) {
  if (to < 0 || to >= draft.targets.length) return
  const [m] = draft.targets.splice(from, 1)
  draft.targets.splice(to, 0, m)
}
</script>
