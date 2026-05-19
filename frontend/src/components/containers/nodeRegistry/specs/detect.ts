// frontend/src/components/containers/nodeRegistry/specs/detect.ts
// Detection / template / color / screenshot nodes (7).
// Mirrors backend internal/services/container/nodekind/specs/template.go.
import { register } from '../registry'

register({
  kind: 'WaitTemplate',
  group: 'detect',
  labelZh: '等待图像',
  description:
    '周期截屏检测模板图像。命中 → found 出口（产 $sys.lastTemplate.point）；超时 → timeout 出口。',
  visual: { icon: 'i-tabler-eye', bg: 'bg-violet-500/15', border: 'border-violet-500/40' },
  execIn: ['in'],
  execOut: ['found', 'timeout'],
  dataIn: { timeoutMs: 'number', threshold: 'number' },
  dataOut: { point: 'point' },
  fields: [{ key: 'template', label: '模板', type: 'template-picker' }],
  defaults: { template: '', literal: { timeoutMs: 5000, threshold: 0.85 } },
})

register({
  kind: 'CheckTemplate',
  group: 'detect',
  labelZh: '试探图像',
  description: '单次检测，立即分支：命中走 yes，否则走 no。不阻塞等待。',
  visual: { icon: 'i-tabler-search', bg: 'bg-violet-500/15', border: 'border-violet-500/40' },
  execIn: ['in'],
  execOut: ['yes', 'no'],
  dataIn: { threshold: 'number' },
  dataOut: { point: 'point' },
  fields: [{ key: 'template', label: '模板', type: 'template-picker' }],
  defaults: { template: '', literal: { threshold: 0.85 } },
})

register({
  kind: 'ClickTemplate',
  group: 'detect',
  labelZh: '点击图像',
  description: '检测到模板图像后点击其中心。命中并点完走 done，超时走 timeout。',
  visual: { icon: 'i-tabler-target', bg: 'bg-violet-500/15', border: 'border-violet-500/40' },
  execIn: ['in'],
  execOut: ['done', 'timeout'],
  dataIn: { timeoutMs: 'number', threshold: 'number' },
  dataOut: { point: 'point' },
  fields: [
    { key: 'template', label: '模板', type: 'template-picker' },
    {
      key: 'button',
      label: '鼠标按键',
      type: 'select',
      options: [
        { value: 'left', label: '左键' },
        { value: 'middle', label: '中键' },
        { value: 'right', label: '右键' },
      ],
    },
  ],
  defaults: { template: '', button: 'left', literal: { timeoutMs: 5000, threshold: 0.85 } },
})

register({
  kind: 'DetectColor',
  group: 'detect',
  labelZh: '检测颜色',
  description:
    'ROI 内统计落在 HSV/RGB 范围内的像素。≥ minPixels 走 yes，否则 no。结果写 $sys.lastColor.{count, cx, cy}。',
  visual: { icon: 'i-tabler-color-picker', bg: 'bg-cyan-500/15', border: 'border-cyan-500/40' },
  execIn: ['in'],
  execOut: ['yes', 'no'],
  dataIn: { minPixels: 'number' },
  dataOut: { point: 'point' },
  fields: [
    {
      key: 'region',
      label: 'ROI (x,y,w,h 比例)',
      type: 'text',
      placeholder: '0.4,0.55,0.2,0.05',
      hint: '客户区比例 0..1, 逗号分隔. 留空 = 全屏',
    },
    {
      key: 'mode',
      label: '颜色模式',
      type: 'select',
      options: [
        { value: 'hsv', label: 'HSV (色相饱和明度)' },
        { value: 'rgb', label: 'RGB (红绿蓝)' },
      ],
    },
    {
      key: 'range',
      label: '颜色范围 (6 个值 CSV)',
      type: 'text',
      placeholder: 'HSV: 50,60,67,127,253,255',
      hint: 'HSV: hMin,hMax,sMin,sMax,vMin,vMax  (H: 0-360, S/V: 0-255). RGB: rMin,rMax,gMin,gMax,bMin,bMax (0-255)',
    },
  ],
  defaults: {
    region: '0.4,0.55,0.2,0.05',
    mode: 'hsv',
    range: '50,60,67,127,253,255',
    literal: { minPixels: 5 },
  },
})

register({
  kind: 'DetectColorHSV',
  group: 'detect',
  labelZh: 'HSV 颜色检测',
  description:
    '按 HSV 范围在 ROI 中扫描像素, 命中比例 ≥ minPixelRatio 走 yes, 否则下次 poll; 超时走 timeout. 输出 pixelCount / pixelRatio.',
  visual: { icon: 'i-tabler-palette', bg: 'bg-cyan-500/15', border: 'border-cyan-500/40' },
  execIn: ['in'],
  execOut: ['yes', 'no', 'timeout'],
  dataIn: { minPixelRatio: 'number', pollIntervalMs: 'number', timeoutMs: 'number' },
  dataOut: { pixelCount: 'number', pixelRatio: 'number' },
  fields: [],
  defaults: {
    roi: { x: 0, y: 0, w: 100, h: 100 },
    hsv: { hMin: 0, hMax: 180, sMin: 0, sMax: 255, vMin: 0, vMax: 255 },
    literal: { minPixelRatio: 0.05, pollIntervalMs: 100, timeoutMs: 5000 },
  },
})

register({
  kind: 'ROIColorScan',
  group: 'detect',
  labelZh: 'ROI 颜色扫描',
  description:
    'ROI 内沿 scanAxis 聚簇扫描色块, 输出 clusters 数组 (含 startPx/endPx/centerPx/pxCount). 用于溜鱼方向判断等场景.',
  visual: { icon: 'i-tabler-scan-eye', bg: 'bg-cyan-500/15', border: 'border-cyan-500/40' },
  execIn: ['in'],
  execOut: ['found', 'notFound', 'timeout'],
  dataIn: {
    minClusterPx: 'number',
    maxClusterPx: 'number',
    minClusterCount: 'number',
    pollIntervalMs: 'number',
    timeoutMs: 'number',
  },
  dataOut: { clusters: 'any', clusterCount: 'number' },
  fields: [],
  defaults: {
    roi: { x: 0, y: 0, w: 100, h: 100 },
    hsv: { hMin: 0, hMax: 180, sMin: 0, sMax: 255, vMin: 0, vMax: 255 },
    scanAxis: 'x',
    literal: {
      minClusterPx: 2,
      maxClusterPx: 0,
      minClusterCount: 1,
      pollIntervalMs: 100,
      timeoutMs: 5000,
    },
  },
})

register({
  kind: 'Screenshot',
  group: 'detect',
  labelZh: '截图存盘',
  description:
    '抓全帧或 ROI 帧, 按 pathTemplate 存 PNG (限定 bin/data/screenshots/ 下). 占位符: {ts}/{containerId}/{nodeId}/{date}.',
  visual: { icon: 'i-tabler-camera', bg: 'bg-violet-500/15', border: 'border-violet-500/40' },
  execIn: ['in'],
  execOut: ['done'],
  dataIn: {},
  dataOut: { path: 'string' },
  fields: [],
  defaults: { pathTemplate: 'screenshots/{ts}.png' },
})
