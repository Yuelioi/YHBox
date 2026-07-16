<template>
  <div class="schedule-editor">
    <div class="schedule-editor__form">
      <section class="schedule-editor__section">
        <div class="schedule-editor__section-heading">
          <span><UIcon name="i-tabler-adjustments" class="size-4" /></span>
          <div>
            <h2>{{ t('schedule.basics_section') }}</h2>
            <p>{{ t('schedule.workspace.basics_hint') }}</p>
          </div>
        </div>
        <div class="grid gap-4 sm:grid-cols-[minmax(0,1fr)_auto]">
          <UFormField :label="t('schedule.name_label')" required>
            <UInput v-model="draft.name" :aria-label="t('schedule.name_label')" />
          </UFormField>
          <UFormField :label="t('schedule.enabled_label')">
            <div class="flex h-8 items-center gap-2">
              <USwitch v-model="draft.enabled" :aria-label="t('schedule.enabled_label')" />
              <span class="text-xs text-toned">{{
                draft.enabled ? t('schedule.enable') : t('schedule.disable')
              }}</span>
            </div>
          </UFormField>
        </div>
      </section>

      <section class="schedule-editor__section">
        <div class="schedule-editor__section-heading">
          <span><UIcon name="i-tabler-bolt" class="size-4" /></span>
          <div>
            <h2>{{ t('schedule.trigger_section') }}</h2>
            <p>{{ t('schedule.workspace.trigger_hint') }}</p>
          </div>
        </div>
        <div class="grid gap-4 sm:grid-cols-2">
          <UFormField :label="t('schedule.trigger_kind_label')">
            <USelect
              v-model="draft.trigger.kind"
              :items="triggerKinds"
              :aria-label="t('schedule.trigger_kind_label')"
            />
          </UFormField>
          <UFormField
            v-if="draft.trigger.kind === 'cron'"
            :label="t('schedule.cron_subkind_label')"
          >
            <USelect
              v-model="draft.trigger.subKind"
              :items="cronSubKinds"
              :aria-label="t('schedule.cron_subkind_label')"
            />
          </UFormField>
          <UFormField
            v-if="draft.trigger.kind === 'cron' && draft.trigger.subKind === 'daily'"
            :label="t('schedule.daily_at_label')"
          >
            <UInput
              v-model="draft.trigger.at"
              placeholder="05:00"
              :aria-label="t('schedule.daily_at_label')"
            />
          </UFormField>
          <UFormField
            v-if="draft.trigger.kind === 'cron' && draft.trigger.subKind === 'interval'"
            :label="t('schedule.interval_label')"
          >
            <UInputNumber
              :model-value="draft.trigger.everyMinutes ?? 30"
              :min="1"
              :aria-label="t('schedule.interval_label')"
              @update:model-value="draft.trigger.everyMinutes = Number($event)"
            />
          </UFormField>
          <UFormField v-if="draft.trigger.kind === 'hotkey'" :label="t('schedule.hotkey_label')">
            <UInput
              v-model="draft.trigger.hotkey"
              placeholder="Ctrl+Shift+2"
              :aria-label="t('schedule.hotkey_label')"
            />
          </UFormField>
        </div>
      </section>

      <section class="schedule-editor__section">
        <div class="schedule-editor__section-heading">
          <span><UIcon name="i-tabler-stack-2" class="size-4" /></span>
          <div class="min-w-0 flex-1">
            <h2>{{ t('schedule.targets_section') }}</h2>
            <p>{{ t('schedule.targets_hint') }}</p>
          </div>
          <UBadge size="xs" color="neutral" variant="subtle">{{ draft.targets.length }}</UBadge>
        </div>

        <div v-if="draft.targets.length" class="space-y-2">
          <div
            v-for="(target, index) in draft.targets"
            :key="`${target.id}-${index}`"
            class="schedule-target"
          >
            <span class="schedule-target__order">{{ index + 1 }}</span>
            <USelect
              v-model="target.id"
              :items="workflowItems"
              class="min-w-0 flex-1"
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
        <UButton size="sm" variant="soft" color="neutral" icon="i-tabler-plus" @click="addTarget">{{
          t('schedule.add_workflow')
        }}</UButton>
      </section>

      <section class="schedule-editor__section">
        <div class="schedule-editor__section-heading">
          <span><UIcon name="i-tabler-shield-half" class="size-4" /></span>
          <div>
            <h2>{{ t('schedule.limit_label') }}</h2>
            <p>{{ t('schedule.workspace.policy_hint') }}</p>
          </div>
        </div>
        <div class="grid gap-4 sm:grid-cols-2">
          <UFormField :label="t('schedule.timeout_label')" :hint="t('schedule.timeout_hint')">
            <UInputNumber
              :model-value="draft.timeoutMinutes"
              :min="0"
              :aria-label="t('schedule.timeout_label')"
              @update:model-value="draft.timeoutMinutes = Number($event)"
            />
          </UFormField>
          <UFormField :label="t('schedule.on_error_label')">
            <USelect
              v-model="draft.onError"
              :items="onErrorOptions"
              :aria-label="t('schedule.on_error_label')"
            />
          </UFormField>
        </div>
      </section>

      <footer class="schedule-editor__actions">
        <UButton variant="ghost" color="neutral" @click="$emit('cancel')">{{
          t('common.cancel')
        }}</UButton>
        <UButton
          color="primary"
          icon="i-tabler-check"
          :disabled="!draft.name.trim()"
          @click="$emit('save', draft)"
          >{{ t('common.save') }}</UButton
        >
      </footer>
    </div>

    <aside class="schedule-editor__summary">
      <div class="flex items-center justify-between gap-3">
        <p class="text-xs font-semibold text-default">{{ t('schedule.workspace.preview') }}</p>
        <StatusPill
          :status="draft.enabled ? 'online' : 'paused'"
          :label="draft.enabled ? t('schedule.enable') : t('schedule.disable')"
          :dot="draft.enabled"
        />
      </div>
      <div class="schedule-preview__trigger">
        <span><UIcon :name="previewIcon" class="size-6" /></span>
        <div>
          <p>{{ triggerPreview }}</p>
          <small>{{ t('schedule.workspace.trigger_preview_hint') }}</small>
        </div>
      </div>
      <div>
        <p class="schedule-preview__label">{{ t('schedule.targets_section') }}</p>
        <ol v-if="previewTargets.length" class="mt-2 space-y-2">
          <li
            v-for="(name, index) in previewTargets"
            :key="`${name}-${index}`"
            class="schedule-preview__target"
          >
            <span>{{ index + 1 }}</span
            ><strong>{{ name }}</strong>
          </li>
        </ol>
        <p v-else class="mt-2 text-xs text-dimmed">{{ t('schedule.workspace.no_targets') }}</p>
      </div>
      <dl class="schedule-preview__facts">
        <div>
          <dt>{{ t('schedule.timeout_label') }}</dt>
          <dd>{{ timeoutPreview }}</dd>
        </div>
        <div>
          <dt>{{ t('schedule.on_error_label') }}</dt>
          <dd>{{ t(`schedule.error_mode.${draft.onError}`) }}</dd>
        </div>
      </dl>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { TargetKind } from '@bindings/github.com/yottaapp/yotta/internal/services/schedule/models.js'
import type { Schedule } from '@/lib/backend'
import type { SourceView } from '@/app/transport/workflow'
import StatusPill from '@/components/common/StatusPill.vue'

const { t } = useI18n()
const { schedule, workflows } = defineProps<{ schedule: Schedule; workflows: SourceView[] }>()
defineEmits<{ save: [schedule: Schedule]; cancel: [] }>()
const draft = reactive<Schedule>(structuredClone(schedule))
watch(
  () => schedule,
  (value) => Object.assign(draft, structuredClone(value)),
)

const workflowItems = computed(() =>
  workflows.map((workflow) => ({
    label: workflow.name || t('common.untitled'),
    value: workflow.workflowId,
  })),
)
const triggerKinds = computed(() => [
  { label: t('schedule.workflow_unbound'), value: 'manual' },
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
const previewTargets = computed(() =>
  draft.targets.map(
    (target) =>
      workflows.find((workflow) => workflow.workflowId === target.id)?.name ??
      t('schedule.workflow_unnamed'),
  ),
)
const previewIcon = computed(() => {
  if (draft.trigger.kind === 'hotkey') return 'i-tabler-keyboard'
  if (draft.trigger.kind === 'once') return 'i-tabler-rocket'
  if (draft.trigger.kind === 'manual') return 'i-tabler-hand-click'
  return draft.trigger.subKind === 'daily' ? 'i-tabler-sun' : 'i-tabler-repeat'
})
const triggerPreview = computed(() => {
  if (draft.trigger.kind === 'cron' && draft.trigger.subKind === 'daily')
    return t('schedule.display.daily', { at: draft.trigger.at ?? '--:--' })
  if (draft.trigger.kind === 'cron' && draft.trigger.subKind === 'interval')
    return t('schedule.display.interval', { mins: draft.trigger.everyMinutes ?? 30 })
  if (draft.trigger.kind === 'hotkey')
    return t('schedule.display.hotkey', {
      key: draft.trigger.hotkey || t('schedule.workflow_unbound'),
    })
  return t(`schedule.trigger.${draft.trigger.kind}`)
})
const timeoutPreview = computed(() =>
  draft.timeoutMinutes > 0
    ? t('schedule.workspace.timeout_minutes', { n: draft.timeoutMinutes })
    : t('schedule.workspace.no_timeout'),
)

function addTarget() {
  draft.targets.push({ kind: TargetKind.TargetWorkflow, id: workflows[0]?.workflowId ?? '' })
}
function moveTarget(from: number, to: number) {
  if (to < 0 || to >= draft.targets.length) return
  const [target] = draft.targets.splice(from, 1)
  if (target) draft.targets.splice(to, 0, target)
}
</script>
