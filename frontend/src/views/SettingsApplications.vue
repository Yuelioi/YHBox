<template>
  <div class="settings-page">
    <SettingsSection :title="t('settingsApplications.profiles.title')" icon="i-tabler-apps">
      <template #badge>
        <UBadge size="xs" color="neutral" variant="subtle">{{ draft.length }}</UBadge>
      </template>
      <template #actions>
        <UButton
          size="sm"
          color="primary"
          variant="soft"
          icon="i-tabler-folder-open"
          :loading="picking"
          @click="addApplication"
        >
          {{ t('settingsApplications.profiles.add') }}
        </UButton>
      </template>

      <div
        v-if="pickerFailure"
        class="rounded-lg border border-error/30 bg-error/10 px-3 py-2 text-xs text-error"
        role="alert"
      >
        {{ pickerFailure }}
      </div>

      <div v-if="draft.length" class="settings-collection">
        <article v-for="profile in draft" :key="profile.slot" class="ai-profile">
          <div class="settings-entity-summary" @dblclick.prevent="toggleExpanded(profile.slot)">
            <span
              class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-default bg-elevated"
            >
              <UIcon name="i-tabler-app-window" class="size-4 text-toned" />
            </span>
            <span class="min-w-0 flex-1">
              <span class="flex flex-wrap items-center gap-2">
                <span class="truncate text-sm font-medium text-default">{{ profile.label }}</span>
              </span>
              <span class="mt-1 block truncate text-xs text-dimmed">
                {{ fileName(profile.executable) }} · <code>{{ profile.slot }}</code>
              </span>
            </span>
            <div class="shrink-0" @dblclick.stop>
              <UButton
                size="xs"
                color="neutral"
                :variant="expandedSlot === profile.slot ? 'soft' : 'ghost'"
                :icon="expandedSlot === profile.slot ? 'i-tabler-chevron-up' : 'i-tabler-edit'"
                :label="t(expandedSlot === profile.slot ? 'common.close' : 'common.edit')"
                :aria-expanded="expandedSlot === profile.slot"
                :aria-controls="`application-profile-${profile.slot}`"
                @click="toggleExpanded(profile.slot)"
              />
            </div>
          </div>

          <div
            v-if="expandedSlot === profile.slot"
            :id="`application-profile-${profile.slot}`"
            class="ai-profile__details"
          >
            <div class="grid gap-4 sm:grid-cols-2">
              <UFormField :label="t('settingsApplications.profiles.name_label')" required>
                <UInput v-model="profile.label" size="sm" @change="commit" />
              </UFormField>
              <UFormField
                :label="t('settingsApplications.profiles.slot_label')"
                :hint="t('settingsApplications.profiles.slot_hint')"
              >
                <UInput :model-value="profile.slot" size="sm" disabled class="font-mono" />
              </UFormField>
            </div>

            <UFormField :label="t('settingsApplications.profiles.executable_label')">
              <div class="flex gap-2">
                <UInput
                  :model-value="profile.executable"
                  size="sm"
                  disabled
                  class="min-w-0 flex-1 font-mono"
                />
                <UButton
                  size="sm"
                  color="neutral"
                  variant="soft"
                  icon="i-tabler-refresh"
                  :loading="busy[profile.slot]"
                  @click="replaceExecutable(profile)"
                >
                  {{ t('settingsApplications.profiles.replace') }}
                </UButton>
              </div>
            </UFormField>

            <UFormField
              :label="t('settingsApplications.profiles.arguments_label')"
              :hint="t('settingsApplications.profiles.arguments_hint')"
            >
              <UTextarea
                v-model="profile.argumentLines"
                :rows="3"
                class="font-mono"
                :placeholder="t('settingsApplications.profiles.arguments_placeholder')"
                @change="commit"
              />
            </UFormField>

            <div class="flex items-center border-t border-default/60 pt-4">
              <UButton
                class="ml-auto"
                size="sm"
                variant="ghost"
                color="error"
                icon="i-tabler-trash"
                @click="removeApplication(profile)"
              >
                {{ t('settingsApplications.profiles.delete') }}
              </UButton>
            </div>
          </div>
        </article>
      </div>

      <div v-else class="settings-empty-state" role="status">
        <UIcon name="i-tabler-apps" class="size-6 text-dimmed" />
        <p class="text-sm font-medium text-default">
          {{
            t(
              pickerCancelled
                ? 'settingsApplications.profiles.cancelled'
                : 'settingsApplications.profiles.empty',
            )
          }}
        </p>
        <p class="max-w-md text-center text-xs leading-relaxed text-dimmed">
          {{
            t(
              pickerCancelled
                ? 'settingsApplications.profiles.cancelled_hint'
                : 'settingsApplications.profiles.empty_hint',
            )
          }}
        </p>
        <UButton
          size="sm"
          color="primary"
          variant="soft"
          icon="i-tabler-folder-open"
          :loading="picking"
          @click="addApplication"
        >
          {{ t('settingsApplications.profiles.add') }}
        </UButton>
      </div>
    </SettingsSection>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type InstalledApplicationProfile } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import SettingsSection from '@/components/settings/SettingsSection.vue'

interface ApplicationDraft extends InstalledApplicationProfile {
  persisted: boolean
  argumentLines: string
}

const { t } = useI18n()
const { confirm } = useConfirm()
const store = useSettingsStore()
const profiles = computed(() => store.data?.applications.profiles ?? [])
const draft = ref<ApplicationDraft[]>([])
const expandedSlot = ref('')
const busy = reactive<Record<string, boolean>>({})
const picking = ref(false)
const pickerCancelled = ref(false)
const pickerFailure = ref('')

watch(
  profiles,
  () => {
    draft.value = profiles.value.map((profile) => ({
      ...profile,
      arguments: [...profile.arguments],
      argumentLines: profile.arguments.join('\n'),
      persisted: true,
    }))
  },
  { immediate: true },
)

function toggleExpanded(slot: string) {
  expandedSlot.value = expandedSlot.value === slot ? '' : slot
}
function fileName(path: string) {
  return path.split(/[\\/]/).pop() || path
}
function slug(value: string) {
  return (
    value
      .toLocaleLowerCase()
      .replace(/\.exe$/i, '')
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-|-$/g, '') || 'application'
  )
}
function uniqueSlot(base: string): string {
  const taken = new Set([
    ...draft.value.map((profile) => profile.slot),
    ...(store.data?.ai.profiles ?? []).map((profile) => profile.slot),
    ...(store.data?.network.httpOrigins ?? []).map((origin) => origin.slot),
    ...(store.data?.automation.targets ?? []).map((target) => target.slot),
  ])
  if (!taken.has(base)) return base
  for (let index = 2; ; index++) if (!taken.has(`${base}-${index}`)) return `${base}-${index}`
}
function uniqueLabel(base: string): string {
  const taken = new Set(draft.value.map((profile) => profile.label))
  if (!taken.has(base)) return base
  for (let index = 2; ; index++) if (!taken.has(`${base} ${index}`)) return `${base} ${index}`
}
async function pickExecutable(): Promise<string | null> {
  const path = await backend.applications.pickExecutable(t('settingsApplications.picker.title'))
  if (!path) {
    pickerCancelled.value = true
    return null
  }
  return path
}
async function addApplication() {
  if (picking.value) return
  picking.value = true
  pickerCancelled.value = false
  pickerFailure.value = ''
  try {
    const executable = await pickExecutable()
    if (!executable) return
    const name = fileName(executable).replace(/\.exe$/i, '')
    const slot = uniqueSlot(slug(name))
    draft.value.push({
      slot,
      label: uniqueLabel(name),
      executable,
      arguments: [],
      argumentLines: '',
      persisted: false,
    })
    expandedSlot.value = slot
    await commit()
  } catch (error) {
    pickerFailure.value = errorMessage(error)
  } finally {
    picking.value = false
  }
}
function metadata(profile: ApplicationDraft): InstalledApplicationProfile {
  return {
    slot: profile.slot,
    label: profile.label.trim(),
    executable: profile.executable,
    arguments: profile.argumentLines
      .split('\n')
      .map((value) => value.replace(/\r$/, ''))
      .filter((value) => value.length > 0),
  }
}
async function commit() {
  const savable = draft.value.filter((profile) => profile.label.trim() && profile.executable)
  const ok = await store.patchApplicationProfiles(savable.map(metadata))
  if (ok) for (const profile of savable) profile.persisted = true
  return ok
}
async function replaceExecutable(profile: ApplicationDraft) {
  if (busy[profile.slot]) return
  busy[profile.slot] = true
  pickerCancelled.value = false
  pickerFailure.value = ''
  try {
    const executable = await pickExecutable()
    if (executable) {
      profile.executable = executable
      await commit()
    }
  } catch (error) {
    pickerFailure.value = errorMessage(error)
  } finally {
    busy[profile.slot] = false
  }
}
async function removeApplication(profile: ApplicationDraft) {
  const accepted = await confirm({
    title: t('settingsApplications.confirm.delete_title', { name: profile.label }),
    description: t('settingsApplications.confirm.delete_hint'),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    color: 'error',
  })
  if (accepted === true)
    await store.patchApplicationProfiles(
      draft.value.filter((candidate) => candidate.slot !== profile.slot).map(metadata),
    )
}
</script>
