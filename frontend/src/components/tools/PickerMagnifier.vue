<template>
  <!-- 像素级放大镜 (loupe): 画光标周围一小片原生像素放大, 中心方框标当前像素.
       位置由父组件用 absolute top/left 摆放; 这里只负责画内容. -->
  <div
    class="rounded-md overflow-hidden border border-default bg-default shadow-lg pointer-events-none"
    :style="{ width: size + 'px' }"
  >
    <canvas ref="cv" :width="size" :height="size" class="block" />
    <div
      class="px-1.5 py-0.5 text-[10px] font-mono tabular-nums text-toned border-t border-default text-center"
    >
      {{ Math.round(nx) }}, {{ Math.round(ny) }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    // 取源: 离屏 canvas (整图 1:1) 或 img 元素, drawImage 都吃.
    source: HTMLCanvasElement | HTMLImageElement | null
    nx: number // 光标原生 x
    ny: number // 光标原生 y
    srcPx?: number // 放大镜里横向展示多少原生像素 (奇数 → 有正中心像素)
    size?: number // loupe 画布边长 (px)
  }>(),
  { srcPx: 15, size: 136 },
)

const cv = ref<HTMLCanvasElement | null>(null)

function draw() {
  const canvas = cv.value
  if (!canvas || !props.source) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  const span = props.srcPx
  const mag = props.size / span
  const half = Math.floor(span / 2)
  const sx = Math.round(props.nx) - half
  const sy = Math.round(props.ny) - half

  ctx.imageSmoothingEnabled = false
  ctx.clearRect(0, 0, props.size, props.size)
  // 棋盘底 (源越界处透出, 提示在图外).
  ctx.fillStyle = '#18181b'
  ctx.fillRect(0, 0, props.size, props.size)
  // drawImage 越界区域自动裁掉, 不画 → 露底.
  ctx.drawImage(props.source, sx, sy, span, span, 0, 0, props.size, props.size)

  // 中心像素方框 (当前取到的那个像素).
  const c = half * mag
  ctx.strokeStyle = 'rgba(255,255,255,0.9)'
  ctx.lineWidth = 1
  ctx.strokeRect(c + 0.5, c + 0.5, mag - 1, mag - 1)
  ctx.strokeStyle = 'rgba(0,0,0,0.6)'
  ctx.strokeRect(c - 0.5, c - 0.5, mag + 1, mag + 1)
}

watch(() => [props.nx, props.ny, props.source], draw)
onMounted(draw)
</script>
