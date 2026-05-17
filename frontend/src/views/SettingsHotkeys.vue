<template>
  <div class="px-8 py-6 space-y-5">
    <!-- 搜索框 -->
    <UInput
      v-model="searchText"
      placeholder="搜索热键名或绑定..."
      icon="i-tabler-search"
      class="w-full"
    />

    <!-- 按 source 分组渲染 -->
    <section
      v-for="group in filteredGrouped"
      :key="group.source"
      class="rounded-xl bg-default border border-default p-5 space-y-3"
    >
      <h3 class="text-sm font-medium text-highlighted flex items-center gap-2">
        <UIcon :name="groupIcon(group.source)" class="size-4 text-dimmed" />
        {{ groupLabel(group.source) }}
      </h3>
      <div class="space-y-2">
        <div
          v-for="entry in group.entries"
          :key="entry.key"
          class="flex items-center gap-3 px-3 py-2 rounded-md bg-elevated/30 border border-default/60"
        >
          <div class="flex-1 min-w-0">
            <div class="text-sm text-default truncate">{{ entry.label }}</div>
            <div
              v-if="entry.lastError"
              class="text-xs text-error mt-0.5 truncate"
              :title="entry.lastError"
            >
              ⚠ 注册失败：{{ entry.lastError }}
            </div>
            <div v-else-if="entry.readonlyReason" class="text-xs text-dimmed mt-0.5">
              {{ entry.readonlyReason }}
            </div>
            <div v-else-if="entry.status === 'unbound'" class="text-xs text-dimmed mt-0.5">
              未绑定
            </div>
          </div>
          <div class="w-56 shrink-0">
            <HotkeyCaptureInput
              :model-value="entry.hotkeyStr"
              :disabled="!!entry.readonlyReason"
              @update:model-value="(v: string) => onUpdate(entry.key, v)"
            />
          </div>
        </div>
      </div>
    </section>

    <p v-if="filteredGrouped.length === 0" class="text-xs text-dimmed text-center py-8">
      没有匹配的热键
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useToast } from '@nuxt/ui/composables'
import { useHotkeysStore } from '@/stores/hotkeys'
import HotkeyCaptureInput from '@/components/hotkeys/HotkeyCaptureInput.vue'

const store = useHotkeysStore()
const toast = useToast()
const searchText = ref('')

onMounted(() => {
  void store.reload()
})

const filteredGrouped = computed(() => {
  const q = searchText.value.trim().toLowerCase()
  const filtered = q
    ? store.list.filter(
        (e) => e.label.toLowerCase().includes(q) || e.hotkeyStr.toLowerCase().includes(q),
      )
    : store.list
  const groups: Record<string, typeof filtered> = { system: [], action: [] }
  for (const e of filtered) {
    if (groups[e.source]) groups[e.source].push(e)
  }
  // 按 label 字母序排（system 一组 / action 一组）
  for (const k of Object.keys(groups)) {
    groups[k].sort((a, b) => a.label.localeCompare(b.label, 'zh'))
  }
  return Object.entries(groups)
    .filter(([_, entries]) => entries.length > 0)
    .map(([source, entries]) => ({ source, entries }))
})

function groupIcon(source: string): string {
  return source === 'system' ? 'i-tabler-tool' : 'i-tabler-bolt'
}
function groupLabel(source: string): string {
  return source === 'system' ? '系统' : '动作'
}

async function onUpdate(key: string, hotkeyStr: string) {
  const ok = await store.update(key, hotkeyStr)
  if (ok) {
    toast.add({
      title: hotkeyStr ? `已绑定 ${hotkeyStr}` : '已清除热键',
      icon: 'i-tabler-check',
      color: 'neutral',
    })
  }
  // 失败 invoke wrapper 已 toast 带 [conflict]/[reserved]/[invalid] 前缀，不再重复
}
</script>
