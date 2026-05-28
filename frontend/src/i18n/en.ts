export default {
  sidebar: {
    automation: 'Automation',
    tools: 'Tools',
    collapse: 'Collapse',
    fish: 'Fishing',
    cook: 'Cooking',
    piano: 'Piano',
    battle: 'Battle',
    rhythm: 'Rhythm',
    actions: 'Actions',
    tasks: 'Tasks',
    schedules: 'Schedules',
    settings: 'Settings',
    help: 'Help',
    about: 'About',
  },
  controls: {
    start: 'Start',
    pause: 'Pause',
    resume: 'Resume',
    stop: 'Stop',
    state: {
      idle: 'Idle',
      running: 'Running',
      paused: 'Paused',
    },
  },
  settings: {
    language: 'Language',
    language_zh: '中文',
    language_en: 'English',
    language_restart_hint:
      'Some content (templates/configs) requires app restart after language change.',
    language_changed_title: 'Language switched',
    language_changed_desc: 'UI updated immediately; templates/configs require app restart.',
    startup: {
      section_title: '[EN-pending] Startup & Close',
      autostart_label: '[EN-pending] Auto-start on login',
      autostart_hint:
        '[EN-pending] Start YHBox after Windows login. Writes registry HKCU\\...\\Run\\YHBox, no admin needed.',
      tray_label: '[EN-pending] Close minimizes to tray',
      tray_hint:
        '[EN-pending] Clicking close (×) hides to system tray instead of exiting. Right-click tray icon to force quit.',
    },
    capture: {
      section_title: '[EN-pending] Capture method',
      hint_auto:
        '[EN-pending] Auto: WGC on Win11/Server 2022 (no yellow border, stable in bg), GDI elsewhere. Default for new installs.',
      hint_gdi: '[EN-pending] GDI (PrintWindow) — broadest compatibility.',
      hint_wgc:
        '[EN-pending] WGC (Windows Graphics Capture) — stable bg capture but yellow border on Win10.',
      hint_mock:
        '[EN-pending] Mock: replay PNG sequence from bin/mock-frames/. Debug only, no game needed.',
      restart_hint: '[EN-pending] Restart exe to apply.',
      method: {
        auto: '[EN-pending] Auto (OS-based)',
        gdi: 'GDI',
        wgc: 'WGC',
        mock: '[EN-pending] Mock (offline replay)',
      },
      dump_debug_label: '[EN-pending] Dump detect-annotated frames',
      dump_debug_hint:
        '[EN-pending] Bot detection async writes boxed PNGs to debug/captures/. For tuning / debugging detection issues. Applies immediately.',
      method_changed_title: '[EN-pending] Capture switched to {method}',
      method_changed_desc: '[EN-pending] Restart program to apply',
    },
    log: {
      section_title: '[EN-pending] Log',
      hint: '[EN-pending] Folding, file write, timestamps, line-wrap, autoscroll — settings live in the bottom log panel header gear.',
    },
    input: {
      title: '[EN-pending] Input calibration',
      intro:
        '[EN-pending] Mouse hardware DPI affects cross-machine replay of relative-motion recordings (camera turn). When recording a subgraph, the local 360° counts are written into RecordingContext as the source; on replay, MouseMoveRel scales by target/source ratio.',
      intro_box: {
        what_label: '[EN-pending] What this changes',
        what_desc: '[EN-pending] The "default source" for local values. Effects:',
        item_default_source:
          '[EN-pending] Next-created MouseCalibration node uses this as default',
        item_sync_action:
          '[EN-pending] "Sync local value to all containers" button + node Inspector "FOREIGN" warning sync button writes this value into all containers',
        footnote_prefix: '[EN-pending] Changing this here',
        footnote_negation: '[EN-pending] does NOT',
        footnote_rest:
          '[EN-pending] automatically update MouseCalibration nodes inside existing containers — they hold their own value (containers are self-contained). To bulk-update, click "Sync local value to all containers" above, or edit each container manually.',
      },
      record: {
        title: '[EN-pending] Recording config',
        hint: '[EN-pending] Changes require YHBox restart to apply (injected at startup).',
        stop_hotkey_label: '[EN-pending] Stop recording hotkey',
        stop_hotkey_hint:
          '[EN-pending] Pressed in-game to stop recording (LL hook intercepts, not forwarded to game). Default F12.',
        mouse_mode_label: '[EN-pending] Mouse semantics',
        mouse_mode_hint:
          '[EN-pending] relative (FPS): records RawDelta for camera turn. absolute (UI/Slate): records screen px MouseMove for click/hover.',
        mouse_mode: {
          relative: '[EN-pending] Relative (FPS camera)',
          absolute: '[EN-pending] Absolute (UI click)',
        },
      },
      counts: {
        title: '[EN-pending] Local 360° HID counts',
        hint: '[EN-pending] Cumulative |dx| reported by mouse hardware during a 360° in-place turn',
        save_manual: '[EN-pending] Save manual value',
        recalibrate: '[EN-pending] Recalibrate',
        calibrate: '[EN-pending] Start calibration',
        open_hud: '[EN-pending] Open mouse HUD',
        sync_all: '[EN-pending] Sync local value to all containers',
        share_hint:
          '[EN-pending] You can also hand-enter counts shared from another machine\'s script',
      },
      howto: {
        title: '[EN-pending] How to use',
        step_open: '[EN-pending] Click "Start calibration" to open the dialog',
        step_focus: '[EN-pending] Switch to game, aim at a fixed reference, get ready',
        step_start:
          '[EN-pending] Press F8 to start a 3-second countdown (no need to come back to this app!)',
        step_spin:
          '[EN-pending] After countdown, accumulation starts → turn 360° in place at steady speed',
        step_stop: '[EN-pending] Press F8 again to stop',
        step_save: '[EN-pending] Switch back to the app and click Save',
      },
      toast: {
        counts_not_set: '[EN-pending] Local counts360 not set',
        synced_title: '[EN-pending] Synced {n} containers',
        synced_skipped: '[EN-pending] Skipped {n} (no MouseCalibration node)',
      },
      confirm: {
        sync_title: '[EN-pending] Sync to all containers?',
        sync_desc:
          '[EN-pending] Current local counts360 = {cur}.\nSyncing overwrites the value in every local container\'s main-graph MouseCalibration node.',
        sync_confirm: '[EN-pending] Sync',
        sync_cancel: '[EN-pending] Do not sync',
        calibrator_done_desc:
          '[EN-pending] New value: {counts}\nOne-click sync to all local containers?\n(Recommended: replaces counts360 in every container\'s main-graph MouseCalibration node)',
      },
    },
  },
  toast: {
    autostart_on: '[EN-pending] Auto-start enabled',
    autostart_off: '[EN-pending] Auto-start disabled',
    lang_en_warn_title: 'Language switched to English',
    lang_en_warn_desc:
      '[EN-pending] EN templates not yet supported: fish / cook / battle visual templates not captured in EN, those features will show unavailable. UI strings switched.',
  },
  status: {
    idle: '[EN-pending] Idle',
    running: '[EN-pending] ▶ Running: {name}',
    container_fallback: '[EN-pending] Container',
    stop_button: '[EN-pending] Stop',
    stop_tooltip: '[EN-pending] Stop current run + clear queue (Ctrl+Shift+F9)',
  },
  log: {
    header_title: '[EN-pending] Log',
    count: '[EN-pending] {n} lines',
    has_errors: '[EN-pending] has errors',
    write_file_tooltip_on: '[EN-pending] Writing to {dir}/yhfish-*.log',
    write_file_tooltip_off: '[EN-pending] Not writing to file',
    empty: '[EN-pending] No logs.',
    popover: {
      show_time: '[EN-pending] Show time',
      show_tag: '[EN-pending] Show tag',
      wrap_text: '[EN-pending] Wrap text',
      auto_scroll: '[EN-pending] Auto-scroll',
      write_file: '[EN-pending] Write to file',
    },
  },
  common: {
    game_not_detected: 'Game window not detected',
  },
  hotkeys: {
    search_placeholder: '[EN-pending] Search hotkey name or binding...',
    group: {
      system: '[EN-pending] System',
      action: '[EN-pending] Action',
      container: '[EN-pending] Container',
      schedule: '[EN-pending] Schedule',
    },
    status: {
      register_failed: '[EN-pending] Registration failed',
      unbound: '[EN-pending] Unbound',
    },
    empty: '[EN-pending] No matching hotkey',
    toast: {
      bound: '[EN-pending] Bound to {hk}',
      cleared: '[EN-pending] Hotkey cleared',
    },
  },
  // backend ValidationError.Code → user-facing message.
  // Params: {param} placeholders use vue-i18n named-interpolation. Missing keys fall back to backend Message field.
  error: {
    NO_START: 'Main graph has no Start node',
    MULTIPLE_STARTS: 'Main graph has {count} Start nodes (expected exactly 1)',
    DANGLING_EDGE: 'Edge {from} → {to} references missing node ({missing})',
    INVALID_PIN: 'Node {nodeID} ({kind}) has no {side} pin {pin}',
    DUPLICATE_OUTPUT_PIN: 'OutputPin Name {name} duplicated',
    MISSING_TEMPLATE: 'Node {nodeID} references template {template} not found in container templates/',
    MISSING_SUBGRAPH: 'Subgraph node references unknown subgraph {subgraphId}',
    MISSING_MOUSE_CALIBRATION: 'Uses MouseMoveRel but container has no MouseCalibration node',
    DUPLICATE_MOUSE_CALIBRATION: 'Main graph has {count} MouseCalibration nodes (expected at most 1)',
    MOUSE_CALIBRATION_NOT_SET: 'MouseCalibration.counts360 is not set',
    MOUSE_CALIBRATION_FOREIGN: 'MouseCalibration node value {nodeValue} does not match local ({settingsValue}), looks downloaded from another machine',
    MOUSE_CALIBRATION_IN_SUBGRAPH: 'MouseCalibration node must be in main graph',
    EMPTY_SUBGRAPH_OUTPUT: 'Subgraph has no SubgraphOutput node',
    CYCLIC_SUBGRAPH_DEPENDENCY: 'Subgraph calls form a cycle',
    PLAYCLIP_NO_CLIP_ID: 'PlayClip node has no clipID',
    MISSING_WINDOW_TARGET: 'Main graph missing WindowTarget node',
    DUPLICATE_WINDOW_TARGET: 'Main graph has multiple WindowTargets (expected exactly 1)',
    WINDOW_TARGET_IN_SUBGRAPH: 'WindowTarget node must be in main graph',
    INVALID_WINDOW_TARGET_REGEX: 'WindowTarget regex invalid: {error}',
    INVALID_WINDOW_TARGET_EMPTY_MATCH: 'WindowTarget match cannot be empty',
    INVALID_ROI: 'ROI config invalid (w/h must be >= 1, got {w}x{h})',
    INVALID_DUALBAR_ROIS: 'DualColorBarTrack rois config invalid',
    DUPLICATE_DUALBAR_ROI: 'DualColorBarTrack rois contains duplicate resolution ({w}x{h})',
    INVALID_HSV_RANGE: 'HSV range is invalid',
    INVALID_SCAN_AXIS: 'scanAxis must be x or y, got {got}',
    INVALID_CLUSTER_RANGE: 'cluster range invalid (min={min} > max={max})',
    INVALID_VK: 'vk (virtual key code) is invalid',
    INVALID_MOUSE_BUTTON: 'button must be left/right/middle, got {button}',
    UNSAFE_SCREENSHOT_PATH: 'Screenshot pathTemplate is unsafe',
    POLL_TOO_FAST: 'Poll interval too small (<{minMs}ms), will spike CPU',
    STOPWATCH_EMPTY_KEY: 'Stopwatch key cannot be empty',
    STOPWATCH_KEY_MISMATCH: '{kind} uses key {key} but no matching StopwatchStart in same graph',
    THROW_IN_MAIN_GRAPH: 'Throw node cannot be in main graph (must be inside a subgraph)',
    INVALID_SWITCH_CASES: 'Switch cases invalid (empty / duplicate / contains . / named default)',
    INVALID_CRON_EXPR: 'Cron expression invalid: {expr} ({parseErr})',
    // v4 new
    PIN_TYPE_MISMATCH: 'data pin type incompatible: {from} → {to} (edge: {edge})',
    PIN_TYPE_COERCION_WARNING: 'data pin implicit coercion: {from} → {to} (suggest explicit To* node)',
    GETVAR_UNKNOWN_VAR: 'GetVar/SetVar/IncVar references undeclared variable {name}',
    GETVAR_TYPE_MISMATCH: 'GetVar output type does not match downstream expectation',
    LITERAL_TYPE_MISMATCH: 'inline literal type does not match pin type (pin {pin} expected {type}, got {got})',
    DATA_PIN_DANGLING: 'data-in pin {pin} has no edge and no literal',
    DATA_GRAPH_CYCLE: 'data flow forms a cycle: {cycle}',
    EXPR_PARSE_ERROR: 'Expr parse failed: {error}',
    EXPR_UNKNOWN_INPUT: 'Expr references undeclared input {name}',
    EXPR_TYPE_MISMATCH: 'Expr outType differs from inferred (expected {expected}, got {actual})',
    EXPR_DUPLICATE_INPUT: 'Expr inputs has duplicate name: {name}',
    GETSYS_UNKNOWN_PATH: 'GetSys path {path} not in sys schema',
    GETPARAM_UNKNOWN_PARAM: 'GetParam references undeclared input param {name}',
    COLLAPSED_PIN_BROKEN: 'CollapsedNode external pin has no matching marker in backing Subgraph',
    COLLAPSED_REFERENCED_BY_SUBGRAPH_CALL: 'Subgraph node references an isAnonymous Subgraph (it belongs to a CollapsedNode and cannot be reused across graphs)',
    // B11/B3+ added codes
    INVALID_VAR_REF: 'Node references undeclared container variable {varName} (scope={scope})',
    // disabled nodes
    WARN_DISABLED_BRANCH_NODE: 'Branch/async node {nodeID} (kind={kind}) disabled passthrough exit pin — non-deterministic, prefer delete over disable',
    INVALID_DISABLED_TERMINAL: 'Container-level node {nodeID} (kind={kind}) cannot be disabled (Start/WindowTarget)',
    // sentinel scope
    BREAK_OUTSIDE_LOOP: 'Break node must be downstream of a Loop body (exec-reachable in same graph)',
    CONTINUE_OUTSIDE_LOOP: 'Continue node must be downstream of a Loop body (exec-reachable in same graph)',
    THROW_OUTSIDE_TRY: 'Throw node must be inside a Subgraph referenced by Try.SubgraphID',
    // template/clip key validation
    INVALID_TEMPLATE_KEY: 'Template key {key} invalid: {error}',
    TEMPLATE_NOT_FOUND: 'Template {key} not found in container',
    CLIP_NOT_FOUND: 'Clip {id} not found in container',
    // service.go fallback
    CONTAINER_NOT_FOUND: 'Container {id} not found',
  },
  // ValidationErrorPanel modal shell strings — separate from error.* codes.
  validation: {
    title_failed: 'Validation failed — {errorCount} errors',
    title_failed_with_warns: 'Validation failed — {errorCount} errors / {warnCount} warnings',
    title_passed: 'Validation passed',
    desc_no_issues: 'No validation issues. Container can be run.',
    node_label: 'Node:',
    close: 'Close',
    run_button: 'Pass — continue running',
    fix_missing_window_target: 'Auto-add WindowTarget node',
  },
}
