<template>
  <BaseEdge :id="id" :path="path" :style="style" />
  <circle
    data-testid="workflow-reroute-point"
    v-for="(point, index) in reroutes"
    :key="index"
    :cx="point.x"
    :cy="point.y"
    r="6"
    class="cursor-move fill-default stroke-primary stroke-2"
    @pointerdown.stop="startDrag($event, index)"
    @dblclick.stop="remove(index)"
  />
</template>

<script setup lang="ts">
import { computed, ref, watch, type CSSProperties } from 'vue'
import { BaseEdge, useVueFlow } from '@vue-flow/core'
import type { Edge, Position } from '../../../../contracts/workflow/current/workflow-source'

const props = defineProps<{
  id: string
  sourceX: number
  sourceY: number
  targetX: number
  targetY: number
  style?: CSSProperties
  edge: Edge
}>()
const emit = defineEmits<{ update: [reroutes: Position[]] }>()
const { screenToFlowCoordinate } = useVueFlow()
const reroutes = ref<Position[]>([...(props.edge.presentation?.reroutes ?? [])])
watch(
  () => props.edge.presentation?.reroutes,
  (value) => {
    reroutes.value = [...(value ?? [])]
  },
  { deep: true },
)
const path = computed(() => {
  const points = [
    { x: props.sourceX, y: props.sourceY },
    ...reroutes.value,
    { x: props.targetX, y: props.targetY },
  ]
  return points.map((point, index) => `${index ? 'L' : 'M'} ${point.x} ${point.y}`).join(' ')
})

function startDrag(event: PointerEvent, index: number): void {
  const move = (current: PointerEvent) => {
    const next = [...reroutes.value]
    next[index] = screenToFlowCoordinate({ x: current.clientX, y: current.clientY })
    reroutes.value = next
  }
  const stop = () => {
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', stop)
    emit('update', [...reroutes.value])
  }
  window.addEventListener('pointermove', move)
  window.addEventListener('pointerup', stop, { once: true })
  if (event.currentTarget instanceof Element) {
    event.currentTarget.setPointerCapture(event.pointerId)
  }
}

function remove(index: number): void {
  emit(
    'update',
    reroutes.value.filter((_, current) => current !== index),
  )
}
</script>
