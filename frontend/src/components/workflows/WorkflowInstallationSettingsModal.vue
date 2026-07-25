<template>
  <BaseModal
    :open="open"
    :title="t('workflow.installation.settings_title', { name })"
    icon="i-tabler-adjustments"
    size="3xl"
    :dismissible="!saving"
    @update:open="emit('update:open', $event)"
  >
    <div v-if="loading" class="space-y-3" :aria-label="t('common.loading')">
      <USkeleton v-for="index in 4" :key="index" class="h-24 rounded-lg" />
    </div>

    <div
      v-else-if="failure"
      class="flex items-center gap-3 rounded-lg border border-error/35 bg-error/10 px-4 py-3 text-sm text-error"
      role="alert"
    >
      <span class="min-w-0 flex-1">{{ failure }}</span>
      <UButton size="xs" color="error" variant="soft" @click="load">{{
        t('common.retry')
      }}</UButton>
    </div>

    <div v-else-if="settings" class="space-y-5" data-testid="workflow-installation-settings-body">
      <p class="text-xs text-muted">
        {{ t('workflow.installation.settings_description') }}
      </p>

      <section aria-labelledby="installation-targets-title">
        <div class="mb-2 flex items-center gap-2">
          <UIcon name="i-tabler-device-desktop-cog" class="size-4 text-primary" />
          <h4 id="installation-targets-title" class="text-xs font-semibold text-highlighted">
            {{ t('workflow.installation.targets_title') }}
          </h4>
          <UBadge color="neutral" variant="soft" size="xs">{{ settings.targets.length }}</UBadge>
        </div>

        <div
          v-if="settings.targets.length === 0"
          class="rounded-lg border border-default bg-elevated/20 px-4 py-5 text-center text-xs text-dimmed"
        >
          {{ t('workflow.installation.targets_empty') }}
        </div>

        <div
          v-else
          class="divide-y divide-default overflow-hidden rounded-lg border border-default"
        >
          <article
            v-for="target in settings.targets"
            :key="target.definitionId"
            class="space-y-3 bg-elevated/10 px-4 py-3"
          >
            <div class="flex min-w-0 flex-wrap items-start justify-between gap-2">
              <div class="min-w-0">
                <p class="text-sm font-medium text-highlighted">{{ target.name }}</p>
                <p v-if="target.description" class="mt-0.5 text-xs text-muted">
                  {{ target.description }}
                </p>
              </div>
              <UBadge color="neutral" variant="subtle" size="xs">
                {{ target.targetKind }} / {{ target.adapterKind }} / v{{ target.profileVersion }}
              </UBadge>
            </div>

            <UFormField
              :label="t('workflow.installation.local_target')"
              :error="draftErrors[target.definitionId]?.target"
              required
            >
              <AdaptiveSelect
                v-model="drafts[target.definitionId].targetInstallationId"
                :items="targetItems(target)"
                width-mode="fill"
                :placeholder="t('workflow.installation.select_target')"
              />
            </UFormField>

            <div
              v-if="compatibleTargets(target).length === 0"
              class="flex flex-wrap items-center gap-2 rounded-md border border-warning/30 bg-warning/10 px-3 py-2"
              role="status"
            >
              <UIcon name="i-tabler-alert-triangle" class="size-4 shrink-0 text-warning" />
              <p class="min-w-0 flex-1 text-xs text-warning">
                {{ t('workflow.installation.no_compatible_target') }}
              </p>
              <UButton
                size="xs"
                color="neutral"
                variant="soft"
                icon="i-tabler-settings"
                :label="t('workflow.installation.open_target_settings')"
                @click="openAutomationSettings"
              />
            </div>

            <div v-if="target.discoveryHints.length" class="flex flex-wrap gap-1.5">
              <UBadge
                v-for="hint in target.discoveryHints"
                :key="`${hint.kind}:${hint.value}`"
                color="neutral"
                variant="soft"
                size="xs"
              >
                {{ hint.kind }}: {{ hint.value }}
              </UBadge>
            </div>

            <UFormField
              :label="t('workflow.installation.profile_settings')"
              :description="t('workflow.installation.profile_settings_hint')"
              :error="draftErrors[target.definitionId]?.settings"
            >
              <UTextarea
                v-model="drafts[target.definitionId].settingsJson"
                :rows="5"
                autoresize
                class="w-full font-mono text-xs"
                spellcheck="false"
                @blur="formatSettings(target.definitionId)"
              />
            </UFormField>
          </article>
        </div>
      </section>

      <section aria-labelledby="installation-credentials-title">
        <div class="mb-2 flex items-center gap-2">
          <UIcon name="i-tabler-key" class="size-4 text-primary" />
          <h4 id="installation-credentials-title" class="text-xs font-semibold text-highlighted">
            {{ t('workflow.installation.credentials_title') }}
          </h4>
          <UBadge color="neutral" variant="soft" size="xs">{{
            settings.credentials.length
          }}</UBadge>
        </div>

        <p
          v-if="settings.credentials.length === 0"
          class="rounded-lg border border-default bg-elevated/20 px-4 py-5 text-center text-xs text-dimmed"
        >
          {{ t('workflow.installation.credentials_empty') }}
        </p>
        <div
          v-else
          class="divide-y divide-default overflow-hidden rounded-lg border border-default"
        >
          <div
            v-for="credential in settings.credentials"
            :key="credential.slot"
            class="flex items-start gap-3 bg-elevated/10 px-4 py-3"
          >
            <UIcon name="i-tabler-lock" class="mt-0.5 size-4 shrink-0 text-dimmed" />
            <div class="min-w-0 flex-1">
              <p class="text-xs font-medium text-highlighted">{{ credential.purpose }}</p>
              <p class="mt-0.5 truncate font-mono text-[10px] text-dimmed">
                {{ credential.slot }} · {{ credential.kind }}
              </p>
            </div>
            <UBadge :color="credential.bindingId ? 'success' : 'warning'" variant="soft" size="xs">
              {{
                t(
                  credential.bindingId
                    ? 'workflow.installation.credential_bound'
                    : 'workflow.installation.credential_unbound',
                )
              }}
            </UBadge>
          </div>
          <p class="bg-elevated/10 px-4 py-2 text-[11px] text-dimmed">
            {{ t('workflow.installation.credentials_pending_hint') }}
          </p>
        </div>
      </section>

      <p
        v-if="saveFailure"
        class="rounded-md border border-error/35 bg-error/10 px-3 py-2 text-xs text-error"
        role="alert"
      >
        {{ saveFailure }}
      </p>
    </div>

    <template #footer>
      <UButton
        color="neutral"
        variant="ghost"
        :disabled="saving"
        @click="emit('update:open', false)"
      >
        {{ t('common.cancel') }}
      </UButton>
      <UButton
        icon="i-tabler-device-floppy"
        :label="t('common.save')"
        :loading="saving"
        :disabled="loading || !settings || hasErrors || !hasChanges"
        @click="save"
      />
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { workflowTransport, type InstallationSettingsView } from '@/app/transport/workflow'
import type { InstalledAutomationTargetProfile } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import { useSettingsStore } from '@/stores/settings'
import {
  compatibleAutomationTargets,
  profileSettingsIssue,
} from '@/lib/workflowInstallationSettings'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'
import BaseModal from '@/components/common/BaseModal.vue'

type TargetView = InstallationSettingsView['targets'][number]
type Draft = { targetInstallationId: string; settingsJson: string }
type DraftError = { target?: string; settings?: string }

const props = defineProps<{ open: boolean; installationId: string; name: string }>()
const emit = defineEmits<{
  'update:open': [value: boolean]
  saved: []
}>()
const { t } = useI18n()
const router = useRouter()
const settingsStore = useSettingsStore()
const settings = ref<InstallationSettingsView | null>(null)
const loading = ref(false)
const saving = ref(false)
const failure = ref('')
const saveFailure = ref('')
const drafts = reactive<Record<string, Draft>>({})

const draftErrors = computed<Record<string, DraftError>>(() => {
  const result: Record<string, DraftError> = {}
  for (const target of settings.value?.targets ?? []) {
    const draft = drafts[target.definitionId]
    const error: DraftError = {}
    if (!draft?.targetInstallationId) {
      error.target = t('workflow.installation.target_required')
    }
    const settingsIssue = profileSettingsIssue(draft?.settingsJson ?? '')
    if (settingsIssue === 'invalid-json') {
      error.settings = t('workflow.installation.settings_json_invalid')
    } else if (settingsIssue === 'object-required') {
      error.settings = t('workflow.installation.settings_object_required')
    }
    result[target.definitionId] = error
  }
  return result
})
const hasErrors = computed(() =>
  Object.values(draftErrors.value).some((error) => error.target || error.settings),
)
const hasChanges = computed(() =>
  Boolean(
    settings.value?.targets.some((target) => {
      const draft = drafts[target.definitionId]
      return draft ? targetChanged(target, draft) : false
    }),
  ),
)

watch(
  () => [props.open, props.installationId] as const,
  ([open]) => {
    if (open && props.installationId) void load()
  },
  { immediate: true },
)

async function load(): Promise<void> {
  loading.value = true
  failure.value = ''
  saveFailure.value = ''
  try {
    const [snapshot] = await Promise.all([
      workflowTransport.getInstallationSettings(props.installationId),
      settingsStore.loaded ? Promise.resolve() : settingsStore.load(),
    ])
    settings.value = snapshot
    for (const target of snapshot.targets) {
      drafts[target.definitionId] = {
        targetInstallationId: target.targetInstallationId,
        settingsJson: prettyJSON(target.settingsJson),
      }
    }
  } catch (error) {
    failure.value = errorMessage(error)
  } finally {
    loading.value = false
  }
}

function compatibleTargets(target: TargetView): InstalledAutomationTargetProfile[] {
  return compatibleAutomationTargets(settingsStore.data?.automation.targets ?? [], target)
}

function targetItems(target: TargetView): Array<{ label: string; value: string }> {
  const result = compatibleTargets(target).map((candidate) => ({
    label: candidate.workflowConsent
      ? candidate.label
      : `${candidate.label} (${t('workflow.installation.target_not_authorized')})`,
    value: `automation-target/${candidate.slot}`,
  }))
  const current = drafts[target.definitionId]?.targetInstallationId
  if (current && !result.some((item) => item.value === current)) {
    result.unshift({
      label: `${current} (${t('workflow.installation.target_unavailable')})`,
      value: current,
    })
  }
  return result
}

function formatSettings(definitionId: string): void {
  const draft = drafts[definitionId]
  if (!draft) return
  try {
    draft.settingsJson = JSON.stringify(JSON.parse(draft.settingsJson), null, 2)
  } catch {
    // Keep invalid input in place so the user can repair it.
  }
}

async function save(): Promise<void> {
  if (!settings.value || hasErrors.value) return
  saving.value = true
  saveFailure.value = ''
  try {
    let current = settings.value
    for (const target of current.targets) {
      const draft = drafts[target.definitionId]
      const currentTarget =
        current.targets.find((candidate) => candidate.definitionId === target.definitionId) ??
        target
      if (!targetChanged(currentTarget, draft)) continue
      current = await workflowTransport.updateInstallationTargetProfile(
        props.installationId,
        current.generation,
        target.definitionId,
        draft.settingsJson,
        draft.targetInstallationId,
      )
      settings.value = current
    }
    emit('saved')
    emit('update:open', false)
  } catch (error) {
    saveFailure.value = errorMessage(error)
  } finally {
    saving.value = false
  }
}

function targetChanged(target: TargetView, draft: Draft): boolean {
  return (
    target.targetInstallationId !== draft.targetInstallationId ||
    normalizedJSON(target.settingsJson) !== normalizedJSON(draft.settingsJson)
  )
}

function openAutomationSettings(): void {
  emit('update:open', false)
  void router.push({ path: '/settings', query: { section: 'automation' } })
}

function prettyJSON(value: string): string {
  try {
    return JSON.stringify(JSON.parse(value), null, 2)
  } catch {
    return value
  }
}

function normalizedJSON(value: string): string {
  try {
    return JSON.stringify(JSON.parse(value))
  } catch {
    return value
  }
}
</script>
