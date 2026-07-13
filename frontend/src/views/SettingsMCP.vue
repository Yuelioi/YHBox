<template>
  <div class="settings-page">
    <!-- Title section -->
    <section class="settings-section">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-plug" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">{{ t('settingsMCP.title') }}</h2>
      </div>
    </section>

    <!-- Arm toggle section -->
    <section class="settings-section">
      <div class="flex items-center justify-between gap-6">
        <div>
          <div class="text-sm text-default">{{ t('settingsMCP.armLabel') }}</div>
        </div>
        <USwitch
          :model-value="armed"
          :aria-label="t('settingsMCP.armLabel')"
          @update:model-value="onToggleArmed"
        />
      </div>

      <!-- Warning shown when armed -->
      <div
        v-if="armed"
        class="flex items-start gap-2 rounded-md border border-error/50 bg-error/10 px-3 py-2.5"
      >
        <UIcon name="i-tabler-alert-triangle" class="size-4 text-error mt-0.5 shrink-0" />
        <p class="text-xs text-error leading-relaxed">{{ t('settingsMCP.armWarning') }}</p>
      </div>
    </section>

    <!-- Server URL section -->
    <section class="settings-section">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-link" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">{{ t('settingsMCP.urlLabel') }}</h2>
      </div>

      <p class="text-xs text-dimmed">{{ t('settingsMCP.urlHint') }}</p>

      <div class="flex items-center gap-2">
        <UInput
          :model-value="MCP_URL"
          readonly
          size="sm"
          class="flex-1 min-w-0 font-mono"
          :aria-label="t('settingsMCP.urlLabel')"
        />
        <UButton size="sm" variant="soft" color="primary" icon="i-tabler-copy" @click="copyUrl">
          {{ t('settingsMCP.copy') }}
        </UButton>
      </div>

      <p v-if="copied" class="text-xs text-success flex items-center gap-1">
        <UIcon name="i-tabler-circle-check" class="size-3.5" />
        {{ t('settingsMCP.copied') }}
      </p>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '@/stores/settings'

const { t } = useI18n()
const store = useSettingsStore()

const MCP_URL = 'http://127.0.0.1:8765/mcp'

const armed = computed(() => store.data?.mcp?.armed ?? false)

async function onToggleArmed(v: boolean) {
  await store.patch({ mcp: { armed: v } })
}

const copied = ref(false)

async function copyUrl() {
  try {
    await navigator.clipboard.writeText(MCP_URL)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch {
    // clipboard not available
  }
}
</script>
