<!-- 模板缩略图: 按 firstBlobSha 异步拉 dataURL 显示. 未加载/无图 → 占位图标. -->
<template>
  <div v-if="loading" class="h-full w-full animate-pulse bg-elevated/55" aria-hidden="true" />
  <CappedPreviewImage
    v-else-if="src"
    :src="src"
    :alt="alt"
    :max-upscale="maxUpscale"
    draggable="false"
  />
  <div v-else class="flex h-full w-full flex-col items-center justify-center gap-1.5 text-dimmed">
    <UIcon :name="failed ? 'i-tabler-photo-off' : 'i-tabler-photo'" class="size-6" />
    <span v-if="failed" class="text-xs">{{ t('assetBrowser.previewFailed') }}</span>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTemplatesStore } from '@/stores/templates'
import CappedPreviewImage from '@/components/common/CappedPreviewImage.vue'

const {
  sha,
  alt,
  maxUpscale = 2,
} = defineProps<{
  sha?: string
  alt?: string
  maxUpscale?: number
}>()
const tplStore = useTemplatesStore()
const { t } = useI18n()
const src = ref<string | undefined>(undefined)
const loading = ref(false)
const failed = ref(false)

watch(
  () => sha,
  async (sha) => {
    src.value = undefined
    failed.value = false
    if (!sha) return
    loading.value = true
    try {
      const r = await tplStore.readBlobDataURL(sha)
      if (typeof r === 'string') src.value = r
      else failed.value = true
    } catch {
      failed.value = true
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)
</script>
