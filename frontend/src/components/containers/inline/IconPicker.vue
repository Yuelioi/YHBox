<template>
  <!-- 图标选择器：默认显常用 (visualRegistry.ICONS)；搜索框过滤全套 tabler (~5900, 懒加载离线)。
       modelValue = 完整 tabler 名 (i-tabler-xxx)。 -->
  <div class="space-y-2 w-full">
    <UInput
      v-model="query" size="xs" class="w-full" placeholder="搜索图标…" icon="i-tabler-search"
      :loading="searching"
    />
    <div class="flex flex-wrap gap-1.5 max-h-48 overflow-auto">
      <button
        v-for="ic in shown"
        :key="ic"
        type="button"
        class="icp-chip"
        :class="{ 'is-selected': modelValue === ic }"
        :title="ic"
        @click="emit('update:modelValue', ic)"
      >
        <UIcon :name="ic" class="size-4" />
      </button>
      <div v-if="query.trim() && shown.length === 0 && !searching" class="text-[11px] text-dimmed py-2 px-1">
        无匹配图标
      </div>
    </div>
    <p v-if="query.trim()" class="text-[10px] text-dimmed">搜全套 tabler · 最多显 120 个</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ICONS } from '../visualRegistry'

defineProps<{
  modelValue: string | undefined | null
}>()
const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>()

const curated = ICONS.map((e) => e.icon)
const query = ref('')
const allNames = ref<string[]>([]) // 全套 tabler（懒加载）
const searching = ref(false)
let loaded = false

async function ensureLoaded() {
  if (loaded) return
  loaded = true
  searching.value = true
  try {
    const mod: any = await import('@iconify-json/tabler/icons.json')
    const data = mod.default ?? mod
    allNames.value = Object.keys(data.icons ?? {}).map((n) => `i-tabler-${n}`)
  } catch {
    // 离线 / 缺包 → 仅 curated 可用
  } finally {
    searching.value = false
  }
}
watch(query, (q) => {
  if (q.trim()) void ensureLoaded()
})

const shown = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return curated
  const pool = allNames.value.length ? allNames.value : curated
  return pool.filter((n) => n.includes(q)).slice(0, 120) // 限量防卡顿
})
</script>

<style scoped>
.icp-chip {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: var(--ui-text-toned);
  cursor: pointer;
  transition: all 120ms ease;
}
.icp-chip:hover {
  background: rgba(255, 255, 255, 0.12);
  color: var(--ui-text-default);
}
.icp-chip.is-selected {
  background: var(--ui-primary, #6366f1);
  border-color: var(--ui-primary, #6366f1);
  color: white;
}
</style>
