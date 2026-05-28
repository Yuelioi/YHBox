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
    startup: {
      section_title: '启动与关闭',
      autostart_label: '开机自启',
      autostart_hint:
        '登录 Windows 后自动启动 YHBox. 写入注册表 HKCU\\...\\Run\\YHBox, 不需要管理员权限.',
      tray_label: '关闭最小化到托盘',
      tray_hint: '点关闭按钮(×)时不退出, 而是收到右下角系统托盘. 右键托盘图标可强制退出.',
    },
    capture: {
      section_title: '截屏方式',
      hint_auto: 'Auto: Win11/Server 2022 走 WGC (关黄框 + 后台稳), 其它走 GDI. 新装默认.',
      hint_gdi: 'GDI (PrintWindow) 兼容性最好.',
      hint_wgc: 'WGC (Windows Graphics Capture) 后台抓帧稳, 但 Win10 上有黄框关不掉.',
      hint_mock: 'Mock 从 bin/mock-frames/ 读 PNG 序列回放, 调试用, 无需开游戏.',
      restart_hint: '改完需重启 exe 生效.',
      method: {
        auto: 'Auto (按 OS 选)',
        gdi: 'GDI',
        wgc: 'WGC',
        mock: 'Mock (离线回放)',
      },
      dump_debug_label: '落盘 detect 标注图',
      dump_debug_hint:
        'bot 识别关键路径异步把带框的 PNG 写到 debug/captures/, 调参/排查识别问题用. 立即生效.',
      method_changed_title: '截屏方式已切到 {method}',
      method_changed_desc: '重启程序生效',
    },
    log: {
      section_title: '日志',
      hint: '折叠/展开、写入文件、时间戳、折行、自动滚动等设置在底部日志面板 header 的设置图标里调整.',
    },
    input: {
      title: '输入校准',
      intro:
        '鼠标硬件 DPI 影响"相对位移"类录制(视角转动)的跨电脑回放. 录制子图时会把本机 360° counts 写入 RecordingContext 作为源; 回放时按 target/source 比例缩放 MouseMoveRel.',
      intro_box: {
        what_label: '这里改的是什么',
        what_desc: '本机值的"默认源". 改它影响:',
        item_default_source: '下次新建的 MouseCalibration 节点用这个值当默认',
        item_sync_action:
          '"同步本机值到所有容器"按钮 + 节点 Inspector "FOREIGN" 警告里的"同步"按钮, 把这个值写到所有容器',
        footnote_prefix: '这里改它',
        footnote_negation: '不会',
        footnote_rest:
          '自动改已有容器里的 MouseCalibration 节点 — 它们各自持值(容器自包含). 要批量更新点上面"同步本机值到所有容器", 或进入容器手动改.',
      },
      record: {
        title: '录制配置',
        hint: '改动需重启 YHBox 生效 (启动期注入).',
        stop_hotkey_label: '停录热键',
        stop_hotkey_hint: '游戏前台按下停止录制 (LL hook 拦截, 不透传游戏). 默认 F12.',
        mouse_mode_label: '鼠标语义',
        mouse_mode_hint:
          'relative (FPS): 录 RawDelta 给相机转向. absolute (UI/Slate): 录 screen px MouseMove 给 click/hover.',
        mouse_mode: {
          relative: '相对 (FPS 相机)',
          absolute: '绝对 (UI 点击)',
        },
      },
      counts: {
        title: '本机 360° HID counts',
        hint: '原地转身 360° 鼠标硬件上报的累积 |dx|',
        save_manual: '保存手填值',
        recalibrate: '重新校准',
        calibrate: '开始校准',
        open_hud: '打开鼠标 HUD',
        sync_all: '同步本机值到所有容器',
        share_hint: '也可以从其他电脑分享脚本附带的 counts, 直接手填',
      },
      howto: {
        title: '怎么用',
        step_open: '点「开始校准」打开对话框',
        step_focus: '切到游戏, 对准固定参照物, 准备好',
        step_start: '按 F8 开始 3 秒倒计时(不用回到本程序!)',
        step_spin: '倒计时结束后开始累计 → 原地匀速转一整圈 360°',
        step_stop: '转完再按一次 F8 停止',
        step_save: '切回程序点「保存」即可',
      },
      toast: {
        counts_not_set: '本机 counts360 未设置',
        synced_title: '已同步 {n} 个容器',
        synced_skipped: '跳过 {n} 个(无 MouseCalibration 节点)',
      },
      confirm: {
        sync_title: '同步到所有容器？',
        sync_desc:
          '当前本机 counts360 = {cur}.\n同步会覆盖所有本地容器主图 MouseCalibration 节点的值.',
        sync_confirm: '同步',
        sync_cancel: '不同步',
        calibrator_done_desc:
          '新值: {counts}\n是否一键同步到所有本地容器?\n(推荐: 替换所有容器主图 MouseCalibration 节点的 counts360)',
      },
    },
  },
  toast: {
    autostart_on: '已开启开机自启',
    autostart_off: '已关闭开机自启',
    lang_en_warn_title: 'Language switched to English',
    lang_en_warn_desc:
      '尚未支持英文模板: fish / cook / battle 的视觉模板未采集 EN 版本, 这几个功能会显示为不可用. UI 字符串已切换.',
  },
  status: {
    idle: '空闲',
    running: '▶ 跑中: {name}',
    container_fallback: '容器',
    stop_button: '停止',
    stop_tooltip: '停止当前运行 + 清队列 (Ctrl+Shift+F9)',
  },
  log: {
    header_title: '日志',
    count: '{n} 条',
    has_errors: '含错误',
    write_file_tooltip_on: '写入 {dir}/yhfish-*.log',
    write_file_tooltip_off: '未写入文件',
    empty: '无日志.',
    popover: {
      show_time: '显示时间',
      show_tag: '显示标签',
      wrap_text: '自动折行',
      auto_scroll: '自动滚动',
      write_file: '写入文件',
    },
  },
  common: {
    game_not_detected: '未检测到异环窗口',
  },
  hotkeys: {
    search_placeholder: '搜索热键名或绑定...',
    group: {
      system: '系统',
      action: '动作',
      container: '容器',
      schedule: '计划',
      editor: '编辑器',
    },
    status: {
      register_failed: '注册失败',
      unbound: '未绑定',
    },
    empty: '没有匹配的热键',
    toast: {
      bound: '已绑定 {hk}',
      cleared: '已清除热键',
    },
    label: {
      system: {
        execution_stop: '强停所有运行',
        calibrate_toggle: 'DPI 校准 启动/停止',
      },
      container: '容器 {name}',
      schedule: '计划 {name}',
      editor: {
        commandPalette: '命令面板',
        nodeSearch: '画布搜索',
        save: '保存',
        openSettings: '打开设置',
        undo: '撤销',
        redo: '重做',
        toggleExplorer: '节点浏览器 显示/隐藏',
      },
    },
    readonly: {
      editorBuiltin: '编辑器内置, 暂不可改',
    },
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
