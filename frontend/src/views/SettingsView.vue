<template>
  <div class="settings-shell" :data-settings-theme="activeKey" data-testid="settings-view">
    <aside class="settings-sidebar" :aria-label="t('sidebar.settings')">
      <div class="flex items-center gap-2 px-1">
        <div
          class="flex size-8 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10"
        >
          <UIcon name="i-tabler-settings" class="size-4 text-primary" aria-hidden="true" />
        </div>
        <div class="min-w-0">
          <h1 class="truncate text-sm font-semibold text-highlighted">
            {{ t('sidebar.settings') }}
          </h1>
          <p class="truncate text-[11px] text-dimmed">{{ t('settingsCenter.local_hint') }}</p>
        </div>
      </div>

      <UInput
        v-model="searchQuery"
        size="sm"
        icon="i-tabler-search"
        :placeholder="t('settingsCenter.search_placeholder')"
        :aria-label="t('settingsCenter.search_placeholder')"
        class="w-full"
      >
        <template v-if="searchQuery" #trailing>
          <UButton
            size="xs"
            variant="link"
            color="neutral"
            icon="i-tabler-x"
            :aria-label="t('settingsCenter.clear_search')"
            @click="searchQuery = ''"
          />
        </template>
      </UInput>

      <nav
        class="settings-navigation"
        role="tablist"
        aria-orientation="vertical"
        :aria-label="t('settingsCenter.themes_label')"
      >
        <section
          v-for="group in filteredThemeGroups"
          :key="group.key"
          class="settings-nav-group"
          :data-testid="`settings-group-${group.key}`"
          :aria-labelledby="`settings-group-${group.key}`"
        >
          <h2
            :id="`settings-group-${group.key}`"
            class="px-2 pb-1 pt-2 text-[10px] font-semibold tracking-wide text-dimmed"
          >
            {{ t(group.labelKey) }}
          </h2>
          <UButton
            v-for="theme in group.themes"
            :id="`settings-tab-${theme.key}`"
            :key="theme.key"
            size="sm"
            variant="ghost"
            color="neutral"
            role="tab"
            :tabindex="activeKey === theme.key ? 0 : -1"
            :aria-selected="activeKey === theme.key"
            aria-controls="settings-tabpanel"
            class="settings-nav-item"
            :class="activeKey === theme.key ? 'settings-nav-item--active' : ''"
            @click="selectTheme(theme.key)"
            @keydown="onTabKeydown($event, theme.key)"
          >
            <UIcon
              :name="theme.icon"
              class="size-4 shrink-0"
              :class="activeKey === theme.key ? 'text-primary' : 'text-dimmed'"
              aria-hidden="true"
            />
            <span class="min-w-0 flex-1 text-left">
              <span class="block truncate text-xs font-medium">{{ t(theme.labelKey) }}</span>
              <span class="mt-0.5 block truncate text-[10px] font-normal text-dimmed">
                {{ t(theme.descriptionKey) }}
              </span>
            </span>
            <UIcon
              v-if="activeKey === theme.key"
              name="i-tabler-chevron-right"
              class="size-3.5 shrink-0 text-primary"
              aria-hidden="true"
            />
          </UButton>
        </section>

        <p
          v-if="filteredThemeGroups.length === 0"
          class="px-3 py-6 text-center text-xs text-dimmed"
        >
          {{ t('settingsCenter.no_results') }}
        </p>
      </nav>
    </aside>

    <section class="settings-content">
      <SettingsPageHeader
        :title="t(activeTheme.labelKey)"
        :description="t(activeTheme.descriptionKey)"
        :icon="activeTheme.icon"
      />
      <div
        id="settings-tabpanel"
        role="tabpanel"
        :aria-labelledby="`settings-tab-${activeKey}`"
        tabindex="0"
        class="settings-tabpanel min-h-0 min-w-0 flex-1 overflow-auto focus:outline-none"
      >
        <KeepAlive>
          <component :is="activeComponent" />
        </KeepAlive>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import SettingsGeneral from './SettingsGeneral.vue'
import SettingsHotkeys from './SettingsHotkeys.vue'
import SettingsInput from './SettingsInput.vue'
import SettingsLauncher from './SettingsLauncher.vue'
import SettingsAI from './SettingsAI.vue'
import SettingsNetwork from './SettingsNetwork.vue'
import SettingsApplications from './SettingsApplications.vue'
import SettingsAutomation from './SettingsAutomation.vue'
import SettingsPageHeader from '@/components/settings/SettingsPageHeader.vue'
import {
  groupSettingsThemes,
  SETTINGS_THEMES,
  isSettingsThemeKey,
  type SettingsThemeKey,
} from '@/settings/registry'

const LAST_THEME_KEY = 'yotta.settings.section'
const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const searchQuery = ref('')

const componentByTheme = {
  general: SettingsGeneral,
  hotkeys: SettingsHotkeys,
  input: SettingsInput,
  launcher: SettingsLauncher,
  ai: SettingsAI,
  network: SettingsNetwork,
  applications: SettingsApplications,
  automation: SettingsAutomation,
} as const

function initialTheme(): SettingsThemeKey {
  const fromRoute = route.query.section
  if (isSettingsThemeKey(fromRoute)) return fromRoute
  try {
    const stored = localStorage.getItem(LAST_THEME_KEY)
    if (isSettingsThemeKey(stored)) return stored
  } catch {
    // localStorage unavailable: fall back to General.
  }
  return 'general'
}

const activeKey = ref<SettingsThemeKey>(initialTheme())
const activeTheme = computed(
  () => SETTINGS_THEMES.find((theme) => theme.key === activeKey.value) ?? SETTINGS_THEMES[0],
)
const activeComponent = computed(() => componentByTheme[activeKey.value])
const filteredThemes = computed(() => {
  const query = searchQuery.value.trim().toLocaleLowerCase()
  if (!query) return SETTINGS_THEMES
  return SETTINGS_THEMES.filter((theme) =>
    `${t(theme.labelKey)} ${t(theme.descriptionKey)}`.toLocaleLowerCase().includes(query),
  )
})
const filteredThemeGroups = computed(() => groupSettingsThemes(filteredThemes.value))

watch(
  () => route.query.section,
  (section) => {
    if (isSettingsThemeKey(section) && section !== activeKey.value) activeKey.value = section
  },
)

function selectTheme(key: SettingsThemeKey) {
  activeKey.value = key
  try {
    localStorage.setItem(LAST_THEME_KEY, key)
  } catch {
    // Preference persistence is optional; navigation still works.
  }
  if (route.query.section !== key) {
    void router.replace({ query: { ...route.query, section: key } })
  }
}

function onTabKeydown(event: KeyboardEvent, currentKey: SettingsThemeKey) {
  const visible = filteredThemes.value.map((theme) => theme.key)
  const current = visible.indexOf(currentKey)
  let next = current
  if (event.key === 'ArrowDown') next = (current + 1) % visible.length
  else if (event.key === 'ArrowUp') next = (current - 1 + visible.length) % visible.length
  else if (event.key === 'Home') next = 0
  else if (event.key === 'End') next = visible.length - 1
  else return

  const key = visible[next]
  if (!key) return
  event.preventDefault()
  selectTheme(key)
  void nextTick(() => document.getElementById(`settings-tab-${key}`)?.focus())
}
</script>
