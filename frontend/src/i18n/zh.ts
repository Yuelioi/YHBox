// 中文文案。键命名约定：
//   sidebar.<bot>       侧栏每个项的 label
//   controls.<state>    BotControls 按钮 / 状态
//   view.<bot>.<key>    各 view 内部文案
//   settings.<key>      设置面板
//   common.<key>        共用通用文案
//
// 用 ts 而不是 yaml 是因为 unplugin-vue-i18n 的 yaml 输出格式跟 vue-i18n 9
// runtime 不兼容会让 t() 触发 SyntaxError. 直接 ts 给 createI18n 喂 plain
// string，runtime 自己 compile，所有 unicode 字符都 OK.

export default {
  sidebar: {
    automation: 'Automation',
    tools: 'Tools',
    collapse: '收起',
    fish: '钓鱼',
    cook: '店长',
    piano: '弹琴',
    battle: '战斗',
    rhythm: '音游',
    actions: '动作',
    tasks: '任务',
    schedules: '计划',
    settings: '设置',
    help: '帮助',
    about: '关于',
  },
  controls: {
    start: '开始',
    pause: '暂停',
    resume: '继续',
    stop: '停止',
    state: {
      idle: '空闲',
      running: '运行中',
      paused: '已暂停',
    },
  },
  settings: {
    language: '语言',
    language_zh: '中文',
    language_en: 'English',
    language_restart_hint: '切换语言后部分内容需重启 exe 生效，包括模板和配置',
    language_changed_title: '已切换语言',
    language_changed_desc: 'UI 已立即生效，模板和配置切换需要重启程序',
  },
  common: {
    game_not_detected: '未检测到异环窗口',
  },
  // v4 (spec §12): backend ValidationError.Code → user-facing message.
  // Params: {param} 占位符走 vue-i18n named-interpolation. 未匹配 → fallback 显示 raw code.
  error: {
    // v3 existing
    NO_START: '主图没有 Start 节点',
    MULTIPLE_STARTS: '主图有 {count} 个 Start 节点（应恰好 1 个）',
    DANGLING_EDGE: '边 {from} → {to} 引用了不存在的节点 ({missing})',
    INVALID_PIN: '节点 {nodeID} ({kind}) 不存在 {side} pin {pin}',
    DUPLICATE_OUTPUT_PIN: 'OutputPin Name {name} 重复',
    MISSING_TEMPLATE: '节点 {nodeID} 引用的模板 {template} 在容器 templates/ 里找不到',
    MISSING_SUBGRAPH: 'Subgraph 节点引用了不存在的子图 {subgraphId}',
    MISSING_MOUSE_CALIBRATION: '使用了 MouseMoveRel 但容器没有 MouseCalibration 节点',
    DUPLICATE_MOUSE_CALIBRATION: '主图有 {count} 个 MouseCalibration 节点（应最多 1 个）',
    MOUSE_CALIBRATION_NOT_SET: 'MouseCalibration 的 counts360 没设置',
    MOUSE_CALIBRATION_FOREIGN: 'MouseCalibration 跟本机 settings 不一致',
    MOUSE_CALIBRATION_IN_SUBGRAPH: 'MouseCalibration 节点必须在主图',
    EMPTY_SUBGRAPH_OUTPUT: '子图没有任何 SubgraphOutput 节点',
    CYCLIC_SUBGRAPH_DEPENDENCY: '子图调用形成环',
    PLAYCLIP_NO_CLIP_ID: 'PlayClip 节点没指定 clipID',
    MISSING_WINDOW_TARGET: '主图缺 WindowTarget 节点',
    DUPLICATE_WINDOW_TARGET: '主图有多个 WindowTarget（应恰好 1 个）',
    WINDOW_TARGET_IN_SUBGRAPH: 'WindowTarget 节点必须在主图',
    INVALID_WINDOW_TARGET_REGEX: 'WindowTarget 正则不合法',
    INVALID_WINDOW_TARGET_EMPTY_MATCH: 'WindowTarget match 不能为空',
    INVALID_ROI: 'ROI 配置不合法',
    INVALID_HSV_RANGE: 'HSV 范围不合法',
    INVALID_SCAN_AXIS: 'scanAxis 必须是 x 或 y',
    INVALID_CLUSTER_RANGE: 'cluster 范围不合法',
    INVALID_VK: 'vk (虚拟键码) 不合法',
    INVALID_MOUSE_BUTTON: 'button 必须是 left/right/middle',
    UNSAFE_SCREENSHOT_PATH: 'Screenshot pathTemplate 路径不安全',
    POLL_TOO_FAST: '轮询间隔太小 (<{minMs}ms), 会让 CPU 飙升',
    STOPWATCH_EMPTY_KEY: 'Stopwatch key 不能为空',
    STOPWATCH_KEY_MISMATCH: '{kind} 使用 key {key} 但同图中无对应 StopwatchStart',
    THROW_IN_MAIN_GRAPH: 'Throw 节点不能在主图 (必须在子图内)',
    INVALID_SWITCH_CASES: 'Switch cases 不合法 (空 / 重复 / 含 . / 名 default)',
    // v4 新增
    PIN_TYPE_MISMATCH: 'data pin 类型不兼容: {from} → {to} (edge: {edge})',
    PIN_TYPE_COERCION_WARNING: 'data pin 隐式类型转换: {from} → {to} (建议加 To* 节点显式)',
    GETVAR_UNKNOWN_VAR: 'GetVar/SetVar/IncVar 引用未声明的变量 {name}',
    GETVAR_TYPE_MISMATCH: 'GetVar 出的 type 跟下游期望不符',
    LITERAL_TYPE_MISMATCH: 'inline literal 类型跟 pin type 不符 (pin {pin} 期望 {type}, 实际 {got})',
    DATA_PIN_DANGLING: 'data-in pin {pin} 未连边也无 literal',
    DATA_GRAPH_CYCLE: '数据流形成环: {cycle}',
    EXPR_PARSE_ERROR: 'Expr 解析失败: {error}',
    EXPR_UNKNOWN_INPUT: 'Expr 引用未声明的输入 {name}',
    EXPR_TYPE_MISMATCH: 'Expr outType 与推断不符 (期望 {expected}, 实际 {actual})',
    EXPR_DUPLICATE_INPUT: 'Expr inputs 重复名: {name}',
    GETSYS_UNKNOWN_PATH: 'GetSys 路径 {path} 不在 sys schema 中',
    GETPARAM_UNKNOWN_PARAM: 'GetParam 引用未声明入参 {name}',
    COLLAPSED_PIN_BROKEN: 'CollapsedNode 外部 pin 在后备 Subgraph 中找不到 marker',
    COLLAPSED_REFERENCED_BY_SUBGRAPH_CALL: 'Subgraph 节点引用了 isAnonymous Subgraph (该子图属 CollapsedNode, 不可跨 graph 复用)',
  },
}
