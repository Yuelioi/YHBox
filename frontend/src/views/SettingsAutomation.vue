<template>
  <div class="settings-page">
    <div class="flex items-start gap-3 rounded-xl border border-error/30 bg-error/5 p-4">
      <UIcon
        name="i-tabler-device-desktop-cog"
        class="mt-0.5 size-5 shrink-0 text-error"
        aria-hidden="true"
      />
      <div class="min-w-0">
        <p class="text-sm font-medium text-default">{{ t('settingsAutomation.security.title') }}</p>
        <p class="mt-1 text-xs leading-relaxed text-dimmed">
          {{ t('settingsAutomation.security.hint') }}
        </p>
      </div>
    </div>

    <SettingsSection
      :title="t('settingsAutomation.targets.title')"
      :description="t('settingsAutomation.targets.hint')"
      icon="i-tabler-pointer-cog"
    >
      <template #badge>
        <UBadge size="xs" color="neutral" variant="subtle">{{ draft.length }}</UBadge>
      </template>
      <template #actions>
        <UButton
          size="sm"
          color="primary"
          variant="soft"
          icon="i-tabler-plus"
          :disabled="applications.length === 0"
          @click="addTarget"
        >
          {{ t('settingsAutomation.targets.add') }}
        </UButton>
      </template>

      <div v-if="draft.length" class="space-y-3">
        <article v-for="(target, index) in draft" :key="target.slot" class="ai-profile">
          <button
            type="button"
            class="flex min-h-11 min-w-0 flex-1 cursor-pointer items-center gap-3 text-left"
            :aria-expanded="expandedSlot === target.slot"
            :aria-controls="`automation-target-${target.slot}`"
            @click="toggleExpanded(target.slot)"
          >
            <span
              class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-default bg-elevated"
            >
              <UIcon name="i-tabler-pointer-cog" class="size-4 text-toned" aria-hidden="true" />
            </span>
            <span class="min-w-0 flex-1">
              <span class="flex flex-wrap items-center gap-2">
                <span class="truncate text-sm font-medium text-default">{{ target.label }}</span>
                <UBadge
                  :color="target.workflowConsent ? 'success' : 'warning'"
                  size="xs"
                  variant="subtle"
                  :icon="
                    target.workflowConsent ? 'i-tabler-shield-check' : 'i-tabler-shield-exclamation'
                  "
                >
                  {{
                    t(
                      target.workflowConsent
                        ? 'settingsAutomation.targets.workflow_allowed'
                        : 'settingsAutomation.targets.consent_required',
                    )
                  }}
                </UBadge>
                <UBadge size="xs" color="neutral" variant="subtle">
                  {{ t(`settingsAutomation.backend.${target.inputBackend}`) }}
                </UBadge>
                <UBadge size="xs" color="neutral" variant="subtle">
                  {{ t(`settingsAutomation.captureBackend.${target.captureBackend}`) }}
                </UBadge>
              </span>
              <span class="mt-1 block truncate text-xs text-dimmed">
                {{ applicationLabel(target.applicationSlot) }} · <code>{{ target.slot }}</code>
              </span>
            </span>
            <UIcon
              name="i-tabler-chevron-down"
              class="size-4 shrink-0 text-dimmed transition-transform"
              :class="expandedSlot === target.slot ? 'rotate-180' : ''"
              aria-hidden="true"
            />
          </button>

          <div
            v-if="expandedSlot === target.slot"
            :id="`automation-target-${target.slot}`"
            class="ai-profile__details"
          >
            <div class="grid gap-4 sm:grid-cols-2">
              <UFormField :label="t('settingsAutomation.targets.name_label')" required>
                <UInput v-model.trim="target.label" size="sm" @change="commit" />
              </UFormField>
              <UFormField
                :label="t('settingsAutomation.targets.slot_label')"
                :hint="t('settingsAutomation.targets.slot_hint')"
              >
                <UInput :model-value="target.slot" size="sm" disabled class="font-mono" />
              </UFormField>
              <UFormField
                :label="t('settingsAutomation.targets.application_label')"
                :hint="t('settingsAutomation.targets.application_hint')"
                required
              >
                <USelect
                  :model-value="target.applicationSlot"
                  :items="applicationItems"
                  size="sm"
                  @update:model-value="(value: string) => setApplication(index, value)"
                />
              </UFormField>
              <UFormField
                :label="t('settingsAutomation.targets.backend_label')"
                :hint="t('settingsAutomation.targets.backend_hint')"
                required
              >
                <USelect
                  :model-value="target.inputBackend"
                  :items="backendItems"
                  size="sm"
                  @update:model-value="(value: InputBackend) => setBackend(index, value)"
                />
              </UFormField>
              <UFormField
                :label="t('settingsAutomation.targets.capture_backend_label')"
                :hint="t('settingsAutomation.targets.capture_backend_hint')"
                required
              >
                <USelect
                  :model-value="target.captureBackend"
                  :items="captureBackendItems"
                  size="sm"
                  @update:model-value="(value: CaptureBackend) => setCaptureBackend(index, value)"
                />
              </UFormField>
            </div>

            <div class="grid gap-4 sm:grid-cols-2">
              <UFormField
                :label="t('settingsAutomation.targets.window_title_label')"
                :hint="t('settingsAutomation.targets.window_title_hint')"
              >
                <UInput v-model="target.windowTitle" size="sm" @change="commit" />
              </UFormField>
              <UFormField
                :label="t('settingsAutomation.targets.window_class_label')"
                :hint="t('settingsAutomation.targets.window_class_hint')"
              >
                <UInput v-model="target.windowClass" size="sm" class="font-mono" @change="commit" />
              </UFormField>
            </div>

            <UFormField
              :label="t('settingsAutomation.targets.timeout_label')"
              :hint="t('settingsAutomation.targets.timeout_hint')"
            >
              <UInputNumber
                v-model="target.resolveTimeoutMilliseconds"
                :min="100"
                :max="10000"
                :step="100"
                size="sm"
                class="w-full sm:w-48"
                @change="commit"
              />
            </UFormField>

            <div class="rounded-lg border border-warning/30 bg-warning/5 p-3">
              <div class="flex flex-wrap items-start gap-3">
                <UIcon
                  name="i-tabler-shield-exclamation"
                  class="mt-0.5 size-4 shrink-0 text-warning"
                  aria-hidden="true"
                />
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <p class="text-xs font-medium text-default">
                      {{ t('settingsAutomation.consent.title') }}
                    </p>
                    <SettingsRestartBadge />
                  </div>
                  <p class="mt-1 text-xs leading-relaxed text-dimmed">
                    {{ t('settingsAutomation.consent.hint') }}
                  </p>
                </div>
                <UButton
                  v-if="target.workflowConsent"
                  size="sm"
                  variant="soft"
                  color="warning"
                  icon="i-tabler-shield-off"
                  :loading="busy[target.slot]"
                  @click="revoke(target)"
                >
                  {{ t('settingsAutomation.consent.revoke') }}
                </UButton>
                <UButton
                  v-else
                  size="sm"
                  variant="soft"
                  color="primary"
                  icon="i-tabler-shield-check"
                  :loading="busy[target.slot]"
                  :disabled="!target.persisted"
                  @click="grant(target)"
                >
                  {{ t('settingsAutomation.consent.grant') }}
                </UButton>
              </div>
            </div>

            <div class="flex items-center border-t border-default/60 pt-4">
              <UButton
                class="ml-auto"
                size="sm"
                variant="ghost"
                color="error"
                icon="i-tabler-trash"
                @click="removeTarget(target)"
              >
                {{ t('settingsAutomation.targets.delete') }}
              </UButton>
            </div>
          </div>
        </article>
      </div>

      <div v-else class="settings-empty-state" role="status">
        <UIcon name="i-tabler-pointer-cog" class="size-6 text-dimmed" aria-hidden="true" />
        <p class="text-sm font-medium text-default">
          {{
            t(
              applications.length
                ? 'settingsAutomation.targets.empty'
                : 'settingsAutomation.targets.no_applications',
            )
          }}
        </p>
        <p class="max-w-md text-center text-xs leading-relaxed text-dimmed">
          {{
            t(
              applications.length
                ? 'settingsAutomation.targets.empty_hint'
                : 'settingsAutomation.targets.no_applications_hint',
            )
          }}
        </p>
        <UButton
          size="sm"
          color="primary"
          variant="soft"
          icon="i-tabler-plus"
          :disabled="applications.length === 0"
          @click="addTarget"
        >
          {{ t('settingsAutomation.targets.add') }}
        </UButton>
      </div>
    </SettingsSection>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type InstalledAutomationTargetProfile } from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import SettingsRestartBadge from '@/components/settings/SettingsRestartBadge.vue'
import SettingsSection from '@/components/settings/SettingsSection.vue'

type InputBackend = InstalledAutomationTargetProfile['inputBackend']
type CaptureBackend = InstalledAutomationTargetProfile['captureBackend']
interface AutomationTargetDraft extends InstalledAutomationTargetProfile {
  persisted: boolean
}

const { t } = useI18n()
const { confirm } = useConfirm()
const store = useSettingsStore()
const applications = computed(() => store.data?.applications.profiles ?? [])
const targets = computed(() => store.data?.automation.win32Targets ?? [])
const draft = ref<AutomationTargetDraft[]>([])
const expandedSlot = ref('')
const busy = reactive<Record<string, boolean>>({})
const applicationItems = computed(() =>
  applications.value.map((application) => ({ label: application.label, value: application.slot })),
)
const backendItems = computed(() => [
  { label: t('settingsAutomation.backend.sendinput'), value: 'sendinput' },
  { label: t('settingsAutomation.backend.postmessage'), value: 'postmessage' },
])
const captureBackendItems = computed(() => [
  { label: t('settingsAutomation.captureBackend.gdi'), value: 'gdi' },
  { label: t('settingsAutomation.captureBackend.wgc'), value: 'wgc' },
])

watch(
  targets,
  () => {
    draft.value = targets.value.map((target) => ({ ...target, persisted: true }))
  },
  { immediate: true },
)

function toggleExpanded(slot: string) {
  expandedSlot.value = expandedSlot.value === slot ? '' : slot
}
function applicationLabel(slot: string) {
  return applications.value.find((application) => application.slot === slot)?.label ?? slot
}
function uniqueSlot(base: string): string {
  const taken = new Set([
    ...draft.value.map((target) => target.slot),
    ...(store.data?.ai.profiles ?? []).map((profile) => profile.slot),
    ...(store.data?.network.httpOrigins ?? []).map((origin) => origin.slot),
    ...(store.data?.applications.profiles ?? []).map((application) => application.slot),
  ])
  if (!taken.has(base)) return base
  for (let index = 2; ; index++) if (!taken.has(`${base}-${index}`)) return `${base}-${index}`
}
function uniqueLabel(base: string): string {
  const taken = new Set(draft.value.map((target) => target.label))
  if (!taken.has(base)) return base
  for (let index = 2; ; index++) if (!taken.has(`${base} ${index}`)) return `${base} ${index}`
}
async function addTarget() {
  const application = applications.value[0]
  if (!application) return
  const slot = uniqueSlot(`${application.slot}-window`)
  draft.value.push({
    slot,
    label: uniqueLabel(t('settingsAutomation.targets.new_label', { name: application.label })),
    applicationSlot: application.slot,
    windowTitle: '',
    windowClass: '',
    inputBackend: 'sendinput',
    captureBackend: 'gdi',
    resolveTimeoutMilliseconds: 3000,
    persisted: false,
  })
  expandedSlot.value = slot
  await commit()
}
function metadata(target: AutomationTargetDraft): InstalledAutomationTargetProfile {
  return {
    slot: target.slot,
    label: target.label.trim(),
    applicationSlot: target.applicationSlot,
    windowTitle: target.windowTitle,
    windowClass: target.windowClass,
    inputBackend: target.inputBackend,
    captureBackend: target.captureBackend,
    resolveTimeoutMilliseconds: target.resolveTimeoutMilliseconds,
    ...(target.workflowConsent ? { workflowConsent: target.workflowConsent } : {}),
  }
}
async function commit(): Promise<boolean> {
  const savable = draft.value.filter(
    (target) =>
      target.label.trim() &&
      target.applicationSlot &&
      applications.value.some((application) => application.slot === target.applicationSlot),
  )
  const ok = await store.patchAutomationTargets(savable.map(metadata))
  if (ok) for (const target of savable) target.persisted = true
  return ok
}
async function setApplication(index: number, value: string) {
  draft.value[index]!.applicationSlot = value
  await commit()
}
async function setBackend(index: number, value: InputBackend) {
  draft.value[index]!.inputBackend = value
  await commit()
}
async function setCaptureBackend(index: number, value: CaptureBackend) {
  draft.value[index]!.captureBackend = value
  await commit()
}
async function grant(target: AutomationTargetDraft) {
  if (!(await commit())) return
  busy[target.slot] = true
  try {
    await backend.automation.grantWorkflowConsent(target.slot)
  } finally {
    busy[target.slot] = false
  }
}
async function revoke(target: AutomationTargetDraft) {
  busy[target.slot] = true
  try {
    await backend.automation.revokeWorkflowConsent(target.slot)
  } finally {
    busy[target.slot] = false
  }
}
async function removeTarget(target: AutomationTargetDraft) {
  if (!target.persisted) {
    draft.value = draft.value.filter((candidate) => candidate.slot !== target.slot)
    return
  }
  const accepted = await confirm({
    title: t('settingsAutomation.confirm.delete_title', { name: target.label }),
    description: t('settingsAutomation.confirm.delete_hint'),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    color: 'error',
  })
  if (accepted === true) {
    await store.patchAutomationTargets(
      draft.value.filter((candidate) => candidate.slot !== target.slot).map(metadata),
    )
  }
}
</script>
