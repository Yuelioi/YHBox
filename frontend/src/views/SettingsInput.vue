<template>
  <div class="settings-page">
    <SettingsSection
      :title="t('settings.input.record.title')"
      :description="t('settings.input.record.hint')"
      icon="i-tabler-player-record"
    >
      <SettingsRow :label="t('settings.input.record.mouse_mode_label')" :hint="mouseModeHint">
        <template #meta><SettingsRestartBadge /></template>
        <USelect
          :model-value="settings?.ui.recordingMouseMode ?? 'relative'"
          :items="mouseModeItems"
          class="w-56"
          :aria-label="t('settings.input.record.mouse_mode_label')"
          @update:model-value="(value: string) => patchRecord({ recordingMouseMode: value })"
        />
      </SettingsRow>
    </SettingsSection>

    <SettingsSection
      :title="t('settings.input.counts.title')"
      :description="t('settings.input.counts.commercial_hint')"
      icon="i-tabler-target-arrow"
    >
      <template #actions>
        <UButton size="sm" color="primary" variant="soft" icon="i-tabler-plus" @click="addProfile">
          {{ t('settings.input.counts.add_profile') }}
        </UButton>
      </template>

      <div v-if="draftProfiles.length" class="space-y-3">
        <article
          v-for="(profile, index) in draftProfiles"
          :key="profile.localID"
          class="calibration-profile"
          :class="profile.label === activeLabel ? 'calibration-profile--active' : ''"
        >
          <div class="flex min-w-0 flex-1 items-start gap-3">
            <div
              class="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-lg border border-default bg-elevated/50"
            >
              <UIcon name="i-tabler-device-gamepad-2" class="size-4 text-toned" />
            </div>
            <div class="min-w-0 flex-1 space-y-3">
              <div class="flex flex-wrap items-center gap-2">
                <UInput
                  v-model="profile.label"
                  size="sm"
                  class="min-w-48 flex-1"
                  :placeholder="t('settings.input.counts.label_placeholder')"
                  :aria-label="t('settings.input.counts.col_label')"
                  @change="commitProfile(index)"
                />
                <UBadge
                  v-if="profile.label === activeLabel"
                  color="primary"
                  variant="subtle"
                  size="xs"
                  icon="i-tabler-star-filled"
                >
                  {{ t('settings.input.counts.active_badge') }}
                </UBadge>
                <UBadge
                  :color="profile.counts360 > 0 ? 'success' : 'warning'"
                  variant="subtle"
                  size="xs"
                >
                  {{
                    profile.counts360 > 0
                      ? t('settings.input.counts.calibrated', { n: profile.counts360 })
                      : t('settings.input.counts.uncalibrated')
                  }}
                </UBadge>
              </div>

              <p v-if="profileErrors[profile.localID]" class="text-xs text-error" role="alert">
                {{ profileErrors[profile.localID] }}
              </p>

              <div class="flex flex-wrap items-end gap-3">
                <UFormField
                  :label="t('settings.input.counts.advanced_value')"
                  :hint="t('settings.input.counts.advanced_value_hint')"
                  class="min-w-44"
                >
                  <UInputNumber
                    v-model="profile.counts360"
                    :min="0"
                    :max="999999"
                    :step="100"
                    size="sm"
                    class="w-40"
                    :aria-label="t('settings.input.counts.col_counts')"
                    @change="commitProfile(index)"
                  />
                </UFormField>
                <div class="ml-auto flex flex-wrap items-center gap-2">
                  <UButton
                    v-if="profile.label !== activeLabel"
                    size="xs"
                    variant="ghost"
                    color="neutral"
                    icon="i-tabler-star"
                    @click="setActive(profile.label)"
                  >
                    {{ t('settings.input.counts.make_default') }}
                  </UButton>
                  <UButton
                    size="sm"
                    variant="soft"
                    color="primary"
                    icon="i-tabler-target"
                    @click="openCalibratorFor(index)"
                  >
                    {{
                      profile.counts360 > 0
                        ? t('settings.input.counts.recalibrate')
                        : t('settings.input.counts.start_calibration')
                    }}
                  </UButton>
                  <UButton
                    size="xs"
                    variant="ghost"
                    color="error"
                    icon="i-tabler-trash"
                    :title="t('settings.input.counts.delete_profile')"
                    :aria-label="t('settings.input.counts.delete_profile')"
                    @click="removeProfile(index)"
                  />
                </div>
              </div>
            </div>
          </div>
        </article>
      </div>

      <div v-else class="settings-empty-state">
        <UIcon name="i-tabler-target-off" class="size-6 text-dimmed" aria-hidden="true" />
        <div>
          <p class="text-sm font-medium text-default">{{ t('settings.input.counts.empty') }}</p>
          <p class="mt-1 text-xs text-dimmed">{{ t('settings.input.counts.empty_hint') }}</p>
        </div>
        <UButton size="sm" variant="soft" color="primary" icon="i-tabler-plus" @click="addProfile">
          {{ t('settings.input.counts.add_profile') }}
        </UButton>
      </div>

      <div class="rounded-lg border border-default bg-elevated/20 p-3">
        <div class="flex items-start gap-2">
          <UIcon name="i-tabler-info-circle" class="mt-0.5 size-4 shrink-0 text-primary" />
          <div class="min-w-0 flex-1">
            <p class="text-xs font-medium text-default">{{ t('settings.input.howto.title') }}</p>
            <p class="mt-1 text-xs leading-relaxed text-dimmed">
              {{
                t('settings.input.howto.compact', {
                  hk: hotkeys.keyFor('system.calibrate-toggle', 'F8'),
                })
              }}
            </p>
          </div>
        </div>
      </div>

      <div
        v-if="activeCounts > 0"
        class="flex flex-wrap items-center gap-3 border-t border-default/60 pt-4"
      >
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-default">{{ t('settings.input.counts.sync_all') }}</p>
          <p class="mt-1 text-xs text-dimmed">{{ t('settings.input.counts.sync_all_hint') }}</p>
        </div>
        <UButton
          size="sm"
          variant="outline"
          color="neutral"
          icon="i-tabler-refresh"
          @click="onSyncAll"
        >
          {{ t('settings.input.counts.sync_action') }}
        </UButton>
      </div>
    </SettingsSection>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend } from '@/lib/backend'
import { useSettingsStore, type MouseProfile } from '@/stores/settings'
import { useHotkeysStore } from '@/stores/hotkeys'
import { useToast } from '@nuxt/ui/composables'
import { useConfirm } from '@/composables/useConfirm'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import SettingsRestartBadge from '@/components/settings/SettingsRestartBadge.vue'
import SettingsRow from '@/components/settings/SettingsRow.vue'
import SettingsSection from '@/components/settings/SettingsSection.vue'

interface DraftMouseProfile extends MouseProfile {
  localID: string
}

const { t } = useI18n()
const { confirm } = useConfirm()
const hotkeys = useHotkeysStore()
const settingsStore = useSettingsStore()
const settings = computed(() => settingsStore.data)
const toast = useToast()
const draftProfiles = ref<DraftMouseProfile[]>([])
const profileErrors = reactive<Record<string, string>>({})

watch(
  () => settingsStore.mouseProfiles,
  (profiles) => {
    draftProfiles.value = profiles.map((profile) => ({
      ...profile,
      localID: crypto.randomUUID(),
    }))
  },
  { immediate: true, deep: true },
)

const activeLabel = computed(() => settings.value?.ui.activeMouseProfile ?? '')
const activeCounts = computed(() => settingsStore.activeMouseCounts360)
const mouseModeItems = computed(() => [
  { label: t('settings.input.record.mouse_mode.relative'), value: 'relative' },
  { label: t('settings.input.record.mouse_mode.absolute'), value: 'absolute' },
])
const mouseModeHint = computed(() =>
  t(
    `settings.input.record.mouse_mode_detail.${settings.value?.ui.recordingMouseMode ?? 'relative'}`,
  ),
)

function plainProfiles(): MouseProfile[] {
  return draftProfiles.value.map(({ label, counts360 }) => ({ label, counts360 }))
}

async function patchProfiles(list: MouseProfile[], active?: string) {
  const ui: Record<string, unknown> = { mouseProfiles: list }
  if (active !== undefined) ui.activeMouseProfile = active
  await settingsStore.patch({ ui })
}

function uniqueLabel(base: string): string {
  const taken = new Set(draftProfiles.value.map((profile) => profile.label.trim()))
  if (!taken.has(base)) return base
  for (let index = 2; ; index++) {
    const candidate = `${base} ${index}`
    if (!taken.has(candidate)) return candidate
  }
}

async function addProfile() {
  const label = uniqueLabel(t('settings.input.counts.new_profile_label'))
  const list = [...plainProfiles(), { label, counts360: 0 }]
  await patchProfiles(list, draftProfiles.value.length === 0 ? label : undefined)
}

async function removeProfile(index: number) {
  const profile = draftProfiles.value[index]
  if (!profile) return
  const ok = await confirm({
    title: t('settings.input.confirm.delete_profile_title', { name: profile.label }),
    description: t('settings.input.confirm.delete_profile_desc'),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    color: 'error',
  })
  if (ok !== true) return
  const list = plainProfiles().filter((_, itemIndex) => itemIndex !== index)
  const active = profile.label === activeLabel.value ? (list[0]?.label ?? '') : undefined
  await patchProfiles(list, active)
}

async function setActive(label: string) {
  if (!label) return
  await patchProfiles(plainProfiles(), label)
}

async function commitProfile(index: number) {
  const profile = draftProfiles.value[index]
  const original = settingsStore.mouseProfiles[index]
  if (!profile || !original) return
  const label = profile.label.trim()
  if (!label) {
    profileErrors[profile.localID] = t('settings.input.validation.label_required')
    return
  }
  const duplicate = draftProfiles.value.some(
    (other, otherIndex) => otherIndex !== index && other.label.trim() === label,
  )
  if (duplicate) {
    profileErrors[profile.localID] = t('settings.input.validation.label_duplicate')
    return
  }
  delete profileErrors[profile.localID]
  profile.label = label
  profile.counts360 = Math.max(0, Math.floor(Number(profile.counts360) || 0))
  const active = original.label === activeLabel.value ? label : undefined
  await patchProfiles(plainProfiles(), active)
}

function patchRecord(patch: Record<string, unknown>) {
  void settingsStore.patch({ ui: patch })
}

async function onSyncAll() {
  const current = activeCounts.value
  if (current <= 0) return
  const yes = await confirm({
    title: t('settings.input.confirm.sync_title'),
    description: t('settings.input.confirm.sync_desc', { cur: current }),
    confirmText: t('settings.input.confirm.sync_confirm'),
    color: 'primary',
  })
  if (yes !== true) return
  const result = await backend.containers.syncLocalMouseCalibration(current)
  if (result) {
    toast.add({
      title: t('settings.input.toast.synced_title', { n: result.updated?.length ?? 0 }),
      description: result.skipped?.length
        ? t('settings.input.toast.synced_skipped', { n: result.skipped.length })
        : undefined,
      color: 'success',
    })
  }
}

async function openCalibratorFor(index: number) {
  if (!draftProfiles.value[index]) return
  const id = `calib-${crypto.randomUUID()}`
  const ok = await backend.tools.openCalibratorHUD(id)
  if (!ok) return
  const result = await awaitWailsEvent<{ id: string; counts?: number; cancelled?: boolean }>(
    'calibration:result',
    (payload) => payload?.id === id,
  )
  if (!result.cancelled && typeof result.counts === 'number' && result.counts > 0) {
    draftProfiles.value[index].counts360 = result.counts
    await commitProfile(index)
  }
}
</script>
