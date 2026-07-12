<template>
  <div class="px-8 py-6 space-y-6">
    <!-- 启动器总说明 + 全局选项 -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-layout-grid-add" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">{{ t('settingsLauncher.title') }}</h2>
      </div>
      <p class="text-xs text-dimmed leading-relaxed">
        {{ t('settingsLauncher.intro') }}
      </p>

      <div class="border-t border-default/60" />

      <div class="flex items-center justify-between gap-6">
        <div>
          <div class="text-sm text-default">{{ t('settingsLauncher.display_label') }}</div>
          <p class="text-xs text-dimmed mt-0.5">{{ t('settingsLauncher.display_hint') }}</p>
        </div>
        <USelect
          :model-value="display"
          :items="displayItems"
          class="w-32"
          @update:model-value="(v: string) => setDisplay(v)"
        />
      </div>
    </section>

    <!-- 编排：单条有序块列表 -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-3">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-layout-list" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">
          {{ t('settingsLauncher.layout_title') }}
        </h2>
        <span class="text-xs text-dimmed">({{ editItems.length }})</span>
      </div>
      <p class="text-xs text-dimmed leading-relaxed">
        {{ t('settingsLauncher.layout_hint') }}
      </p>

      <div
        v-if="editItems.length === 0"
        class="text-xs text-dimmed py-6 text-center border border-dashed border-default/60 rounded-lg"
      >
        {{ t('settingsLauncher.empty') }}
      </div>
      <VueDraggable
        v-else
        v-model="editItems"
        :animation="150"
        handle=".drag-h"
        class="space-y-2"
        @end="persist"
      >
        <div
          v-for="b in editItems"
          :key="b.id"
          class="flex flex-col gap-1 px-3 py-2 rounded-md bg-elevated/30 border border-default/60"
        >
          <div class="flex items-center gap-2">
            <UIcon
              name="i-tabler-grip-vertical"
              class="drag-h size-4 text-dimmed cursor-grab shrink-0"
            />

            <!-- 容器按钮块 -->
            <template v-if="b.type === 'container'">
              <UPopover :ui="{ content: 'w-[300px] p-2' }">
                <UButton
                  size="xs"
                  variant="outline"
                  color="neutral"
                  square
                  class="shrink-0"
                  :title="t('settingsLauncher.pick_icon')"
                >
                  <UIcon
                    :name="b.icon || 'i-tabler-photo-plus'"
                    class="size-4"
                    :class="b.icon ? 'text-toned' : 'text-dimmed'"
                  />
                </UButton>
                <template #content>
                  <div class="space-y-2">
                    <IconPicker
                      :model-value="b.icon"
                      @update:model-value="(v: string) => setIcon(b.id, v)"
                    />
                    <UButton
                      v-if="b.icon"
                      size="xs"
                      variant="ghost"
                      color="neutral"
                      block
                      @click="setIcon(b.id, '')"
                    >
                      {{ t('settingsLauncher.clear_icon') }}
                    </UButton>
                  </div>
                </template>
              </UPopover>
              <UInput
                :model-value="b.label"
                size="sm"
                class="flex-1 min-w-0"
                :placeholder="containerName(b.containerId)"
                :title="containerName(b.containerId)"
                @update:model-value="(v: string | number) => setLabel(b.id, String(v))"
              />
              <HotkeyCaptureInput
                class="w-28 sm:w-32 shrink-0"
                :model-value="containerHotkey(b.containerId)"
                @update:model-value="(v: string) => setHotkey(b.containerId!, v)"
              />
            </template>

            <!-- 文字标题块 -->
            <template v-else-if="b.type === 'label'">
              <UIcon name="i-tabler-heading" class="size-4 text-dimmed shrink-0" />
              <UInput
                :model-value="b.label"
                size="sm"
                class="flex-1 min-w-0"
                :placeholder="t('settingsLauncher.label_placeholder')"
                @update:model-value="(v: string | number) => setLabel(b.id, String(v))"
              />
            </template>

            <!-- 水平分隔符块 -->
            <div
              v-else-if="b.type === 'hsep'"
              class="flex-1 flex items-center gap-2 text-xs text-dimmed"
            >
              <span class="flex-1 border-t border-default/60" />
              <span class="shrink-0 inline-flex items-center gap-1"
                ><UIcon name="i-tabler-separator-horizontal" class="size-4" />
                {{ t('settingsLauncher.hsep') }}</span
              >
              <span class="flex-1 border-t border-default/60" />
            </div>

            <!-- 垂直分隔符块 -->
            <div
              v-else-if="b.type === 'vsep'"
              class="flex-1 inline-flex items-center gap-1 text-xs text-dimmed"
            >
              <UIcon name="i-tabler-separator-vertical" class="size-4" />
              {{ t('settingsLauncher.vsep') }}
            </div>

            <UButton
              size="xs"
              variant="ghost"
              color="error"
              icon="i-tabler-trash"
              :title="t('settingsLauncher.delete_block')"
              @click="removeBlock(b.id)"
            />
          </div>
          <!-- 容器块：自定义了显示名时，常驻一行原容器名兜底 -->
          <p v-if="b.type === 'container' && b.label" class="pl-7 text-xs text-dimmed truncate">
            {{ t('settingsLauncher.from_container', { name: containerName(b.containerId) }) }}
          </p>
        </div>
      </VueDraggable>

      <!-- 添加块 -->
      <div class="flex flex-wrap items-center gap-2 pt-1">
        <USelect
          v-if="containerItems.length"
          :model-value="undefined"
          :items="containerItems"
          size="sm"
          class="w-44"
          :placeholder="t('settingsLauncher.add_container')"
          @update:model-value="(v: string) => addContainer(v)"
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
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend } from '@/lib/backend'
import { useSettingsStore, type LauncherBlock } from '@/stores/settings'
import { useContainersStore } from '@/stores/containers'
import { useHotkeysStore } from '@/stores/hotkeys'
import { VueDraggable } from 'vue-draggable-plus'
import IconPicker from '@/components/containers/inline/IconPicker.vue'
import HotkeyCaptureInput from '@/components/hotkeys/HotkeyCaptureInput.vue'

const settingsStore = useSettingsStore()
const containersStore = useContainersStore()
const hotkeysStore = useHotkeysStore()
const { t } = useI18n()

// 本地工作副本（浅拷贝每块），本页是 launcherItems 唯一编辑者，每次改动立即 persist。
const editItems = ref<LauncherBlock[]>([])
function copyItems(items: LauncherBlock[]): LauncherBlock[] {
  return items.map((b) => ({ ...b }))
}
function syncFromStore() {
  editItems.value = copyItems(settingsStore.data?.ui.launcherItems ?? [])
}
watch(() => settingsStore.data?.ui.launcherItems, syncFromStore, { immediate: true })

const display = computed(() => settingsStore.data?.ui.launcherDisplay || 'both')
const displayItems = computed(() => [
  { label: t('settingsLauncher.display_both'), value: 'both' },
  { label: t('settingsLauncher.display_icon'), value: 'icon' },
  { label: t('settingsLauncher.display_text'), value: 'text' },
])
function setDisplay(v: string) {
  void settingsStore.patch({ ui: { launcherDisplay: v } })
}

function persist() {
  void settingsStore.patch({ ui: { launcherItems: copyItems(editItems.value) } })
}
function genId(): string {
  return 'lb_' + Math.random().toString(36).slice(2, 10)
}
function block(id: string) {
  return editItems.value.find((b) => b.id === id)
}
function addContainer(cid: string) {
  if (!cid) return
  editItems.value.push({ id: genId(), type: 'container', containerId: cid, icon: '', label: '' })
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
  editItems.value = editItems.value.filter((b) => b.id !== id)
  persist()
}
function setIcon(id: string, icon: string) {
  const b = block(id)
  if (b) {
    b.icon = icon
    persist()
  }
}
function setLabel(id: string, label: string) {
  const b = block(id)
  if (b) {
    b.label = label
    persist()
  }
}
async function setHotkey(cid: string, hk: string) {
  await backend.hotkeys.update('container.' + cid, hk)
  await hotkeysStore.reload()
}
function containerName(id: string | undefined): string {
  return (
    containersStore.list.find((c) => c.id === id)?.name ?? t('settingsLauncher.deleted_container')
  )
}
function containerHotkey(id: string | undefined): string {
  return hotkeysStore.list.find((e) => e.key === 'container.' + id)?.hotkeyStr ?? ''
}
// 块自由编排，同一容器允许出现多次 → 不去重，列全部容器。
const containerItems = computed(() =>
  containersStore.list.map((c) => ({ label: c.name, value: c.id })),
)

onMounted(() => {
  void settingsStore.load()
  void containersStore.reload()
  void hotkeysStore.reload()
})
</script>
