<!-- 模板缩略图: 按 firstBlobSha 异步拉 dataURL 显示. 未加载/无图 → 占位图标. -->
<template>
  <img
    v-if="src"
    :src="src"
    class="max-w-full max-h-full object-contain"
    alt=""
    draggable="false"
  />
  <UIcon v-else name="i-tabler-photo" class="size-6 text-dimmed" />
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useTemplatesStore } from '@/stores/templates'

const props = defineProps<{ sha?: string }>()
const tplStore = useTemplatesStore()
const src = ref<string | undefined>(undefined)

watch(
  () => props.sha,
  async (sha) => {
    src.value = undefined
    if (!sha) return
    const r = await tplStore.readBlobDataURL(sha)
    if (typeof r === 'string') src.value = r
  },
  { immediate: true },
)
</script>
