<!-- Drag-time alignment guides (PS smart-guides style).
     渲染在 vue-flow viewport transform 内部, 自动跟随 pan/zoom. -->
<template>
  <svg
    v-if="guides.length > 0"
    class="absolute inset-0 pointer-events-none w-full h-full"
    style="z-index: 5; overflow: visible;"
  >
    <g :transform="transform">
      <line
        v-for="(g, idx) in guides"
        :key="idx"
        :x1="g.x1"
        :y1="g.y1"
        :x2="g.x2"
        :y2="g.y2"
        :stroke="g.axis === 'y' ? '#06b6d4' : '#d946ef'"
        stroke-width="1"
        :stroke-dasharray="`${4 / Math.max(viewport.zoom, 0.5)} ${2 / Math.max(viewport.zoom, 0.5)}`"
        vector-effect="non-scaling-stroke"
      />
    </g>
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useVueFlow } from '@vue-flow/core'

export interface SnapGuide {
  axis: 'x' | 'y'
  x1: number; y1: number
  x2: number; y2: number
}

defineProps<{
  guides: SnapGuide[]
}>()

const { viewport } = useVueFlow()

const transform = computed(() => {
  return `translate(${viewport.value.x}, ${viewport.value.y}) scale(${viewport.value.zoom})`
})
</script>
