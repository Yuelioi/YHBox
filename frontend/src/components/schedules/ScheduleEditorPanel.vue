<template>
  <div class="schedule-editor" data-testid="schedule-editor">
    <div class="schedule-editor__form">
      <section class="schedule-editor__section">
        <div class="schedule-editor__section-heading">
          <span><UIcon name="i-tabler-adjustments" class="size-4" /></span>
          <h2>{{ t('schedule.basics_section') }}</h2>
        </div>

        <div class="schedule-editor__row">
          <label class="schedule-editor__label" for="schedule-name">
            {{ t('schedule.name_label') }} <span aria-hidden="true">*</span>
          </label>
          <UInput
            id="schedule-name"
            v-model="draft.name"
            class="schedule-editor__control"
            :aria-label="t('schedule.name_label')"
          />
        </div>
        <p v-if="showValidation && nameError" class="schedule-editor__validation" role="alert">
          {{ nameError }}
        </p>

        <div class="schedule-editor__row">
          <span class="schedule-editor__label">{{ t('schedule.enabled_label') }}</span>
          <div class="schedule-editor__control schedule-editor__switch">
            <USwitch v-model="draft.enabled" :aria-label="t('schedule.enabled_label')" />
            <span>{{ draft.enabled ? t('schedule.enable') : t('schedule.disable') }}</span>
          </div>
        </div>
      </section>

      <section class="schedule-editor__section">
        <div class="schedule-editor__section-heading">
          <span><UIcon name="i-tabler-stack-2" class="size-4" /></span>
          <h2>{{ t('schedule.targets_section') }}</h2>
          <UBadge size="xs" color="neutral" variant="subtle">{{ draft.targets.length }}</UBadge>
        </div>

        <div v-if="draft.targets.length" class="space-y-2">
          <div
            v-for="(target, index) in draft.targets"
            :key="`${target.id}-${index}`"
            class="schedule-target"
            data-testid="schedule-target"
            :data-workflow-id="target.id"
          >
            <span class="schedule-target__order">{{ index + 1 }}</span>
            <AdaptiveSelect
              v-model="target.id"
              :items="installationItems"
              class="min-w-0 flex-1"
              width-mode="fill"
              :aria-label="t('schedule.target_n', { n: index + 1 })"
            />
            <UButton
              size="xs"
              variant="ghost"
              color="neutral"
              icon="i-tabler-arrow-up"
              :disabled="index === 0"
              :aria-label="t('schedule.move_up')"
              @click="moveTarget(index, index - 1)"
            />
            <UButton
              size="xs"
              variant="ghost"
              color="neutral"
              icon="i-tabler-arrow-down"
              :disabled="index === draft.targets.length - 1"
              :aria-label="t('schedule.move_down')"
              @click="moveTarget(index, index + 1)"
            />
            <UButton
              size="xs"
              variant="ghost"
              color="error"
              icon="i-tabler-x"
              :aria-label="t('schedule.remove_target')"
              @click="draft.targets.splice(index, 1)"
            />
          </div>
        </div>
        <div v-else class="schedule-target-empty">
          <UIcon name="i-tabler-box-off" class="size-5 text-dimmed" />
          <span>{{ t('schedule.workspace.no_targets_hint') }}</span>
        </div>
        <p v-if="showValidation && targetsError" class="schedule-editor__validation" role="alert">
          {{ targetsError }}
        </p>
        <div>
          <UButton
            size="sm"
            variant="soft"
            color="neutral"
            icon="i-tabler-plus"
            data-testid="schedule-add-target"
            @click="addTarget"
          >
            {{ t('schedule.add_workflow') }}
          </UButton>
        </div>
      </section>

      <section class="schedule-editor__section">
        <div class="schedule-editor__section-heading">
          <span><UIcon name="i-tabler-bolt" class="size-4" /></span>
          <h2>{{ t('schedule.trigger_section') }}</h2>
        </div>

        <div class="schedule-editor__row">
          <span class="schedule-editor__label">{{ t('schedule.trigger_kind_label') }}</span>
          <AdaptiveSelect
            v-model="draft.trigger.kind"
            :items="triggerKinds"
            class="schedule-editor__control"
            width-mode="fill"
            :aria-label="t('schedule.trigger_kind_label')"
          />
        </div>

        <template v-if="draft.trigger.kind === 'cron'">
          <div class="schedule-editor__row">
            <span class="schedule-editor__label">{{ t('schedule.cron_subkind_label') }}</span>
            <AdaptiveSelect
              v-model="cronSubKindModel"
              :items="cronSubKinds"
              class="schedule-editor__control"
              width-mode="fill"
              :aria-label="t('schedule.cron_subkind_label')"
            />
          </div>
          <div v-if="draft.trigger.subKind === 'daily'" class="schedule-editor__row">
            <span class="schedule-editor__label">{{ t('schedule.daily_at_label') }}</span>
            <UInput
              v-model="draft.trigger.at"
              class="schedule-editor__control"
              placeholder="05:00"
              :aria-label="t('schedule.daily_at_label')"
            />
          </div>
          <div v-else-if="draft.trigger.subKind === 'interval'" class="schedule-editor__row">
            <span class="schedule-editor__label">{{ t('schedule.interval_label') }}</span>
            <UInputNumber
              :model-value="draft.trigger.everyMinutes"
              :min="1"
              class="schedule-editor__control"
              :aria-label="t('schedule.interval_label')"
              @update:model-value="draft.trigger.everyMinutes = Number($event)"
            />
          </div>
        </template>

        <div v-if="draft.trigger.kind === 'hotkey'" class="schedule-editor__row">
          <span class="schedule-editor__label">{{ t('schedule.hotkey_label') }}</span>
          <UInput
            v-model="draft.trigger.hotkey"
            class="schedule-editor__control"
            placeholder="Ctrl+Shift+2"
            :aria-label="t('schedule.hotkey_label')"
          />
        </div>
        <p v-if="showValidation && triggerError" class="schedule-editor__validation" role="alert">
          {{ triggerError }}
        </p>
      </section>

      <section class="schedule-editor__section">
        <div class="schedule-editor__section-heading">
          <span><UIcon name="i-tabler-shield-half" class="size-4" /></span>
          <h2>{{ t('schedule.limit_label') }}</h2>
        </div>

        <div class="schedule-editor__row">
          <span class="schedule-editor__label">
            {{ t('schedule.timeout_label') }}
            <small>{{ t('schedule.timeout_hint') }}</small>
          </span>
          <UInputNumber
            :model-value="draft.timeoutMinutes"
            :min="0"
            class="schedule-editor__control"
            :aria-label="t('schedule.timeout_label')"
            @update:model-value="draft.timeoutMinutes = Math.max(0, Number($event))"
          />
        </div>

        <div class="schedule-editor__row">
          <span class="schedule-editor__label">{{ t('schedule.on_error_label') }}</span>
          <AdaptiveSelect
            v-model="draft.onError"
            :items="onErrorOptions"
            class="schedule-editor__control"
            width-mode="fill"
            :aria-label="t('schedule.on_error_label')"
          />
        </div>
      </section>

      <footer class="schedule-editor__actions">
        <UButton variant="ghost" color="neutral" @click="emit('cancel')">
          {{ t('common.cancel') }}
        </UButton>
        <UButton color="primary" icon="i-tabler-check" data-testid="schedule-save" @click="submit">
          {{ t('common.save') }}
        </UButton>
      </footer>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, toRaw, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  CronSubKind,
  OnErrorMode,
  TargetKind,
  TriggerKind,
} from '@bindings/github.com/yottaapp/yotta/internal/services/schedule/models.js'
import type { Schedule } from '@/lib/backend'
import type { InstallationView } from '@/app/transport/workflow'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'

const { t } = useI18n()
const { schedule, installations } = defineProps<{
  schedule: Schedule
  installations: InstallationView[]
}>()
const emit = defineEmits<{ save: [schedule: Schedule]; cancel: [] }>()
const draft = reactive<Schedule>(cloneSchedule(schedule))
const showValidation = ref(false)

watch(
  () => schedule,
  (value) => {
    Object.assign(draft, cloneSchedule(value))
    showValidation.value = false
  },
)
watch(
  () => [draft.trigger.kind, draft.trigger.subKind] as const,
  () => normalizeTrigger(draft),
  { immediate: true },
)

function cloneSchedule(value: Schedule): Schedule {
  const clone = structuredClone(toRaw(value))
  clone.targets ??= []
  normalizeSchedule(clone)
  return clone
}

function normalizeSchedule(value: Schedule): void {
  value.trigger ??= { kind: TriggerKind.TriggerManual }
  if (!value.trigger.kind) value.trigger.kind = TriggerKind.TriggerManual
  if (!value.onError) value.onError = OnErrorMode.OnErrorStop
  if (!Number.isFinite(value.timeoutMinutes) || value.timeoutMinutes < 0) value.timeoutMinutes = 0
  normalizeTrigger(value)
}

function normalizeTrigger(value: Schedule): void {
  if (value.trigger.kind !== TriggerKind.TriggerCron) return
  if (!value.trigger.subKind) value.trigger.subKind = CronSubKind.CronDaily
  if (value.trigger.subKind === CronSubKind.CronDaily && !value.trigger.at?.trim()) {
    value.trigger.at = '05:00'
  }
  if (
    value.trigger.subKind === CronSubKind.CronInterval &&
    (!Number.isFinite(value.trigger.everyMinutes) || (value.trigger.everyMinutes ?? 0) <= 0)
  ) {
    value.trigger.everyMinutes = 30
  }
}

const installationItems = computed(() =>
  installations.map((installation) => ({
    label: installation.name || t('common.untitled'),
    value: installation.installationId,
  })),
)
const triggerKinds = computed(() => [
  { label: t('schedule.workflow_unbound'), value: TriggerKind.TriggerManual },
  { label: t('schedule.trigger.cron'), value: TriggerKind.TriggerCron },
  { label: t('schedule.trigger.once'), value: TriggerKind.TriggerOnce },
  { label: t('schedule.trigger.hotkey'), value: TriggerKind.TriggerHotkey },
])
const cronSubKinds = computed(() => [
  { label: t('schedule.trigger.daily'), value: CronSubKind.CronDaily },
  { label: t('schedule.trigger.interval'), value: CronSubKind.CronInterval },
])
const onErrorOptions = computed(() => [
  { label: t('schedule.error_mode.stop'), value: OnErrorMode.OnErrorStop },
  { label: t('schedule.error_mode.continue'), value: OnErrorMode.OnErrorContinue },
])
const cronSubKindModel = computed<CronSubKind>({
  get: () => draft.trigger.subKind ?? CronSubKind.CronDaily,
  set: (value) => {
    draft.trigger.subKind = value
  },
})

const nameError = computed(() => (!draft.name.trim() ? t('schedule.validation.name') : ''))
const targetsError = computed(() => {
  if (!draft.targets.length) return t('schedule.validation.targets')
  if (draft.targets.some((target) => !target.id)) return t('schedule.validation.target')
  return ''
})
const triggerError = computed(() => {
  if (draft.trigger.kind === TriggerKind.TriggerCron) {
    if (draft.trigger.subKind === CronSubKind.CronDaily) {
      return /^([01]\d|2[0-3]):[0-5]\d$/.test(draft.trigger.at ?? '')
        ? ''
        : t('schedule.validation.daily')
    }
    if (draft.trigger.subKind === CronSubKind.CronInterval) {
      return (draft.trigger.everyMinutes ?? 0) > 0 ? '' : t('schedule.validation.interval')
    }
  }
  if (draft.trigger.kind === TriggerKind.TriggerHotkey && !draft.trigger.hotkey?.trim()) {
    return t('schedule.validation.hotkey')
  }
  return ''
})

function addTarget() {
  draft.targets.push({
    kind: TargetKind.TargetWorkflowInstallation,
    id: installations[0]?.installationId ?? '',
  })
}
function moveTarget(from: number, to: number) {
  if (to < 0 || to >= draft.targets.length) return
  const [target] = draft.targets.splice(from, 1)
  if (target) draft.targets.splice(to, 0, target)
}
function submit(): void {
  normalizeSchedule(draft)
  showValidation.value = true
  if (nameError.value || targetsError.value || triggerError.value) return
  emit('save', cloneSchedule(draft))
}
</script>
