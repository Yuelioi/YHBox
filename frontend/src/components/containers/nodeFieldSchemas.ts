// NodeInspector form field 静态 schema. 每个 kind 列出 *配置字段* (跟 data-in pin 区分):
// - 配置字段 (mode/scope/level/template/vk/varName/color 等) — 不走 data-in pin, 在这里渲染.
// - 数值/字符串 thresholds (durationMs/threshold/timeoutMs/x/y/message 等) — 走 data-in pin,
//   Inspector 的 "数据输入 (literal)" 段自动渲染 (读 PIN_SPECS[kind].dataIn). 这里 NOT 列出.
//
// v4 移除: 所有 type: 'expr' 字段都被搬到 data-in pin. 用户在 "数据输入 (literal)" 段编辑.
// (历史 v3 schema 列了 expr 字段, 但 runtime 已经不读 config.message / config.durationMs 等,
//  只读 config.literal.<pin>. 同时存在两个会让用户编辑错地方 — 见 image copy 14 bug.)

export interface Field {
  key: string
  label: string
  type:
    | 'select'
    | 'text'
    | 'action-picker'
    | 'template-picker'
    | 'key-capture'
    | 'var-name-select' // v4: dropdown of Container.Vars (NodeInspector reads editor store)
  options?: { label: string; value: string }[]
  placeholder?: string
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
  // v4: Sleep durationMs 走 data-in pin. 这里没有配置字段.
  Sleep: [],
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
      hint: 'count → 读 data-in pin "count"; while → 读 data-in pin "condition"; forever → 都不读',
    },
  ],
  // v4: If.condition 走 data-in pin.
  If: [],
  // v4: Parallel/Race.n 走 data-in pin.
  Parallel: [],
  Race: [],
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
  // v4 GetSys: 选 sys path (与 runtime SysPathSchema 同步; 改 schema 时这里也要更新).
  GetSys: [
    {
      key: 'path',
      label: '系统路径',
      type: 'select',
      options: [
        { label: 'runId (number)', value: 'runId' },
        { label: 'iter (number) — Loop 当前 iter', value: 'iter' },
        { label: 'winnerIdx (number) — Race 获胜分支', value: 'winnerIdx' },
        { label: 'lastTemplate.found (bool)', value: 'lastTemplate.found' },
        { label: 'lastTemplate.point (point)', value: 'lastTemplate.point' },
        { label: 'lastTemplate.point.x (number)', value: 'lastTemplate.point.x' },
        { label: 'lastTemplate.point.y (number)', value: 'lastTemplate.point.y' },
        { label: 'lastTemplate.region (any)', value: 'lastTemplate.region' },
        { label: 'lastColor.count (number)', value: 'lastColor.count' },
        { label: 'lastColor.cx (number)', value: 'lastColor.cx' },
        { label: 'lastColor.cy (number)', value: 'lastColor.cy' },
        { label: 'lastColor.center (point)', value: 'lastColor.center' },
        { label: 'lastColor.center.x (number)', value: 'lastColor.center.x' },
        { label: 'lastColor.center.y (number)', value: 'lastColor.center.y' },
        { label: 'lastStopwatch.elapsedMs (number)', value: 'lastStopwatch.elapsedMs' },
        { label: 'lastTry.errorMsg (string)', value: 'lastTry.errorMsg' },
        { label: 'lastDetect.pixelCount (number)', value: 'lastDetect.pixelCount' },
        { label: 'lastDetect.pixelRatio (number)', value: 'lastDetect.pixelRatio' },
        { label: 'lastROIScan.clusterCount (number)', value: 'lastROIScan.clusterCount' },
        { label: 'lastROIScan.clusters (any)', value: 'lastROIScan.clusters' },
        { label: 'lastScreenshot.path (string)', value: 'lastScreenshot.path' },
      ],
      hint: '运行时 sys 状态 — 同 tick 内一致 (per-tick snapshot, spec §10.3)',
    },
  ],
  // v4 GetParam: 子图入参; 候选 = 当前所在子图的 inputParams (后续 wire to store).
  GetParam: [
    { key: 'paramName', label: '参数名 (paramName)', type: 'text', placeholder: '当前子图 inputParams 里的 name' },
  ],
  // v4 CommentBox (§9.1): pure-UI grouping. label/color/size 都是 config (不走 pin — 不是数据流, 只是渲染)
  CommentBox: [
    { key: 'label', label: '标题', type: 'text', placeholder: '注释' },
    { key: 'color', label: '颜色 (hex)', type: 'text', placeholder: '#fbbf24' },
    { key: 'width', label: '宽度 (px)', type: 'text', placeholder: '200', hint: '纯数字' },
    { key: 'height', label: '高度 (px)', type: 'text', placeholder: '150', hint: '纯数字' },
  ],
  // v4 Expr: expr 字符串 + outType + inputs 数组. inputs 单独 UI (ExprInspector 后续 dedicated).
  Expr: [
    { key: 'expr', label: '表达式', type: 'text', placeholder: 'i + 1', hint: '只引用 inputs 中声明的 identifier (无 $ 前缀)' },
    {
      key: 'outType',
      label: '输出类型',
      type: 'select',
      options: [
        { label: 'auto (默认, 推断)', value: 'auto' },
        { label: 'number', value: 'number' },
        { label: 'bool', value: 'bool' },
        { label: 'string', value: 'string' },
        { label: 'point', value: 'point' },
        { label: 'any', value: 'any' },
      ],
    },
    // inputs[] 暂无专门 UI — 用户手动 JSON 编辑或后续 Phase D ExprInspector.
  ],
  // v4: 模板节点 — template/button 还是 config; timeoutMs/threshold 走 data-in pin.
  WaitTemplate: [
    { key: 'template', label: '模板', type: 'template-picker' },
  ],
  CheckTemplate: [
    { key: 'template', label: '模板', type: 'template-picker' },
  ],
  ClickTemplate: [
    { key: 'template', label: '模板', type: 'template-picker' },
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
  // DetectColor (legacy v3 CSV configs). Threshold via pin; region/mode/range 仍是 CSV config.
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
  ],
  // v4: ClickAt button 还是 config; xRatio/yRatio/durationMs 走 data-in pin.
  ClickAt: [
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
  ],
  // v4: MouseMoveRel — dx/dy/durationMs 走 data-in pin, 这里无 config.
  MouseMoveRel: [],
  // v4: Scroll — 全 data-in pin.
  Scroll: [],
  // v4 OnEvent: kind/template/retriggerPolicy 是 config (枚举/picker); 数值走 pin.
  OnEvent: [
    {
      key: 'kind',
      label: '事件类型',
      type: 'select',
      options: [{ label: '模板出现 (template_appeared)', value: 'template_appeared' }],
    },
    { key: 'template', label: '模板', type: 'template-picker' },
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
  ],
  // v4 Log/Toast: level/color 是 config; message/title 走 data-in pin.
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
  ],
  Toast: [
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
