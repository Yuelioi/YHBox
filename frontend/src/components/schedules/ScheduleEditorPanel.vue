<template>
  <div class="schedule-editor mx-auto w-full max-w-[860px]" data-testid="schedule-editor">
    <div class="schedule-editor__form flex min-w-0 flex-col gap-3">
      <section
        class="schedule-editor__section flex flex-col gap-3.5 rounded-[10px] border border-default bg-[var(--ui-surface)] px-5 py-[18px] max-[620px]:p-4"
      >
        <div class="schedule-editor__section-heading flex items-center gap-2.5 pb-0.5">
          <span
            class="flex size-[30px] shrink-0 items-center justify-center rounded-lg bg-elevated text-toned"
            ><UIcon name="i-tabler-adjustments" class="size-4"
          /></span>
          <h2 class="text-[13px] font-semibold text-highlighted">
            {{ t('schedule.basics_section') }}
          </h2>
        </div>

        <div
          class="schedule-editor__row grid min-w-0 grid-cols-[minmax(140px,1fr)_minmax(280px,380px)] items-center gap-6 border-t border-default/70 pt-3.5 max-[620px]:grid-cols-[minmax(0,1fr)] max-[620px]:gap-2"
        >
          <label
            class="schedule-editor__label min-w-0 text-xs font-medium text-toned [&>span]:text-error"
            for="schedule-name"
          >
            {{ t('schedule.name_label') }} <span aria-hidden="true">*</span>
          </label>
          <UInput
            id="schedule-name"
            v-model="draft.name"
            class="schedule-editor__control w-full min-w-0"
            :aria-label="t('schedule.name_label')"
          />
        </div>
        <p
          v-if="showValidation && nameError"
          class="schedule-editor__validation text-[11px] leading-snug text-error"
          role="alert"
        >
          {{ nameError }}
        </p>

        <div
          class="schedule-editor__row grid min-w-0 grid-cols-[minmax(140px,1fr)_minmax(280px,380px)] items-center gap-6 border-t border-default/70 pt-3.5 max-[620px]:grid-cols-[minmax(0,1fr)] max-[620px]:gap-2"
        >
          <label
            class="schedule-editor__label min-w-0 text-xs font-medium text-toned"
            for="schedule-description"
          >
            {{ t('schedule.description_label') }}
          </label>
          <UTextarea
            id="schedule-description"
            v-model="draft.description"
            :rows="3"
            autoresize
            class="schedule-editor__control w-full min-w-0"
            :placeholder="t('schedule.description_placeholder')"
          />
        </div>

        <div
          class="schedule-editor__row grid min-w-0 grid-cols-[minmax(140px,1fr)_minmax(280px,380px)] items-center gap-6 border-t border-default/70 pt-3.5 max-[620px]:grid-cols-[minmax(0,1fr)] max-[620px]:gap-2"
        >
          <span class="schedule-editor__label min-w-0 text-xs font-medium text-toned">{{
            t('common.category')
          }}</span>
          <UInputMenu
            v-model="draft.category"
            :items="availableCategories"
            :create-item="'always'"
            class="schedule-editor__control w-full min-w-0"
            :placeholder="t('schedule.category_placeholder')"
            @create="createCategory"
          />
        </div>

        <div
          class="schedule-editor__row grid min-w-0 grid-cols-[minmax(140px,1fr)_minmax(280px,380px)] items-center gap-6 border-t border-default/70 pt-3.5 max-[620px]:grid-cols-[minmax(0,1fr)] max-[620px]:gap-2"
        >
          <span class="schedule-editor__label min-w-0 text-xs font-medium text-toned">{{
            t('common.tags')
          }}</span>
          <UInputMenu
            v-model="draft.tags"
            :items="availableTags"
            :create-item="'always'"
            multiple
            class="schedule-editor__control w-full min-w-0"
            :placeholder="t('schedule.tags_placeholder')"
            @create="createTag"
          />
        </div>

        <div
          class="schedule-editor__row grid min-w-0 grid-cols-[minmax(140px,1fr)_minmax(280px,380px)] items-center gap-6 border-t border-default/70 pt-3.5 max-[620px]:grid-cols-[minmax(0,1fr)] max-[620px]:gap-2"
        >
          <span class="schedule-editor__label min-w-0 text-xs font-medium text-toned">{{
            t('schedule.enabled_label')
          }}</span>
          <div
            class="schedule-editor__control schedule-editor__switch flex w-full min-w-0 items-center gap-2 text-[11px] text-dimmed"
          >
            <USwitch v-model="draft.enabled" :aria-label="t('schedule.enabled_label')" />
            <span>{{ draft.enabled ? t('schedule.enable') : t('schedule.disable') }}</span>
          </div>
        </div>
      </section>

      <section
        class="schedule-editor__section flex flex-col gap-3.5 rounded-[10px] border border-default bg-[var(--ui-surface)] px-5 py-[18px] max-[620px]:p-4"
      >
        <div class="schedule-editor__section-heading flex items-center gap-2.5 pb-0.5">
          <span
            class="flex size-[30px] shrink-0 items-center justify-center rounded-lg bg-elevated text-toned"
            ><UIcon name="i-tabler-stack-2" class="size-4"
          /></span>
          <h2 class="text-[13px] font-semibold text-highlighted">
            {{ t('schedule.targets_section') }}
          </h2>
          <UBadge size="xs" color="neutral" variant="subtle">{{ draft.targets.length }}</UBadge>
        </div>

        <div v-if="draft.targets.length" class="space-y-2">
          <div
            v-for="(target, index) in draft.targets"
            :key="`${target.id}-${index}`"
            class="schedule-target flex items-center gap-1.5 rounded-[9px] border border-default bg-elevated/30 p-2"
            data-testid="schedule-target"
            :data-workflow-id="target.id"
          >
            <span
              class="schedule-target__order flex size-6 shrink-0 items-center justify-center rounded-full bg-primary/10 text-[11px] font-semibold text-primary"
              >{{ index + 1 }}</span
            >
            <AdaptiveSelect
              v-model="target.id"
              :items="workflowItems"
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
        <div
          v-else
          class="schedule-target-empty flex min-h-[88px] items-center justify-center gap-2 rounded-[10px] border border-dashed border-default text-xs text-dimmed"
        >
          <UIcon name="i-tabler-box-off" class="size-5 text-dimmed" />
          <span>{{ t('schedule.workspace.no_targets_hint') }}</span>
        </div>
        <p
          v-if="showValidation && targetsError"
          class="schedule-editor__validation text-[11px] leading-snug text-error"
          role="alert"
        >
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

      <section
        class="schedule-editor__section flex flex-col gap-3.5 rounded-[10px] border border-default bg-[var(--ui-surface)] px-5 py-[18px] max-[620px]:p-4"
      >
        <div class="schedule-editor__section-heading flex items-center gap-2.5 pb-0.5">
          <span
            class="flex size-[30px] shrink-0 items-center justify-center rounded-lg bg-elevated text-toned"
            ><UIcon name="i-tabler-bolt" class="size-4"
          /></span>
          <h2 class="text-[13px] font-semibold text-highlighted">
            {{ t('schedule.trigger_section') }}
          </h2>
        </div>

        <div
          class="schedule-editor__row grid min-w-0 grid-cols-[minmax(140px,1fr)_minmax(280px,380px)] items-center gap-6 border-t border-default/70 pt-3.5 max-[620px]:grid-cols-[minmax(0,1fr)] max-[620px]:gap-2"
        >
          <span class="schedule-editor__label min-w-0 text-xs font-medium text-toned">{{
            t('schedule.trigger_kind_label')
          }}</span>
          <AdaptiveSelect
            v-model="draft.trigger.kind"
            :items="triggerKinds"
            class="schedule-editor__control w-full min-w-0"
            width-mode="fill"
            :aria-label="t('schedule.trigger_kind_label')"
          />
        </div>

        <template v-if="draft.trigger.kind === 'cron'">
          <div
            class="schedule-editor__row grid min-w-0 grid-cols-[minmax(140px,1fr)_minmax(280px,380px)] items-center gap-6 border-t border-default/70 pt-3.5 max-[620px]:grid-cols-[minmax(0,1fr)] max-[620px]:gap-2"
          >
            <span class="schedule-editor__label min-w-0 text-xs font-medium text-toned">{{
              t('schedule.cron_subkind_label')
            }}</span>
            <AdaptiveSelect
              v-model="cronSubKindModel"
              :items="cronSubKinds"
              class="schedule-editor__control w-full min-w-0"
              width-mode="fill"
              :aria-label="t('schedule.cron_subkind_label')"
            />
          </div>
          <div
            v-if="draft.trigger.subKind === 'daily'"
            class="schedule-editor__row grid min-w-0 grid-cols-[minmax(140px,1fr)_minmax(280px,380px)] items-center gap-6 border-t border-default/70 pt-3.5 max-[620px]:grid-cols-[minmax(0,1fr)] max-[620px]:gap-2"
          >
            <span class="schedule-editor__label min-w-0 text-xs font-medium text-toned">{{
              t('schedule.daily_at_label')
            }}</span>
            <UInput
              v-model="draft.trigger.at"
              class="schedule-editor__control w-full min-w-0"
              placeholder="05:00"
              :aria-label="t('schedule.daily_at_label')"
            />
          </div>
          <div
            v-else-if="draft.trigger.subKind === 'interval'"
            class="schedule-editor__row grid min-w-0 grid-cols-[minmax(140px,1fr)_minmax(280px,380px)] items-center gap-6 border-t border-default/70 pt-3.5 max-[620px]:grid-cols-[minmax(0,1fr)] max-[620px]:gap-2"
          >
            <span class="schedule-editor__label min-w-0 text-xs font-medium text-toned">{{
              t('schedule.interval_label')
            }}</span>
            <UInputNumber
              :model-value="draft.trigger.everyMinutes"
              :min="1"
              class="schedule-editor__control w-full min-w-0"
              :aria-label="t('schedule.interval_label')"
              @update:model-value="draft.trigger.everyMinutes = Number($event)"
            />
          </div>
        </template>

        <div
          v-if="draft.trigger.kind === 'hotkey'"
          class="schedule-editor__row grid min-w-0 grid-cols-[minmax(140px,1fr)_minmax(280px,380px)] items-center gap-6 border-t border-default/70 pt-3.5 max-[620px]:grid-cols-[minmax(0,1fr)] max-[620px]:gap-2"
        >
          <span class="schedule-editor__label min-w-0 text-xs font-medium text-toned">{{
            t('schedule.hotkey_label')
          }}</span>
          <UInput
            v-model="draft.trigger.hotkey"
            class="schedule-editor__control w-full min-w-0"
            placeholder="Ctrl+Shift+2"
            :aria-label="t('schedule.hotkey_label')"
          />
        </div>
        <p
          v-if="showValidation && triggerError"
          class="schedule-editor__validation text-[11px] leading-snug text-error"
          role="alert"
        >
          {{ triggerError }}
        </p>
      </section>

      <section
        class="schedule-editor__section flex flex-col gap-3.5 rounded-[10px] border border-default bg-[var(--ui-surface)] px-5 py-[18px] max-[620px]:p-4"
        data-testid="schedule-advanced"
      >
        <div class="schedule-editor__section-heading flex items-center gap-2.5 pb-0.5">
          <span
            class="flex size-[30px] shrink-0 items-center justify-center rounded-lg bg-elevated text-toned"
            ><UIcon name="i-tabler-adjustments-horizontal" class="size-4"
          /></span>
          <h2 class="text-[13px] font-semibold text-highlighted">
            {{ t('schedule.advanced_settings') }}
          </h2>
        </div>

        <div
          class="schedule-editor__row grid min-w-0 grid-cols-[minmax(140px,1fr)_minmax(280px,380px)] items-center gap-6 border-t border-default/70 pt-3.5 max-[620px]:grid-cols-[minmax(0,1fr)] max-[620px]:gap-2"
        >
          <span class="schedule-editor__label min-w-0 text-xs font-medium text-toned">
            {{ t('schedule.target_interval_label') }}
            <small class="mt-0.5 block text-[10px] font-normal text-dimmed">{{
              t('schedule.target_interval_hint')
            }}</small>
          </span>
          <UInputNumber
            :model-value="draft.targetIntervalSeconds"
            :min="0"
            :step="1"
            class="schedule-editor__control w-full min-w-0"
            data-testid="schedule-target-interval"
            :aria-label="t('schedule.target_interval_label')"
            @update:model-value="draft.targetIntervalSeconds = nonNegativeInteger($event)"
          />
        </div>

        <div
          class="schedule-editor__row grid min-w-0 grid-cols-[minmax(140px,1fr)_minmax(280px,380px)] items-center gap-6 border-t border-default/70 pt-3.5 max-[620px]:grid-cols-[minmax(0,1fr)] max-[620px]:gap-2"
        >
          <span class="schedule-editor__label min-w-0 text-xs font-medium text-toned">
            {{ t('schedule.timeout_label') }}
            <small class="mt-0.5 block text-[10px] font-normal text-dimmed">{{
              t('schedule.timeout_hint')
            }}</small>
          </span>
          <UInputNumber
            :model-value="draft.timeoutMinutes"
            :min="0"
            class="schedule-editor__control w-full min-w-0"
            :aria-label="t('schedule.timeout_label')"
            @update:model-value="draft.timeoutMinutes = nonNegativeInteger($event)"
          />
        </div>

        <div
          class="schedule-editor__row grid min-w-0 grid-cols-[minmax(140px,1fr)_minmax(280px,380px)] items-center gap-6 border-t border-default/70 pt-3.5 max-[620px]:grid-cols-[minmax(0,1fr)] max-[620px]:gap-2"
        >
          <span class="schedule-editor__label min-w-0 text-xs font-medium text-toned">{{
            t('schedule.on_error_label')
          }}</span>
          <AdaptiveSelect
            v-model="draft.onError"
            :items="onErrorOptions"
            class="schedule-editor__control w-full min-w-0"
            width-mode="fill"
            :aria-label="t('schedule.on_error_label')"
          />
        </div>
      </section>

      <footer class="schedule-editor__actions flex justify-end gap-2 pt-1">
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
import type { SourceView } from '@/app/transport/workflow'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'

const { t } = useI18n()
const { schedule, workflows, categoryOptions, tagOptions } = defineProps<{
  schedule: Schedule
  workflows: SourceView[]
  categoryOptions: string[]
  tagOptions: string[]
}>()
const emit = defineEmits<{ save: [schedule: Schedule]; cancel: [] }>()
const draft = reactive<Schedule>(cloneSchedule(schedule))
const showValidation = ref(false)
const createdCategories = ref<string[]>([])
const createdTags = ref<string[]>([])

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
  value.description ??= ''
  value.category ??= ''
  value.tags ??= []
  value.trigger ??= { kind: TriggerKind.TriggerManual }
  if (!value.trigger.kind) value.trigger.kind = TriggerKind.TriggerManual
  if (!value.onError) value.onError = OnErrorMode.OnErrorStop
  value.targetIntervalSeconds = nonNegativeInteger(value.targetIntervalSeconds)
  value.timeoutMinutes = nonNegativeInteger(value.timeoutMinutes)
  normalizeTrigger(value)
}

function nonNegativeInteger(value: unknown): number {
  const number = Number(value)
  return Number.isFinite(number) && number >= 0 ? Math.trunc(number) : 0
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

const workflowItems = computed(() =>
  workflows.map((workflow) => ({
    label: workflow.name || t('common.untitled'),
    value: workflow.workflowId,
  })),
)
const availableCategories = computed(() =>
  uniqueValues([...(categoryOptions ?? []), ...createdCategories.value, draft.category ?? '']),
)
const availableTags = computed(() =>
  uniqueValues([...(tagOptions ?? []), ...createdTags.value, ...(draft.tags ?? [])]),
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
    kind: TargetKind.TargetWorkflow,
    id: workflows[0]?.workflowId ?? '',
  })
}
function moveTarget(from: number, to: number) {
  if (to < 0 || to >= draft.targets.length) return
  const [target] = draft.targets.splice(from, 1)
  if (target) draft.targets.splice(to, 0, target)
}
function createCategory(value: string): void {
  const category = value.trim()
  if (!category) return
  createdCategories.value = uniqueValues([...createdCategories.value, category])
  draft.category = category
}
function createTag(value: string): void {
  const tag = value.trim()
  if (!tag) return
  createdTags.value = uniqueValues([...createdTags.value, tag])
  draft.tags = uniqueValues([...(draft.tags ?? []), tag])
}
function uniqueValues(values: string[]): string[] {
  const seen = new Set<string>()
  return values
    .map((value) => value.trim())
    .filter((value) => {
      const key = value.toLocaleLowerCase()
      if (!key || seen.has(key)) return false
      seen.add(key)
      return true
    })
}
function submit(): void {
  normalizeSchedule(draft)
  showValidation.value = true
  if (nameError.value || targetsError.value || triggerError.value) return
  emit('save', cloneSchedule(draft))
}
</script>
