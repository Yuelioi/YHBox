<template>
  <div class="space-y-6">
    <!-- 基础 -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-adjustments" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">{{ t('schedule.basics_section') }}</h2>
      </div>

      <div class="flex items-center justify-between gap-6">
        <div class="text-sm text-default">{{ t('schedule.name_label') }}</div>
        <UInput v-model="draft.name" class="w-64" />
      </div>

      <div class="border-t border-default/60" />

      <div class="flex items-center justify-between gap-6">
        <div class="text-sm text-default">{{ t('schedule.enabled_label') }}</div>
        <USwitch v-model="draft.enabled" />
      </div>
    </section>

    <!-- 目标容器 -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-stack-2" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">{{ t('schedule.targets_section') }}</h2>
        <span class="text-xs text-dimmed">({{ draft.targets.length }})</span>
      </div>
      <p class="text-xs text-dimmed">{{ t('schedule.targets_hint') }}</p>

      <div class="space-y-2">
        <div v-for="(tg, i) in draft.targets" :key="i" class="flex items-center gap-2">
          <span class="text-dimmed text-xs w-4 tabular-nums shrink-0">{{ i + 1 }}.</span>
          <USelect v-model="tg.id" :items="containerItems" class="flex-1" />
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
      </div>

      <UButton size="xs" variant="soft" color="neutral" icon="i-tabler-plus" @click="addTarget">{{
        t('schedule.add_container')
      }}</UButton>
    </section>

    <!-- 触发 -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-bolt" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">{{ t('schedule.trigger_section') }}</h2>
      </div>

      <div class="flex items-center justify-between gap-6">
        <div class="text-sm text-default">{{ t('schedule.trigger_kind_label') }}</div>
        <USelect v-model="draft.trigger.kind" :items="triggerKinds" class="w-48" />
      </div>

      <template v-if="draft.trigger.kind === 'cron'">
        <div class="border-t border-default/60" />
        <div class="flex items-center justify-between gap-6">
          <div class="text-sm text-default">{{ t('schedule.cron_subkind_label') }}</div>
          <USelect v-model="draft.trigger.subKind" :items="cronSubKinds" class="w-48" />
        </div>
        <div
          v-if="draft.trigger.subKind === 'daily'"
          class="flex items-center justify-between gap-6"
        >
          <div class="text-sm text-default">{{ t('schedule.daily_at_label') }}</div>
          <UInput v-model="draft.trigger.at" placeholder="05:00" class="w-32" />
        </div>
        <div
          v-else-if="draft.trigger.subKind === 'interval'"
          class="flex items-center justify-between gap-6"
        >
          <div class="text-sm text-default">{{ t('schedule.interval_label') }}</div>
          <UInputNumber
            :model-value="draft.trigger.everyMinutes ?? 30"
            :min="1"
            class="w-32"
            @update:model-value="draft.trigger.everyMinutes = Number($event)"
          />
        </div>
      </template>

      <template v-if="draft.trigger.kind === 'hotkey'">
        <div class="border-t border-default/60" />
        <div class="flex items-center justify-between gap-6">
          <div class="text-sm text-default">{{ t('schedule.hotkey_label') }}</div>
          <UInput v-model="draft.trigger.hotkey" placeholder="Ctrl+Shift+2" class="w-48" />
        </div>
      </template>
    </section>

    <!-- 限制 -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-shield-half" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">{{ t('schedule.limit_label') }}</h2>
      </div>

      <div class="flex items-center justify-between gap-6">
        <div>
          <div class="text-sm text-default">{{ t('schedule.timeout_label') }}</div>
          <p class="text-xs text-dimmed mt-0.5">{{ t('schedule.timeout_hint') }}</p>
        </div>
        <UInputNumber
          :model-value="draft.timeoutMinutes"
          :min="0"
          class="w-32"
          @update:model-value="draft.timeoutMinutes = Number($event)"
        />
      </div>

      <div class="border-t border-default/60" />

      <div class="flex items-center justify-between gap-6">
        <div class="text-sm text-default">{{ t('schedule.on_error_label') }}</div>
        <USelect v-model="draft.onError" :items="onErrorOptions" class="w-48" />
      </div>
    </section>

    <div class="flex justify-end gap-2">
      <UButton variant="ghost" color="neutral" @click="$emit('cancel')">{{
        t('common.cancel')
      }}</UButton>
      <UButton color="primary" icon="i-tabler-check" @click="$emit('save', draft)">{{
        t('common.save')
      }}</UButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Container, Schedule } from '@/lib/backend'

const { t } = useI18n()

const props = defineProps<{ schedule: Schedule; containers: Container[] }>()
defineEmits<{ save: [s: Schedule]; cancel: [] }>()

// 深拷贝一次进 reactive；外部 schedule prop 变动也同步
const draft = reactive<Schedule>(JSON.parse(JSON.stringify(props.schedule)))
watch(
  () => props.schedule,
  (s) => Object.assign(draft, JSON.parse(JSON.stringify(s))),
)

const containerItems = computed(() =>
  props.containers.map((c) => ({ label: c.name || t('common.untitled'), value: c.id })),
)

const triggerKinds = computed(() => [
  { label: t('schedule.container_unbound'), value: 'manual' },
  { label: t('schedule.trigger.cron'), value: 'cron' },
  { label: t('schedule.trigger.once'), value: 'once' },
  { label: t('schedule.trigger.hotkey'), value: 'hotkey' },
])

const cronSubKinds = computed(() => [
  { label: t('schedule.trigger.daily'), value: 'daily' },
  { label: t('schedule.trigger.interval'), value: 'interval' },
])

const onErrorOptions = computed(() => [
  { label: t('schedule.error_mode.stop'), value: 'stop' },
  { label: t('schedule.error_mode.continue'), value: 'continue' },
])

function addTarget() {
  draft.targets.push({ kind: 'container', id: props.containers[0]?.id ?? '' })
}

function moveTarget(from: number, to: number) {
  if (to < 0 || to >= draft.targets.length) return
  const [m] = draft.targets.splice(from, 1)
  draft.targets.splice(to, 0, m)
}
</script>
