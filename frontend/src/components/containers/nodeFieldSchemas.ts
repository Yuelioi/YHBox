// NodeInspector form field 静态 schema. 每个 kind 列出可编辑字段, NodeInspector 按 schema
// 渲染 input. 数据驱动 — 加新 kind 在这里补一条即可, 不需要改 NodeInspector 模板.

export interface Field {
  key: string
  label: string
  type:
    | 'expr'
    | 'select'
    | 'text'
    | 'action-picker'
    | 'template-picker'
    | 'key-capture'
    | 'var-name-select' // v4: dropdown of Container.Vars (NodeInspector reads editor store)
  options?: { label: string; value: string }[]
  placeholder?: string
  exprType?: 'number' | 'bool' | 'string' | 'point'
  hint?: string
}

// v4: shared scope dropdown options for GetVar/SetVar/IncVar.
// SetVar/IncVar default "local"; GetVar default "local" (spec §3.1, DeepSeek 1.1).
const SCOPE_OPTIONS: { label: string; value: string }[] = [
  { label: 'local (默认, 当前 frame 私有)', value: 'local' },
  { label: 'global (容器共享)', value: 'global' },
]
// GetVar 还允许 auto (escape hatch — frame chain → global)
const SCOPE_OPTIONS_WITH_AUTO: { label: string; value: string }[] = [
  ...SCOPE_OPTIONS,
  { label: 'auto (frame chain → global)', value: 'auto' },
]

export const NODE_FIELD_SCHEMAS: Record<string, Field[]> = {
  Sleep: [
    { key: 'durationMs', label: '休眠 ms', type: 'expr', placeholder: '1000', exprType: 'number' },
  ],
  Loop: [
    {
      key: 'mode',
      label: '模式',
      type: 'select',
      options: [
        { label: '固定次数 (count)', value: 'count' },
        { label: '条件循环 (while)', value: 'while' },
        { label: '无限循环 (forever)', value: 'forever' },
      ],
    },
    {
      key: 'count',
      label: '次数(count 模式)',
      type: 'expr',
      placeholder: '10',
      exprType: 'number',
    },
    {
      key: 'condition',
      label: '条件表达式(while 模式)',
      type: 'expr',
      placeholder: '$vars.x < 5',
      exprType: 'bool',
    },
  ],
  If: [{ key: 'condition', label: '条件表达式', type: 'expr', exprType: 'bool' }],
  Parallel: [
    { key: 'n', label: '分支数 (2-8)', type: 'expr', placeholder: '2', exprType: 'number' },
  ],
  Race: [{ key: 'n', label: '分支数 (2-8)', type: 'expr', placeholder: '2', exprType: 'number' }],
  Stop: [{ key: 'error', label: '错误信息(可选)', type: 'text' }],
  // v4: SetVar/IncVar value/delta moved to data-in pins (画布上 inline literal or 连边).
  // 只保留 varName + scope. 详见 spec §3.2 / §3.3.
  SetVar: [
    { key: 'varName', label: '变量名', type: 'var-name-select' },
    { key: 'scope', label: '作用域', type: 'select', options: SCOPE_OPTIONS, hint: 'local=frame 私有 / global=容器共享' },
  ],
  IncVar: [
    { key: 'varName', label: '变量名', type: 'var-name-select' },
    { key: 'scope', label: '作用域', type: 'select', options: SCOPE_OPTIONS, hint: 'local=frame 私有 / global=容器共享' },
  ],
  // v4 新增: GetVar 纯数据节点, 把变量值通过 data-out pin 暴露给下游.
  GetVar: [
    { key: 'varName', label: '变量名', type: 'var-name-select' },
    { key: 'scope', label: '作用域', type: 'select', options: SCOPE_OPTIONS_WITH_AUTO, hint: '默认 local; 想跨 frame 读 global; auto = frame chain → global' },
  ],
  WaitTemplate: [
    { key: 'template', label: '模板', type: 'template-picker' },
    { key: 'timeoutMs', label: '超时 ms', type: 'expr', placeholder: '5000', exprType: 'number' },
    {
      key: 'threshold',
      label: '匹配阈值',
      type: 'expr',
      placeholder: '0.85',
      exprType: 'number',
      hint: '0..1, 越大越严格',
    },
  ],
  CheckTemplate: [
    { key: 'template', label: '模板', type: 'template-picker' },
    {
      key: 'threshold',
      label: '匹配阈值',
      type: 'expr',
      placeholder: '0.85',
      exprType: 'number',
      hint: '0..1, 越大越严格',
    },
  ],
  ClickTemplate: [
    { key: 'template', label: '模板', type: 'template-picker' },
    { key: 'timeoutMs', label: '超时 ms', type: 'expr', placeholder: '5000', exprType: 'number' },
    { key: 'threshold', label: '匹配阈值', type: 'expr', placeholder: '0.85', exprType: 'number' },
    {
      key: 'button',
      label: '鼠标按键',
      type: 'select',
      options: [
        { label: '左键', value: 'left' },
        { label: '中键', value: 'middle' },
        { label: '右键', value: 'right' },
      ],
    },
  ],
  DetectColor: [
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
        { label: 'HSV (色相饱和明度)', value: 'hsv' },
        { label: 'RGB (红绿蓝)', value: 'rgb' },
      ],
    },
    {
      key: 'range',
      label: '颜色范围 (6 个值 CSV)',
      type: 'text',
      placeholder: 'HSV: 50,60,67,127,253,255',
      hint: 'HSV: hMin,hMax,sMin,sMax,vMin,vMax  (H: 0-360, S/V: 0-255). RGB: rMin,rMax,gMin,gMax,bMin,bMax (0-255)',
    },
    {
      key: 'minPixels',
      label: '最少命中像素',
      type: 'expr',
      placeholder: '5',
      exprType: 'number',
      hint: '≥ 此值走 yes 出口; 否则 no',
    },
  ],
  ClickAt: [
    {
      key: 'xRatio',
      label: 'X 比例',
      type: 'expr',
      placeholder: '0.5',
      exprType: 'number',
      hint: '0..1, 0=左 1=右; 1280×720 屏上 0.5 = 中心 640px',
    },
    {
      key: 'yRatio',
      label: 'Y 比例',
      type: 'expr',
      placeholder: '0.5',
      exprType: 'number',
      hint: '0..1, 0=顶 1=底',
    },
    {
      key: 'durationMs',
      label: '按下时长 ms',
      type: 'expr',
      placeholder: '50',
      exprType: 'number',
    },
    {
      key: 'button',
      label: '鼠标按键',
      type: 'select',
      options: [
        { label: '左键', value: 'left' },
        { label: '中键', value: 'middle' },
        { label: '右键', value: 'right' },
      ],
    },
  ],
  KeyPress: [
    { key: 'vk', label: '按键', type: 'key-capture', hint: '点输入框后按下任意键自动捕获' },
    {
      key: 'durationMs',
      label: '按下时长 ms',
      type: 'expr',
      placeholder: '50',
      exprType: 'number',
    },
  ],
  MouseMoveRel: [
    { key: 'dx', label: 'dx (像素)', type: 'expr', exprType: 'number' },
    { key: 'dy', label: 'dy (像素)', type: 'expr', exprType: 'number' },
    {
      key: 'durationMs',
      label: '移动用时 ms',
      type: 'expr',
      placeholder: '200',
      exprType: 'number',
    },
  ],
  Scroll: [
    { key: 'xRatio', label: 'X 比例', type: 'expr', exprType: 'number' },
    { key: 'yRatio', label: 'Y 比例', type: 'expr', exprType: 'number' },
    { key: 'delta', label: '滚动格数', type: 'expr', placeholder: '3', exprType: 'number' },
  ],
  OnEvent: [
    {
      key: 'kind',
      label: '事件类型',
      type: 'select',
      options: [{ label: '模板出现 (template_appeared)', value: 'template_appeared' }],
    },
    { key: 'template', label: '模板', type: 'template-picker' },
    {
      key: 'pollIntervalMs',
      label: '轮询间隔 ms',
      type: 'expr',
      placeholder: '100',
      exprType: 'number',
    },
    {
      key: 'maxConcurrent',
      label: '最大并发子图数',
      type: 'expr',
      placeholder: '1',
      exprType: 'number',
    },
    {
      key: 'retriggerPolicy',
      label: '重触发策略',
      type: 'select',
      options: [
        { label: '丢弃 (drop)', value: 'drop' },
        { label: '排队 (queue)', value: 'queue' },
        { label: '重启 (restart)', value: 'restart' },
      ],
    },
    { key: 'cooldownMs', label: '冷却 ms', type: 'expr', placeholder: '0', exprType: 'number' },
  ],
  Log: [
    {
      key: 'level',
      label: '级别',
      type: 'select',
      options: [
        { label: 'info', value: 'info' },
        { label: 'warn', value: 'warn' },
        { label: 'error', value: 'error' },
      ],
    },
    { key: 'message', label: '消息(表达式)', type: 'expr', exprType: 'string' },
  ],
  Toast: [
    { key: 'title', label: '标题(表达式)', type: 'expr', exprType: 'string' },
    { key: 'message', label: '内容(表达式)', type: 'expr', exprType: 'string' },
    {
      key: 'color',
      label: '颜色',
      type: 'select',
      options: [
        { label: '中性 (neutral)', value: 'neutral' },
        { label: '主色 (primary)', value: 'primary' },
        { label: '成功 (success)', value: 'success' },
        { label: '警告 (warning)', value: 'warning' },
        { label: '错误 (error)', value: 'error' },
      ],
    },
  ],
}
