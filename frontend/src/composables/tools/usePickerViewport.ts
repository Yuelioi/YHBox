// usePickerViewport — 截图选择器的图区缩放/平移 (zoom + pan) 状态机.
// 内部 box 按原生像素 1:1 布局, 再用 CSS transform (translate+scale, origin 0,0) 显示;
// 选区/覆盖层用原生坐标定位, 随同一 transform 缩放 → 对齐天然保证.
//   - 滚轮: 对准光标缩放 (光标下那个原生点保持不动)
//   - 平移: 调 offset (空格拖 / 中键拖, 由 view 决定何时触发)
//   - screenToNative: 鼠标 client 坐标 → 原生像素 (选点/拉框统一入口)
import { computed, ref } from 'vue'

const MIN_ZOOM = 0.05
const MAX_ZOOM = 32

function clamp(z: number) {
  return Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, z))
}

export function usePickerViewport(getViewportEl: () => HTMLElement | null) {
  const zoom = ref(1)
  const offset = ref({ x: 0, y: 0 })
  const panning = ref(false)
  const spaceHeld = ref(false) // 空格按住 = 进入平移模式 (左键拖动改成平移)

  const transformStyle = computed(() => ({
    transform: `translate(${offset.value.x}px, ${offset.value.y}px) scale(${zoom.value})`,
    transformOrigin: '0 0',
  }))

  function viewportSize() {
    const el = getViewportEl()
    return { w: el?.clientWidth ?? 0, h: el?.clientHeight ?? 0 }
  }

  // 鼠标 client 坐标 → 原生像素 (可能落在图外, 由调用方夹取).
  function screenToNative(clientX: number, clientY: number) {
    const el = getViewportEl()
    if (!el) return { x: 0, y: 0 }
    const r = el.getBoundingClientRect()
    return {
      x: (clientX - r.left - offset.value.x) / zoom.value,
      y: (clientY - r.top - offset.value.y) / zoom.value,
    }
  }

  // 对准 client 坐标缩放: 缩放后该位置下的原生点保持不动.
  function zoomAt(clientX: number, clientY: number, factor: number) {
    const el = getViewportEl()
    if (!el) return
    const r = el.getBoundingClientRect()
    const z1 = zoom.value
    const z2 = clamp(z1 * factor)
    if (z2 === z1) return
    const localX = clientX - r.left
    const localY = clientY - r.top
    const nx = (localX - offset.value.x) / z1
    const ny = (localY - offset.value.y) / z1
    zoom.value = z2
    offset.value = { x: localX - nx * z2, y: localY - ny * z2 }
  }

  function onWheel(e: WheelEvent) {
    e.preventDefault()
    zoomAt(e.clientX, e.clientY, e.deltaY < 0 ? 1.1 : 1 / 1.1)
  }

  // fit: 适应窗口 (整图可见, 居中). actualSize: 1:1 原始像素 (居中, 超出可平移).
  function fit(natW: number, natH: number) {
    const { w, h } = viewportSize()
    if (!natW || !natH || !w || !h) return
    const z = clamp(Math.min(w / natW, h / natH))
    zoom.value = z
    offset.value = { x: (w - natW * z) / 2, y: (h - natH * z) / 2 }
  }
  function actualSize(natW: number, natH: number) {
    const { w, h } = viewportSize()
    zoom.value = 1
    offset.value = { x: (w - natW) / 2, y: (h - natH) / 2 }
  }

  // 平移拖动: view 在 pointerdown 判定该平移时调 beginPan, move 调 movePan, up 调 endPan.
  let panStart = { x: 0, y: 0 }
  let offStart = { x: 0, y: 0 }
  function beginPan(clientX: number, clientY: number) {
    panning.value = true
    panStart = { x: clientX, y: clientY }
    offStart = { ...offset.value }
  }
  function movePan(clientX: number, clientY: number) {
    if (!panning.value) return
    offset.value = {
      x: offStart.x + (clientX - panStart.x),
      y: offStart.y + (clientY - panStart.y),
    }
  }
  function endPan() {
    panning.value = false
  }

  function onKeyDown(e: KeyboardEvent) {
    if (e.code === 'Space') spaceHeld.value = true
  }
  function onKeyUp(e: KeyboardEvent) {
    if (e.code === 'Space') spaceHeld.value = false
  }

  return {
    zoom,
    offset,
    panning,
    spaceHeld,
    transformStyle,
    screenToNative,
    zoomAt,
    onWheel,
    fit,
    actualSize,
    beginPan,
    movePan,
    endPan,
    onKeyDown,
    onKeyUp,
    MIN_ZOOM,
    MAX_ZOOM,
  }
}
