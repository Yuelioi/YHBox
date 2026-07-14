<template>
  <div class="settings-page settings-page--wide">
    <SettingsSection
      :title="t('settingsLauncher.appearance_title')"
      :description="t('settingsLauncher.display_hint')"
      icon="i-tabler-adjustments-horizontal"
    >
      <SettingsRow :label="t('settingsLauncher.display_label')">
        <USelect
          :model-value="display"
          :items="displayItems"
          class="w-44"
          :aria-label="t('settingsLauncher.display_label')"
          @update:model-value="setDisplay"
        />
      </SettingsRow>
    </SettingsSection>

    <SettingsSection
      :title="t('settingsLauncher.layout_title')"
      :description="t('settingsLauncher.layout_hint')"
      icon="i-tabler-layout-list"
      :badge="String(editItems.length)"
    >
      <div class="launcher-builder">
        <div class="min-w-0 space-y-3">
          <div v-if="editItems.length === 0" class="settings-empty-state">
            <UIcon name="i-tabler-layout-off" class="size-6 text-dimmed" />
            <p class="text-sm font-medium text-default">{{ t('settingsLauncher.empty') }}</p>
          </div>
          <VueDraggable
            v-else
            v-model="editItems"
            :animation="150"
            handle=".drag-h"
            class="space-y-2"
            @end="persist"
          >
            <article v-for="(block, index) in editItems" :key="block.id" class="launcher-block">
              <UIcon
                name="i-tabler-grip-vertical"
                class="drag-h size-4 shrink-0 cursor-grab text-dimmed"
              />
              <template v-if="block.type === 'container'">
                <UPopover :ui="{ content: 'w-[300px] p-2' }">
                  <UButton
                    size="xs"
                    variant="outline"
                    color="neutral"
                    square
                    :title="t('settingsLauncher.pick_icon')"
                    :aria-label="t('settingsLauncher.pick_icon')"
                  >
                    <UIcon :name="block.icon || 'i-tabler-photo-plus'" class="size-4" />
                  </UButton>
                  <template #content>
                    <div class="space-y-2">
                      <IconPicker
                        :model-value="block.icon"
                        @update:model-value="(value: string) => setIcon(block.id, value)"
                      />
                      <UButton
                        v-if="block.icon"
                        size="xs"
                        variant="ghost"
                        color="neutral"
                        block
                        @click="setIcon(block.id, '')"
                      >
                        {{ t('settingsLauncher.clear_icon') }}
                      </UButton>
                    </div>
                  </template>
                </UPopover>
                <div class="min-w-0 flex-1">
                  <UInput
                    :model-value="block.label"
                    size="sm"
                    :placeholder="containerName(block.containerId)"
                    :aria-label="t('settingsLauncher.label_placeholder')"
                    @update:model-value="
                      (value: string | number) => setLabel(block.id, String(value))
                    "
                    @change="persist"
                  />
                  <p class="mt-1 truncate text-[11px] text-dimmed">
                    {{
                      t('settingsLauncher.from_container', {
                        name: containerName(block.containerId),
                      })
                    }}
                  </p>
                </div>
                <HotkeyCaptureInput
                  class="w-32 shrink-0"
                  :model-value="containerHotkey(block.containerId)"
                  :aria-label="
                    t('settingsLauncher.hotkey_aria', { name: containerName(block.containerId) })
                  "
                  @update:model-value="(value: string) => setHotkey(block.containerId!, value)"
                />
              </template>
              <template v-else-if="block.type === 'label'">
                <UIcon name="i-tabler-heading" class="size-4 shrink-0 text-dimmed" />
                <UInput
                  :model-value="block.label"
                  class="min-w-0 flex-1"
                  size="sm"
                  :placeholder="t('settingsLauncher.label_placeholder')"
                  @update:model-value="
                    (value: string | number) => setLabel(block.id, String(value))
                  "
                  @change="persist"
                />
              </template>
              <div v-else class="flex min-w-0 flex-1 items-center gap-2 text-xs text-dimmed">
                <UIcon
                  :name="
                    block.type === 'hsep'
                      ? 'i-tabler-separator-horizontal'
                      : 'i-tabler-separator-vertical'
                  "
                  class="size-4"
                />
                {{ t(block.type === 'hsep' ? 'settingsLauncher.hsep' : 'settingsLauncher.vsep') }}
              </div>
              <div class="ml-auto flex shrink-0 items-center">
                <UButton
                  size="xs"
                  variant="ghost"
                  color="neutral"
                  icon="i-tabler-arrow-up"
                  :disabled="index === 0"
                  :aria-label="t('settingsLauncher.move_up')"
                  @click="moveBlock(index, index - 1)"
                />
                <UButton
                  size="xs"
                  variant="ghost"
                  color="neutral"
                  icon="i-tabler-arrow-down"
                  :disabled="index === editItems.length - 1"
                  :aria-label="t('settingsLauncher.move_down')"
                  @click="moveBlock(index, index + 1)"
                />
                <UButton
                  size="xs"
                  variant="ghost"
                  color="error"
                  icon="i-tabler-trash"
                  :aria-label="t('settingsLauncher.delete_block')"
                  @click="removeBlock(block.id)"
                />
              </div>
            </article>
          </VueDraggable>

          <div class="launcher-library">
            <p class="text-xs font-medium text-default">
              {{ t('settingsLauncher.library_title') }}
            </p>
            <div class="mt-2 flex flex-wrap items-center gap-2">
              <USelect
                v-if="containerItems.length"
                :model-value="undefined"
                :items="containerItems"
                size="sm"
                class="w-48"
                :placeholder="t('settingsLauncher.add_container')"
                @update:model-value="addContainer"
              />
              <UButton
                size="xs"
                variant="soft"
                color="neutral"
                icon="i-tabler-heading"
                @click="addLabel"
                >{{ t('settingsLauncher.label_block') }}</UButton
              >
              <UButton
                size="xs"
                variant="soft"
                color="neutral"
                icon="i-tabler-separator-horizontal"
                @click="addHsep"
                >{{ t('settingsLauncher.hsep') }}</UButton
              >
              <UButton
                size="xs"
                variant="soft"
                color="neutral"
                icon="i-tabler-separator-vertical"
                @click="addVsep"
                >{{ t('settingsLauncher.vsep') }}</UButton
              >
            </div>
          </div>
        </div>

        <aside class="launcher-preview" aria-live="polite">
          <div class="mb-3 flex items-center justify-between">
            <p class="text-xs font-medium text-default">
              {{ t('settingsLauncher.preview_title') }}
            </p>
            <UBadge size="xs" variant="subtle" color="neutral">{{
              t('settingsLauncher.live_badge')
            }}</UBadge>
          </div>
          <div class="launcher-preview__window">
            <div class="launcher-preview__handle" />
            <template v-if="editItems.length">
              <template v-for="block in editItems" :key="block.id">
                <p v-if="block.type === 'label'" class="launcher-preview__label">
                  {{ block.label || t('settingsLauncher.untitled_label') }}
                </p>
                <div v-else-if="block.type === 'hsep'" class="launcher-preview__hsep" />
                <div v-else-if="block.type === 'vsep'" class="launcher-preview__vsep" />
                <div
                  v-else
                  class="launcher-preview__button"
                  :class="
                    containerExists(block.containerId) ? '' : 'launcher-preview__button--missing'
                  "
                >
                  <UIcon
                    v-if="display !== 'text'"
                    :name="block.icon || 'i-tabler-player-play-filled'"
                    class="size-4"
                  />
                  <span v-if="display !== 'icon'" class="truncate">{{
                    block.label || containerName(block.containerId)
                  }}</span>
                </div>
              </template>
            </template>
            <p v-else class="text-center text-xs text-dimmed">
              {{ t('settingsLauncher.preview_empty') }}
            </p>
          </div>
        </aside>
      </div>
    </SettingsSection>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { VueDraggable } from 'vue-draggable-plus'
import { backend } from '@/lib/backend'
import { useSettingsStore, type LauncherBlock } from '@/stores/settings'
import { useContainersStore } from '@/stores/containers'
import { useHotkeysStore } from '@/stores/hotkeys'
import IconPicker from '@/components/containers/inline/IconPicker.vue'
import HotkeyCaptureInput from '@/components/hotkeys/HotkeyCaptureInput.vue'
import SettingsRow from '@/components/settings/SettingsRow.vue'
import SettingsSection from '@/components/settings/SettingsSection.vue'

const settingsStore = useSettingsStore()
const containersStore = useContainersStore()
const hotkeysStore = useHotkeysStore()
const { t } = useI18n()
const editItems = ref<LauncherBlock[]>([])
const copyItems = (items: LauncherBlock[]) => items.map((block) => ({ ...block }))
const syncFromStore = () =>
  (editItems.value = copyItems(settingsStore.data?.ui.launcherItems ?? []))
watch(() => settingsStore.data?.ui.launcherItems, syncFromStore, { immediate: true })

const display = computed(() => settingsStore.data?.ui.launcherDisplay || 'both')
const displayItems = computed(() => [
  { label: t('settingsLauncher.display_both'), value: 'both' },
  { label: t('settingsLauncher.display_icon'), value: 'icon' },
  { label: t('settingsLauncher.display_text'), value: 'text' },
])
const containerItems = computed(() =>
  containersStore.list.map((container) => ({ label: container.name, value: container.id })),
)
const persist = () =>
  void settingsStore.patch({ ui: { launcherItems: copyItems(editItems.value) } })
const setDisplay = (value: string) => void settingsStore.patch({ ui: { launcherDisplay: value } })
const block = (id: string) => editItems.value.find((item) => item.id === id)
const genId = () => `lb_${crypto.randomUUID()}`

function moveBlock(from: number, to: number) {
  if (to < 0 || to >= editItems.value.length) return
  const [item] = editItems.value.splice(from, 1)
  if (!item) return
  editItems.value.splice(to, 0, item)
  persist()
}
function addContainer(containerId: string) {
  if (!containerId) return
  editItems.value.push({ id: genId(), type: 'container', containerId, icon: '', label: '' })
  persist()
}
function addLabel() {
  editItems.value.push({ id: genId(), type: 'label', label: '' })
  persist()
}
function addHsep() {
  editItems.value.push({ id: genId(), type: 'hsep' })
  persist()
}
function addVsep() {
  editItems.value.push({ id: genId(), type: 'vsep' })
  persist()
}
function removeBlock(id: string) {
  editItems.value = editItems.value.filter((item) => item.id !== id)
  persist()
}
function setIcon(id: string, icon: string) {
  const item = block(id)
  if (item) {
    item.icon = icon
    persist()
  }
}
function setLabel(id: string, label: string) {
  const item = block(id)
  if (item) item.label = label
}
async function setHotkey(containerId: string, hotkey: string) {
  await backend.hotkeys.update(`container.${containerId}`, hotkey)
  await hotkeysStore.reload()
}
function containerExists(id?: string) {
  return containersStore.list.some((container) => container.id === id)
}
function containerName(id?: string) {
  return (
    containersStore.list.find((container) => container.id === id)?.name ??
    t('settingsLauncher.deleted_container')
  )
}
function containerHotkey(id?: string) {
  return hotkeysStore.list.find((item) => item.key === `container.${id}`)?.hotkeyStr ?? ''
}

onMounted(() => {
  void containersStore.reload()
  void hotkeysStore.reload()
})
</script>
