<template>
  <div class="space-y-6 max-w-2xl">
    <!-- 基础 -->
    <section class="space-y-3">
      <SectionHeader :title="t('schedule.basics_section')" icon="i-tabler-adjustments" />
      <UFormField :label="t('schedule.name_label')">
        <UInput v-model="draft.name" class="w-full" />
      </UFormField>
      <UFormField :label="t('schedule.enabled_label')">
        <USwitch v-model="draft.enabled" />
      </UFormField>
    </section>

    <!-- 目标容器 -->
    <section class="space-y-3">
      <SectionHeader
        :title="t('schedule.targets_section')"
        icon="i-tabler-stack-2"
        :count="draft.targets.length"
      />
      <p class="text-[11px] text-dimmed leading-snug">{{ t('schedule.targets_hint') }}</p>
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
      <UButton size="xs" variant="soft" color="neutral" icon="i-tabler-plus" @click="addTarget">{{
        t('schedule.add_container')
      }}</UButton>
    </section>

    <!-- 触发 -->
    <section class="space-y-3">
      <SectionHeader :title="t('schedule.trigger_section')" icon="i-tabler-bolt" />
      <UFormField :label="t('schedule.trigger_kind_label')">
        <USelect v-model="draft.trigger.kind" :items="triggerKinds" class="w-48" />
      </UFormField>

      <template v-if="draft.trigger.kind === 'cron'">
        <UFormField :label="t('schedule.cron_subkind_label')">
          <USelect v-model="draft.trigger.subKind" :items="cronSubKinds" class="w-48" />
        </UFormField>
        <UFormField v-if="draft.trigger.subKind === 'daily'" :label="t('schedule.daily_at_label')">
          <UInput v-model="draft.trigger.at" placeholder="05:00" class="w-32" />
        </UFormField>
        <UFormField
          v-else-if="draft.trigger.subKind === 'interval'"
          :label="t('schedule.interval_label')"
        >
          <UInputNumber
            :model-value="draft.trigger.everyMinutes ?? 30"
            :min="1"
            class="w-32"
            @update:model-value="draft.trigger.everyMinutes = Number($event)"
          />
        </UFormField>
      </template>

      <UFormField v-if="draft.trigger.kind === 'hotkey'" :label="t('schedule.hotkey_label')">
        <UInput v-model="draft.trigger.hotkey" placeholder="Ctrl+Shift+2" class="w-48" />
      </UFormField>
    </section>

    <!-- 限制 -->
    <section class="space-y-3">
      <SectionHeader :title="t('schedule.limit_label')" icon="i-tabler-shield-half" />
      <UFormField :label="t('schedule.timeout_label')" :hint="t('schedule.minutes')">
        <UInputNumber
          :model-value="draft.timeoutMinutes"
          :min="0"
          class="w-32"
          @update:model-value="draft.timeoutMinutes = Number($event)"
        />
      </UFormField>
      <UFormField :label="t('schedule.on_error_label')">
        <USelect v-model="draft.onError" :items="onErrorOptions" class="w-48" />
      </UFormField>
    </section>

    <div class="flex justify-end gap-2 pt-3 border-t border-default">
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
import SectionHeader from '@/components/common/SectionHeader.vue'

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
