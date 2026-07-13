<template>
  <div class="settings-page">
    <!-- 总说明 -->
    <section class="settings-section">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-mouse" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">{{ t('settings.input.title') }}</h2>
      </div>
      <p class="text-xs text-dimmed">{{ t('settings.input.intro') }}</p>
      <div
        class="rounded-md bg-elevated/30 border border-default/60 px-3 py-2 text-xs text-muted leading-relaxed"
      >
        <UIcon
          name="i-tabler-info-circle"
          class="size-3.5 inline-block align-middle mr-1 text-warning/80"
        />
        <span class="text-toned">{{ t('settings.input.intro_box.what_label') }}</span
        >: {{ t('settings.input.intro_box.what_desc') }}
        <ul class="list-disc pl-5 mt-1 space-y-0.5">
          <li>{{ t('settings.input.intro_box.item_default_source') }}</li>
          <li>{{ t('settings.input.intro_box.item_sync_action') }}</li>
        </ul>
        <span class="mt-1 block">
          {{ t('settings.input.intro_box.footnote_prefix')
          }}<span class="text-warning">{{ t('settings.input.intro_box.footnote_negation') }}</span
          >{{ t('settings.input.intro_box.footnote_rest') }}
        </span>
      </div>
    </section>

    <!-- 录制配置 -->
    <section class="settings-section">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-player-record" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">{{ t('settings.input.record.title') }}</h2>
      </div>
      <p class="text-xs text-dimmed">{{ t('settings.input.record.hint') }}</p>

      <div class="border-t border-default/60" />

      <!-- 鼠标语义 -->
      <div class="flex items-center justify-between gap-6">
        <div class="min-w-0">
          <div class="text-sm text-default">{{ t('settings.input.record.mouse_mode_label') }}</div>
          <p class="text-xs text-dimmed mt-0.5">{{ t('settings.input.record.mouse_mode_hint') }}</p>
        </div>
        <USelect
          :model-value="settings?.ui.recordingMouseMode ?? 'relative'"
          :items="mouseModeItems"
          class="w-56"
          :aria-label="t('settings.input.record.mouse_mode_label')"
          @update:model-value="(v: string) => patchRecord({ recordingMouseMode: v })"
        />
      </div>
    </section>

    <section class="settings-section">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-target" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">{{ t('settings.input.counts.title') }}</h2>
      </div>
      <p class="text-xs text-dimmed">{{ t('settings.input.counts.hint') }}</p>

      <!-- profile 列表：每行 = 默认单选 + 名字 + counts + 校准 + 删除 -->
      <div v-if="profiles.length" class="space-y-2">
        <div class="flex items-center gap-2 text-xs uppercase tracking-wider text-dimmed px-3">
          <span class="w-8 text-center">{{ t('settings.input.counts.col_active') }}</span>
          <span class="flex-1">{{ t('settings.input.counts.col_label') }}</span>
          <span class="w-28">{{ t('settings.input.counts.col_counts') }}</span>
          <span class="w-[88px]" />
        </div>
        <div
          v-for="(p, i) in profiles"
          :key="i"
          class="flex items-center gap-2 rounded-md border px-3 py-2"
          :class="
            p.label === activeLabel
              ? 'border-primary/50 bg-primary/5'
              : 'border-default/60 bg-elevated/30'
          "
        >
          <div class="w-8 flex justify-center">
            <URadio
              :model-value="activeLabel"
              :value="p.label"
              :disabled="!p.label"
              :aria-label="t('settings.input.counts.set_active', { name: p.label })"
              @update:model-value="() => setActive(p.label)"
            />
          </div>
          <UInput
            :model-value="p.label"
            size="sm"
            class="flex-1 min-w-0"
            :placeholder="t('settings.input.counts.label_placeholder')"
            :aria-label="t('settings.input.counts.col_label')"
            @update:model-value="(v: string) => updateLabel(i, v)"
          />
          <div class="w-28 shrink-0">
            <UInputNumber
              :model-value="p.counts360"
              :min="0"
              :max="999999"
              :step="100"
              size="sm"
              class="w-full"
              :aria-label="t('settings.input.counts.col_counts')"
              @update:model-value="(v: number) => updateCounts(i, v)"
            />
          </div>
          <div class="w-[88px] flex gap-1 justify-end">
            <UButton
              size="xs"
              variant="soft"
              color="primary"
              icon="i-tabler-target"
              :title="t('settings.input.counts.recalibrate')"
              :aria-label="t('settings.input.counts.recalibrate')"
              @click="openCalibratorFor(i)"
            />
            <UButton
              size="xs"
              variant="ghost"
              color="error"
              icon="i-tabler-trash"
              :title="t('settings.input.counts.delete_profile')"
              :aria-label="t('settings.input.counts.delete_profile')"
              @click="removeProfile(i)"
            />
          </div>
        </div>
      </div>
      <p v-else class="text-xs text-dimmed italic">{{ t('settings.input.counts.empty') }}</p>

      <div class="flex items-center gap-2 flex-wrap">
        <UButton size="sm" color="primary" variant="soft" icon="i-tabler-plus" @click="addProfile">
          {{ t('settings.input.counts.add_profile') }}
        </UButton>
        <UButton
          v-if="activeCounts > 0"
          size="sm"
          variant="soft"
          color="primary"
          icon="i-tabler-refresh"
          @click="onSyncAll"
        >
          {{ t('settings.input.counts.sync_all') }}
        </UButton>
        <span class="ml-auto text-xs text-dimmed">
          {{ t('settings.input.counts.share_hint') }}
        </span>
      </div>
    </section>

    <!-- 说明 -->
    <section class="settings-section">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-list-numbers" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">{{ t('settings.input.howto.title') }}</h2>
      </div>
      <ol class="list-decimal pl-5 space-y-1 text-xs text-dimmed">
        <li>{{ t('settings.input.howto.step_open') }}</li>
        <li>{{ t('settings.input.howto.step_focus') }}</li>
        <li>
          {{
            t('settings.input.howto.step_start', {
              hk: hotkeys.keyFor('system.calibrate-toggle', 'F8'),
            })
          }}
        </li>
        <li>{{ t('settings.input.howto.step_spin') }}</li>
        <li>
          {{
            t('settings.input.howto.step_stop', {
              hk: hotkeys.keyFor('system.calibrate-toggle', 'F8'),
            })
          }}
        </li>
        <li>{{ t('settings.input.howto.step_save') }}</li>
      </ol>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend } from '@/lib/backend'
import { useSettingsStore, type MouseProfile } from '@/stores/settings'
import { useHotkeysStore } from '@/stores/hotkeys'
import { useToast } from '@nuxt/ui/composables'
import { useConfirm } from '@/composables/useConfirm'
import { awaitWailsEvent } from '@/composables/useWailsEvent'

const { t } = useI18n()
const { confirm } = useConfirm()
const hotkeys = useHotkeysStore()

const settingsStore = useSettingsStore()
const settings = computed(() => settingsStore.data)
const toast = useToast()

const mouseModeItems = computed(() => [
  { label: t('settings.input.record.mouse_mode.relative'), value: 'relative' },
  { label: t('settings.input.record.mouse_mode.absolute'), value: 'absolute' },
])

// ─── 鼠标校准 profile 列表 ───────────────────────────────────────────────
const profiles = computed<MouseProfile[]>(() => settingsStore.mouseProfiles)
const activeLabel = computed(() => settings.value?.ui.activeMouseProfile ?? '')
const activeCounts = computed(() => settingsStore.activeMouseCounts360)

// profiles 是数组, RFC7386 整体替换 — 每次改都提交全新列表 (+ 可选改 active label)。
async function patchProfiles(list: MouseProfile[], active?: string) {
  const ui: Record<string, any> = { mouseProfiles: list }
  if (active !== undefined) ui.activeMouseProfile = active
  await settingsStore.patch({ ui })
}

function uniqueLabel(base: string): string {
  const taken = new Set(profiles.value.map((p) => p.label))
  if (!taken.has(base)) return base
  for (let i = 2; ; i++) {
    const cand = `${base} ${i}`
    if (!taken.has(cand)) return cand
  }
}

async function addProfile() {
  const label = uniqueLabel(t('settings.input.counts.new_profile_label'))
  const list = [...profiles.value, { label, counts360: 0 }]
  // 列表本来空 → 新档自动设为 active。
  await patchProfiles(list, profiles.value.length === 0 ? label : undefined)
}

async function removeProfile(i: number) {
  const removed = profiles.value[i]
  const list = profiles.value.filter((_, idx) => idx !== i)
  // 删的是当前 active → active 落到剩下第一个 (没了就清空)。
  const active = removed.label === activeLabel.value ? (list[0]?.label ?? '') : undefined
  await patchProfiles(list, active)
}

async function setActive(label: string) {
  if (!label) return
  await patchProfiles(profiles.value, label)
}

async function updateLabel(i: number, v: string) {
  const old = profiles.value[i]
  if (!old || v === old.label) return
  const list = profiles.value.map((p, idx) => (idx === i ? { ...p, label: v } : p))
  // 改的是 active 那行 → active 引用跟着改名。
  const active = old.label === activeLabel.value ? v : undefined
  await patchProfiles(list, active)
}

async function updateCounts(i: number, v: number) {
  const n = Number(v)
  if (!Number.isFinite(n) || n < 0) return
  const list = profiles.value.map((p, idx) => (idx === i ? { ...p, counts360: Math.floor(n) } : p))
  await patchProfiles(list)
}

async function patchRecord(patch: Record<string, any>) {
  if (!settings.value) return
  await settingsStore.patch({ ui: patch })
}

async function onSyncAll() {
  const cur = activeCounts.value
  if (cur <= 0) {
    toast.add({ title: t('settings.input.toast.counts_not_set'), color: 'warning' })
    return
  }
  const yes = await confirm({
    title: t('settings.input.confirm.sync_title'),
    description: t('settings.input.confirm.sync_desc', { cur }),
    confirmText: t('settings.input.confirm.sync_confirm'),
    color: 'primary',
  })
  if (yes !== true) return
  const r = (await backend.containers.syncLocalMouseCalibration(cur)) as any
  if (r) {
    toast.add({
      title: t('settings.input.toast.synced_title', { n: r.updated?.length ?? 0 }),
      description: r.skipped?.length
        ? t('settings.input.toast.synced_skipped', { n: r.skipped.length })
        : undefined,
      color: 'success',
    })
  }
}

// 校准：开独立置顶校准 HUD 窗 (用户自己切到目标游戏按 F8), 等它 emit 结果写回这一档。
async function openCalibratorFor(i: number) {
  if (i < 0 || i >= profiles.value.length) return
  const id = 'calib-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now()
  const ok = await backend.tools.openCalibratorHUD(id)
  if (!ok) return // 开窗失败 (invoke 已 toast)
  const r = await awaitWailsEvent<{ id: string; counts?: number; cancelled?: boolean }>(
    'calibration:result',
    (p) => p?.id === id,
  )
  if (!r.cancelled && typeof r.counts === 'number' && r.counts > 0) {
    await updateCounts(i, r.counts)
  }
}
</script>
