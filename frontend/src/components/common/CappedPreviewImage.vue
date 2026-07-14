<!-- 保真预览：图片可适配容器，但最多按原始像素放大指定倍数，避免小素材被强行插值放糊。 -->
<template>
  <img :src="src" :alt="alt" class="h-full w-full object-contain" :style="sizeCap" @load="onLoad" />
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    src: string
    alt?: string
    maxUpscale?: number
  }>(),
  {
    alt: '',
    maxUpscale: 2,
  },
)

const naturalSize = ref({ width: 0, height: 0 })

const sizeCap = computed(() => {
  const scale = Math.max(1, props.maxUpscale)
  if (!naturalSize.value.width || !naturalSize.value.height) return undefined
  return {
    maxWidth: `${naturalSize.value.width * scale}px`,
    maxHeight: `${naturalSize.value.height * scale}px`,
  }
})

function onLoad(event: Event) {
  const image = event.currentTarget as HTMLImageElement
  naturalSize.value = {
    width: image.naturalWidth,
    height: image.naturalHeight,
  }
}

watch(
  () => props.src,
  () => {
    naturalSize.value = { width: 0, height: 0 }
  },
)
</script>
