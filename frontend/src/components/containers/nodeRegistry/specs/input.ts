// frontend/src/components/containers/nodeRegistry/specs/input.ts
// Input nodes (10): ClickAt, KeyPress, MouseMoveRel, Scroll, KeyHoldStart/Stop, MouseHoldStart/Stop, OnEvent, PlayClip.
// Mirrors backend internal/services/container/nodekind/specs/input.go.
import { register } from '../registry'

register({
  kind: 'ClickAt',
  group: 'input',
  labelZh: '点击位置',
  description: '点击客户区比例坐标 (xRatio, yRatio)。durationMs 是按下时长。',
  visual: { icon: 'i-tabler-click', bg: 'bg-orange-500/15', border: 'border-orange-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: { xRatio: 'number', yRatio: 'number', durationMs: 'number' },
  dataOut: {},
  fields: [
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
  defaults: { button: 'left', literal: { xRatio: 0.5, yRatio: 0.5, durationMs: 50 } },
})

register({
  kind: 'KeyPress',
  group: 'input',
  labelZh: '按键',
  description: '按下并松开 vk（虚拟键码字符串如 "W"/"Space"），间隔 durationMs。',
  visual: { icon: 'i-tabler-keyboard', bg: 'bg-orange-500/15', border: 'border-orange-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: { durationMs: 'number' },
  dataOut: {},
  fields: [{ key: 'vk', label: '按键', type: 'key-capture' }],
  defaults: { vk: 'W', literal: { durationMs: 50 } },
})

register({
  kind: 'MouseMoveRel',
  group: 'input',
  labelZh: '相对移动',
  description: '相对当前位置移动 (dx, dy) 像素，移动用时 durationMs。',
  visual: { icon: 'i-tabler-arrows-move', bg: 'bg-orange-500/15', border: 'border-orange-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: { dx: 'number', dy: 'number', durationMs: 'number' },
  dataOut: {},
  fields: [],
  defaults: { literal: { dx: 0, dy: 0, durationMs: 200 } },
})

register({
  kind: 'Scroll',
  group: 'input',
  labelZh: '滚轮',
  description: '在 (xRatio, yRatio) 滚动 delta 格。正数向上，负数向下。',
  visual: { icon: 'i-tabler-mouse', bg: 'bg-orange-500/15', border: 'border-orange-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: { xRatio: 'number', yRatio: 'number', delta: 'number' },
  dataOut: {},
  fields: [],
  defaults: { literal: { xRatio: 0.5, yRatio: 0.5, delta: 3 } },
})

register({
  kind: 'KeyHoldStart',
  group: 'input',
  labelZh: '按下键 (异步)',
  description: '按下虚拟键并立即返回 (异步长按). 配对 KeyHoldStop; 容器终止时 ReleaseAll 兜底.',
  visual: { icon: 'i-tabler-keyboard', bg: 'bg-orange-500/15', border: 'border-orange-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: {},
  dataOut: {},
  fields: [],
  defaults: { vk: 'A' },
})

register({
  kind: 'KeyHoldStop',
  group: 'input',
  labelZh: '松开键',
  description: '松开虚拟键. 跟 KeyHoldStart 配对.',
  visual: { icon: 'i-tabler-keyboard-off', bg: 'bg-orange-500/15', border: 'border-orange-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: {},
  dataOut: {},
  fields: [],
  defaults: { vk: 'A' },
})

register({
  kind: 'MouseHoldStart',
  group: 'input',
  labelZh: '按下鼠标 (异步)',
  description: '在指定客户区坐标 (x, y) 按下鼠标键并立即返回. 配对 MouseHoldStop.',
  visual: { icon: 'i-tabler-hand-click', bg: 'bg-orange-500/15', border: 'border-orange-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: { xRatio: 'number', yRatio: 'number' },
  dataOut: {},
  fields: [],
  defaults: { button: 'left', literal: { xRatio: 0.5, yRatio: 0.5 } },
})

register({
  kind: 'MouseHoldStop',
  group: 'input',
  labelZh: '松开鼠标',
  description: '松开鼠标键 (Backend stateful, 不需位置). 跟 MouseHoldStart 配对.',
  visual: { icon: 'i-tabler-hand-off', bg: 'bg-orange-500/15', border: 'border-orange-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: {},
  dataOut: {},
  fields: [],
  defaults: { button: 'left' },
})

register({
  kind: 'OnEvent',
  group: 'input',
  labelZh: '事件监听',
  description:
    '与 Start 同级的入口节点。容器启动时起 listener；事件命中 spawn 子图。v1 仅支持 template_appeared。',
  visual: { icon: 'i-tabler-radio', bg: 'bg-pink-500/15', border: 'border-pink-500/40' },
  execIn: [],
  execOut: ['out'],
  dataIn: {
    threshold: 'number',
    pollIntervalMs: 'number',
    maxConcurrent: 'number',
    cooldownMs: 'number',
  },
  dataOut: {},
  fields: [
    {
      key: 'kind',
      label: '事件类型',
      type: 'select',
      options: [{ value: 'template_appeared', label: '模板出现 (template_appeared)' }],
    },
    { key: 'template', label: '模板', type: 'template-picker' },
    {
      key: 'retriggerPolicy',
      label: '重触发策略',
      type: 'select',
      options: [
        { value: 'drop', label: '丢弃 (drop)' },
        { value: 'queue', label: '排队 (queue)' },
        { value: 'restart', label: '重启 (restart)' },
      ],
    },
  ],
  defaults: {
    kind: 'template_appeared',
    template: '',
    retriggerPolicy: 'drop',
    literal: { pollIntervalMs: 100, maxConcurrent: 1, cooldownMs: 0 },
  },
})

register({
  kind: 'PlayClip',
  group: 'input',
  labelZh: '播放录制',
  description:
    '播放一段录制的输入事件流 (键鼠 events). config.clipID 指定 clip; config.keepRanges 可选裁剪 [{fromUs, toUs}, ...] 让你跳过录制中的停顿段.',
  visual: { icon: 'i-tabler-vinyl', bg: 'bg-pink-500/15', border: 'border-pink-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: {},
  dataOut: {},
  fields: [],
  defaults: { clipID: '', keepRanges: [] },
})
