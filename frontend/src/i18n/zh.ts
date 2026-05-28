// 中文文案. 单源 of truth (en.ts 是平行翻译).
// 键命名约定 (扁平 namespace, 数据驱动 chrome):
//   sidebar.<k>            侧栏 label
//   controls.<k>           BotControls 按钮 / 状态
//   common.<k>             通用 (确认/取消/保存/关闭/加载中)
//   settings.<k>           Settings* view 内文案
//   editor.<k>             编辑器主壳 (P2)
//   editor.canvas.<k>      画布操作提示 (P2)
//   editor.menu.<k>        右键菜单 item (P3)
//   editor.palette.<k>     CommandPalette item (P3)
//   editor.inspector.<k>   Inspector 表单 label/hint (P3/P4)
//   editor.snippet.<k>     SnippetsPanel / SaveSnippetDrawer (P3)
//   log.<k>                LogPanel header / filter / settings popover
//   status.<k>             AppStatusBar 状态
//   hotkeys.<k>            SettingsHotkeys group label / 状态文案
//   hotkeys.label.<k>      HotkeyEntry.Label 翻译键 (backend 填 key, FE t() 渲染)
//   dialog.<k>             ConfirmDialog / Modal title/body (P3)
//   toast.<k>              所有 toast.add({title,description})
//   error.<k>              ValidationError.Code → user message
//   validation.<k>         ValidationErrorPanel 壳文案
//
// 用 ts 而不是 yaml: unplugin-vue-i18n 的 yaml 输出格式跟 vue-i18n 9 runtime
// 不兼容会 throw SyntaxError. ts 给 createI18n 喂 plain object, runtime 自己
// compile, 所有 unicode 字符都 OK.

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
  // backend ValidationError.Code → user-facing message.
  // Params: {param} 占位符走 vue-i18n named-interpolation. 未匹配 → fallback 显示 raw code.
  error: {
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
    MOUSE_CALIBRATION_FOREIGN: 'MouseCalibration 节点值 {nodeValue} 跟本机 ({settingsValue}) 不一致, 疑似从别人机器下载',
    MOUSE_CALIBRATION_IN_SUBGRAPH: 'MouseCalibration 节点必须在主图',
    EMPTY_SUBGRAPH_OUTPUT: '子图没有任何 SubgraphOutput 节点',
    CYCLIC_SUBGRAPH_DEPENDENCY: '子图调用形成环',
    PLAYCLIP_NO_CLIP_ID: 'PlayClip 节点没指定 clipID',
    MISSING_WINDOW_TARGET: '主图缺 WindowTarget 节点',
    DUPLICATE_WINDOW_TARGET: '主图有多个 WindowTarget（应恰好 1 个）',
    WINDOW_TARGET_IN_SUBGRAPH: 'WindowTarget 节点必须在主图',
    INVALID_WINDOW_TARGET_REGEX: 'WindowTarget 正则不合法: {error}',
    INVALID_WINDOW_TARGET_EMPTY_MATCH: 'WindowTarget match 不能为空',
    INVALID_ROI: 'ROI 配置不合法 (w/h 必须 >= 1, 得到 {w}x{h})',
    INVALID_DUALBAR_ROIS: 'DualColorBarTrack rois 配置无效',
    DUPLICATE_DUALBAR_ROI: 'DualColorBarTrack rois 含重复分辨率 ({w}x{h})',
    INVALID_HSV_RANGE: 'HSV 范围不合法',
    INVALID_SCAN_AXIS: 'scanAxis 必须是 x 或 y, 得到 {got}',
    INVALID_CLUSTER_RANGE: 'cluster 范围不合法 (min={min} > max={max})',
    INVALID_VK: 'vk (虚拟键码) 不合法',
    INVALID_MOUSE_BUTTON: 'button 必须是 left/right/middle, 得到 {button}',
    UNSAFE_SCREENSHOT_PATH: 'Screenshot pathTemplate 路径不安全',
    POLL_TOO_FAST: '轮询间隔太小 (<{minMs}ms), 会让 CPU 飙升',
    STOPWATCH_EMPTY_KEY: 'Stopwatch key 不能为空',
    STOPWATCH_KEY_MISMATCH: '{kind} 使用 key {key} 但同图中无对应 StopwatchStart',
    THROW_IN_MAIN_GRAPH: 'Throw 节点不能在主图 (必须在子图内)',
    INVALID_SWITCH_CASES: 'Switch cases 不合法 (空 / 重复 / 含 . / 名 default)',
    INVALID_CRON_EXPR: 'Cron 表达式无效: {expr} ({parseErr})',
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
    // B11/B3+ 加的 code
    INVALID_VAR_REF: '节点引用未声明的容器变量 {varName} (scope={scope})',
    // 禁用节点
    WARN_DISABLED_BRANCH_NODE: '分支/异步节点 {nodeID} (kind={kind}) 禁用走 passthrough exit pin — 行为非确定, 建议删而非禁',
    INVALID_DISABLED_TERMINAL: '容器级节点 {nodeID} (kind={kind}) 不允许禁用 (Start/WindowTarget)',
    // sentinel scope
    BREAK_OUTSIDE_LOOP: 'Break 节点必须在 Loop body 下游 (同图内 exec 可达)',
    CONTINUE_OUTSIDE_LOOP: 'Continue 节点必须在 Loop body 下游 (同图内 exec 可达)',
    THROW_OUTSIDE_TRY: 'Throw 节点必须在被 Try.SubgraphID 引用的子图内',
    // template/clip key validation
    INVALID_TEMPLATE_KEY: '模板 key {key} 不合法: {error}',
    TEMPLATE_NOT_FOUND: '模板 {key} 在容器中找不到',
    CLIP_NOT_FOUND: '剪辑 {id} 在容器中找不到',
    // service.go 兜底
    CONTAINER_NOT_FOUND: '容器 {id} 不存在',
  },
  // ValidationErrorPanel 模态壳文案 — 跟 error.* 错误码分开.
  validation: {
    title_failed: '校验失败 — {errorCount} 错',
    title_failed_with_warns: '校验失败 — {errorCount} 错 / {warnCount} 警告',
    title_passed: '校验通过',
    desc_no_issues: '没有发现校验错误。容器可以试运行。',
    node_label: '节点:',
    close: '关闭',
    run_button: '通过 — 继续运行',
    fix_missing_window_target: '一键添加 WindowTarget 节点',
  },
}
