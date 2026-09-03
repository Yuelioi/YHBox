export default {
  appClosing: {
    title: 'Closing Yotta',
    checking: 'Checking unfinished work…',
    stopping: 'Stopping the current operation…',
    restoring: 'Restoring the unsaved workflow…',
    closing: 'Releasing resources and closing windows…',
  },
  sidebar: {
    workflows: 'Workflows',
    workflow_edit: 'Edit workflow',
    assets: 'Library',
    schedules: 'Schedules',
    settings: 'Settings',
    about: 'About',
    open_launcher: 'Open floating launcher',
    primary_navigation: 'Primary navigation',
  },
  controls: {
    start: 'Start',
    pause: 'Pause',
    resume: 'Resume',
    stop: 'Stop',
    stopping: 'Stopping...',
    state: {
      idle: 'Idle',
      running: 'Running',
      paused: 'Paused',
    },
  },
  settings: {
    general: {
      appearance_title: 'Interface & language',
      capture_diagnostics_title: 'Capture & diagnostics',
    },
    editor_display: {
      section_title: 'Editor display',
      detail_label: 'Node detail level',
      detail_hint:
        'Only changes technical metadata visibility; variables, debugging, and output bindings stay available.',
    },
    language: 'Language',
    language_zh: '中文',
    language_en: 'English',
    language_restart_hint:
      'The interface switches now; templates and configuration switch on restart.',
    language_changed_title: 'Language switched',
    language_changed_desc: 'UI updated immediately; templates/configs require app restart.',
    startup: {
      section_title: 'Startup & Close',
      autostart_label: 'Auto-start on login',
      tray_label: 'Close minimizes to tray',
    },
    capture: {
      section_title: 'Capture method',
      hint_auto:
        'Auto: WGC on Win11/Server 2022 (no yellow border, stable in bg), GDI elsewhere. Default for new installs.',
      hint_gdi: 'GDI (PrintWindow) — broadest compatibility.',
      hint_wgc: 'WGC (Windows Graphics Capture) — stable bg capture but yellow border on Win10.',
      hint_mock: 'Mock: replay PNG sequence from bin/mock-frames/. Debug only, no game needed.',
      restart_hint: 'Restart exe to apply.',
      method: {
        auto: 'Auto (OS-based)',
        gdi: 'GDI',
        wgc: 'WGC',
        mock: 'Mock (offline replay)',
      },
      method_hint: {
        auto: 'Recommended. Chooses a stable capture method for the current Windows version.',
        gdi: 'Broadest compatibility, but some 3D apps may return a frozen frame in the background.',
        wgc: 'More reliable background capture for Windows 11 and modern graphics apps.',
        mock: 'Development only. Replays a local sequence of PNG frames.',
      },
      dump_debug_label: 'Dump detect-annotated frames',
      dump_debug_hint: 'Write annotated frames to debug/captures/ for detection troubleshooting.',
      method_changed_title: 'Capture switched to {method}',
      method_changed_desc: 'Restart program to apply',
    },
    log: {
      section_title: 'Log',
      hint: 'Folding, file write, timestamps, line-wrap, autoscroll — settings live in the bottom log panel header gear.',
      enabled_label: 'Enable runtime logging',
      level_label: 'Minimum level',
      level_hint: 'Record this level and more severe messages.',
      live_label: 'Stream to the log panel',
    },
    input: {
      title: 'Input calibration',
      intro:
        'Mouse hardware DPI affects cross-machine replay of relative-motion recordings (camera turns). Recording stores the local 360° count in InputClip metadata; playback scales by the target-to-source ratio.',
      record: {
        title: 'Recording config',
        hint: 'Applies to the next recording.',
        mouse_mode_label: 'Mouse semantics',
        mouse_mode_hint:
          'relative (FPS): records RawDelta for camera turn. absolute (UI/Slate): records screen px MouseMove for click/hover.',
        mouse_mode: {
          relative: 'Relative (FPS camera)',
          absolute: 'Absolute (UI click)',
        },
        mouse_mode_detail: {
          relative: 'Records raw mouse deltas for camera movement. Restart Yotta after changing.',
          absolute:
            'Records screen coordinates for clicks, hover and drag. Restart Yotta after changing.',
        },
      },
      counts: {
        title: 'Mouse calibration profiles',
        commercial_hint: 'Save a 360° raw-movement baseline per game or sensitivity.',
        hint: `Each profile = one game's cumulative {'|'}dx{'|'} for a 360° turn; if in-game sensitivity differs per game on the same machine, make one profile each and pick a default`,
        col_active: 'Default',
        col_label: 'Name',
        col_counts: 'counts360',
        set_active: 'Set “{name}” as the default calibration profile',
        label_placeholder: 'e.g. Genshin / Valorant',
        empty:
          'No calibration profiles yet. Click "Add profile" below, then "Calibrate" to measure the value.',
        add_profile: 'Add profile',
        new_profile_label: 'New profile',
        delete_profile: 'Delete this profile',
        recalibrate: 'Calibrate this profile',
        start_calibration: 'Start calibration',
        active_badge: 'Default',
        calibrated: 'Calibrated · {n}',
        uncalibrated: 'Needs calibration',
        advanced_value: 'Advanced value',
        advanced_value_hint: 'Raw counts for a 360° turn',
        make_default: 'Make default',
        empty_hint: 'Once calibrated, recordings and cross-device replay use the selected default.',
        share_hint: "You can also hand-enter counts shared from another machine's script",
      },
      howto: {
        title: 'How to use',
        compact:
          'In the target game, press {hk}, turn exactly 360° at a steady speed, then press it again. The result is saved to this profile.',
        step_open: 'Click "Start calibration" to open the dialog',
        step_focus: 'Switch to game, aim at a fixed reference, get ready',
        step_start: 'Press {hk} to start a 3-second countdown (no need to come back to this app!)',
        step_spin: 'After countdown, accumulation starts → turn 360° in place at steady speed',
        step_stop: 'Press {hk} again to stop',
        step_save: 'Switch back to the app and click Save',
      },
      confirm: {
        delete_profile_title: 'Delete “{name}”?',
        delete_profile_desc:
          'This calibration profile will be removed from local settings. Recorded assets are not deleted.',
      },
      validation: {
        label_required: 'Enter a profile name.',
        label_duplicate: 'Profile names must be unique.',
      },
    },
  },
  toast: {
    lang_en_warn_title: 'Language switched to English',
    lang_en_warn_desc:
      "Some scenarios' visual templates haven't been captured in English yet; related features may show as unavailable. UI strings switched.",
    save_failed: 'Save failed',
    operation_failed: 'Operation failed',
    and_n_more: ' (and {n} more)',
  },
  editor: {
    window: {
      controls: 'Window controls',
      minimize: 'Minimize',
      maximize: 'Maximize',
      restore: 'Restore',
      close: 'Close',
    },
  },
  type: {
    core: {
      string: { title: 'String', description: 'Durable Unicode text.' },
      number: { title: 'Number', description: 'A finite binary64 number.' },
      integer: {
        title: 'Integer',
        description: 'An exact integer within the interoperable JSON safe range.',
      },
      boolean: { title: 'Boolean', description: 'A strict true or false value.' },
      json: {
        title: 'JSON value',
        description: 'A canonical value from the interoperable JSON profile.',
      },
      binary: {
        title: 'Binary',
        description: 'Binary content represented by a durable blob or a leased runtime stream.',
      },
    },
    media: {
      image: {
        title: 'Image',
        description: 'Encoded image content represented only by a durable BlobRef.',
      },
    },
    geometry: {
      point_unit: { title: 'Coordinate unit', description: 'A ratio or pixel coordinate unit.' },
      point: {
        title: 'Point',
        description: 'A typed two-dimensional point with an explicit unit.',
      },
      region: { title: 'Region', description: 'A typed rectangular region with an explicit unit.' },
    },
    random: {
      distribution: {
        title: 'Random distribution',
        description: 'The probability distribution used by a recorded random observation.',
      },
    },
    time: {
      durationMilliseconds: {
        title: 'Duration (milliseconds)',
        description: 'A non-negative delay up to 24 hours, measured in milliseconds.',
      },
    },
    filesystem: {
      metadata: {
        title: 'File metadata',
        description: 'Canonical metadata for one file inside the Yotta-managed workflow workspace.',
      },
    },
    observability: {
      message: {
        title: 'Log message',
        description: 'Bounded text explicitly emitted to the attributed workflow log.',
      },
    },
    automation: {
      macro: {
        title: 'Keyboard macro',
        description:
          'A versioned, editable atomic keyboard and pointer macro carried as a durable BlobRef.',
      },
      inputClip: {
        title: 'Precise input trajectory',
        description:
          'A content-addressed recording of pointer motion, dragging, relative movement, keys, and original timing.',
      },
      pointer_button: {
        title: 'Pointer button',
        description: 'The left, right, or middle pointer button.',
      },
      pointer_motion: {
        title: 'Pointer motion',
        description:
          'Arrive instantly, follow a constant-speed line, or use a smooth Bézier curve.',
      },
      key_code: {
        title: 'Single key code',
        description: 'One canonical keyboard key; use the Key chord state type for combinations.',
      },
      held_input: {
        title: 'Held input lease',
        description:
          'Valid only for the current Run; release, cancellation, and failure all clean up held state.',
      },
    },
    vision: {
      templateMatch: {
        title: 'Template match',
        description: 'One scored template location in pixel coordinates.',
      },
      qrCode: { title: 'QR code', description: 'Decoded text and locator points for one QR code.' },
      colorRange: {
        title: 'Color range',
        description: 'An explicit inclusive RGB or HSV channel range.',
      },
      colorBlob: {
        title: 'Color blob',
        description: 'One four-connected color component with area and geometry.',
      },
    },
  },
  log: {
    header_title: 'Log',
    count: '{n} lines',
    has_errors: 'has errors',
    dropped: '{n} dropped',
    enable: 'Enable logging',
    disable: 'Disable logging',
    disabled: 'Logging is disabled; diagnostics are no longer produced or transported',
    live_paused: 'Live logs are paused; file logging continues when enabled',
    write_file_tooltip_on: 'Writing to {dir}/yotta-*.log',
    write_file_tooltip_off: 'Not writing to file',
    empty: 'No logs.',
    settings: 'Log display settings',
    clear: 'Clear logs',
    filter_label: 'Log source',
    filter_all: 'All logs',
    filter_sys: 'System logs',
    filter_wf: 'Workflow logs',
    popover: {
      enabled: 'Logging',
      enabled_hint: 'Stops production, transport, and file writes at the backend source',
      live_view: 'Stream to log panel',
      level: 'Minimum level',
      show_time: 'Show time',
      show_tag: 'Show tag',
      wrap_text: 'Wrap text',
      auto_scroll: 'Auto-scroll',
      write_file: 'Write to file',
    },
  },
  common: {
    more: 'More',
    search_options: 'Search options',
    cancel: 'Cancel',
    continue: 'Continue',
    confirm: 'Confirm',
    copied: 'Copied',
    save: 'Save',
    delete: 'Delete',
    edit: 'Edit',
    rename: 'Rename',
    close: 'Close',
    back: 'Back',
    copy: 'Copy',
    add: 'Add',
    create: 'Create',
    loading: 'Loading...',
    required: 'Required',
    optional: 'Optional',
    name: 'Name',
    description: 'Description',
    tags: 'Tags',
    category: 'Category',
    retest: 'Retest',
    retry: 'Retry',
    refresh: 'Refresh',
    coming_soon: 'Coming soon',
    untitled: '(Untitled)',
    yes: 'Yes',
    no: 'No',
    exec_in_pin: 'Run',
    fail_pin: 'Fail',
  },
  hotkeys: {
    search_placeholder: 'Search hotkey name or binding...',
    filter_label: 'Filter hotkey status',
    reset_menu: 'Reset & clean up',
    capture_aria: 'Set the shortcut for {name}',
    empty_hint: 'Try a different search term or status filter.',
    clear_filters: 'Clear filters',
    filter: {
      all: 'All statuses',
      failed: 'Registration failed',
      unbound: 'Unbound only',
    },
    summary: {
      total: '{n} total',
      failed: '{n} failed',
      unbound: '{n} unbound',
    },
    group: {
      system: 'System',
      recording: 'Recording',
      action: 'Action',
      schedule: 'Schedule',
      editor: 'Editor',
    },
    status: {
      register_failed: 'Registration failed',
    },
    empty: 'No matching hotkey',
    reset_system: 'Reset defaults',
    toast: {
      bound: 'Bound to {hk}',
      cleared: 'Hotkey cleared',
      reset_done: 'Built-in hotkeys reset to defaults',
    },
    confirm: {
      reset_title: 'Reset built-in hotkeys?',
      reset_desc:
        'Strong-stop / calibrate / recording start / recording stop / recording pause / recording cancel will return to factory defaults.',
      reset_ok: 'Reset',
    },
    label: {
      workflow: 'Workflow {name}',
      launcher_slot: 'Launcher slot {n}: {name}',
      system: {
        execution_stop: 'Stop all running',
        calibrate_toggle: 'DPI calibration toggle',
        window_capture: 'Window capture (press key to grab game window)',
        launcher_toggle: 'Show/hide launcher window',
      },
      recording: {
        start: 'Start recording',
        stop: 'Stop recording',
        pause: 'Pause / resume recording',
        cancel: 'Cancel and discard recording',
      },
      schedule: 'Schedule {name}',
      editor: {
        commandPalette: 'Command palette',
        nodeSearch: 'Canvas node search',
        save: 'Save',
        openSettings: 'Open settings',
        undo: 'Undo',
        redo: 'Redo',
        toggleExplorer: 'Toggle node explorer',
        togglePalette: 'Toggle left panel',
        toggleInspector: 'Toggle right panel',
      },
    },
    readonly: {
      editorBuiltin: 'Editor built-in, not changeable yet',
    },
  },
  error: {
    UNSUPPORTED_WORKFLOW_FORMAT: 'Unsupported Workflow format or version',
    INVALID_WORKFLOW_JSON: 'Workflow JSON is invalid',
    DUPLICATE_FIELD: 'A field is duplicated',
    UNKNOWN_FIELD: 'An unknown field is present',
    MISSING_REQUIRED_FIELD: 'A required field is missing',
    INVALID_FIELD: 'A field value is invalid',
    DUPLICATE_ID: 'An ID is duplicated',
    MISSING_ENTRY_GRAPH: 'The entry graph is missing',
    UNKNOWN_NODE_KIND: 'The node kind is invalid',
    UNSUPPORTED_NODE_CONTRACT: 'The node contract is unsupported',
    UNSUPPORTED_GRAPH_CONTRACT: 'The graph contract is unsupported',
    INVALID_GRAPH_ENTRY: 'The graph entry is invalid',
    MISSING_GRAPH_OUTPUT: 'The graph is missing a declared output',
    INVALID_GRAPH_BOUNDARY_EDGE: 'A graph-boundary edge is invalid',
    UNKNOWN_CALLEE_GRAPH: 'The called graph does not exist',
    INVALID_CALLEE_GRAPH_KIND: 'The called graph kind cannot be invoked',
    SUBGRAPH_CALL_CYCLE: 'Subgraph calls form a cycle',
    CALL_PIN_TYPE_MISMATCH: 'Graph-call port types do not match',
    INVALID_DYNAMIC_PORT_DECLARATION: 'A dynamic-port declaration is invalid',
    DYNAMIC_PORT_BUDGET_EXCEEDED: 'The dynamic-port budget was exceeded',
    INPUT_CONSTRAINT_VIOLATION: 'An input constraint is not satisfied',
    INPUT_CONSTRAINT_BUDGET_EXCEEDED: 'The input-constraint budget was exceeded',
    DIAGNOSTIC_BUDGET_EXCEEDED: 'The diagnostic budget was exceeded',
    MISSING_CAPABILITY_DECLARATION: 'A required capability declaration is missing',
    UNUSED_CAPABILITY_DECLARATION: 'A capability declaration is unused',
    INVALID_CATALOG: 'The node catalog is invalid',
    UNKNOWN_NODE_TYPE: 'The node type is not in the catalog',
    NODE_CONTRACT_MISMATCH:
      'This node definition does not match the current version. Open the workflow, locate the node, then upgrade or replace it with the current node.',
    UNKNOWN_PORT:
      'A workflow connection points to an input or output that no longer exists. Open the workflow, locate the affected node, remove the old connection, then reconnect using the inputs or outputs currently shown.',
    EDGE_CHANNEL_MISMATCH: 'Edge channel kinds do not match',
    TYPE_MISMATCH: 'Data types do not match',
    UNRESOLVED_TYPE: 'A data type could not be resolved',
    RESOURCE_LEASE_MISMATCH: 'Resource leases do not match',
    MISSING_INPUT_BINDING: 'A required input binding is missing',
    DUPLICATE_INPUT_BINDING: 'An input is bound more than once',
    DUPLICATE_SIGNAL_ROUTE: 'A signal route is duplicated',
    REGION_SIGNAL_SCOPE: 'A signal crosses its region scope',
    INVALID_BINDING: 'An input binding is invalid',
    BLOB_UNAVAILABLE: 'The bound resource content is unavailable',
    INVALID_CONFIG: 'Node configuration is invalid',
    INVALID_STATE_VARIABLE: 'A state-variable declaration is invalid',
    INVALID_STATE_ACCESS: 'State-variable access is invalid',
    INVALID_CAPABILITY_BINDING: 'A capability binding is invalid',
    INVALID_INSTRUCTION_PLACEMENT: 'The node instruction is not valid in this graph',
    NO_EXECUTION_ROOT: 'No execution root is available',
    UNREACHABLE_EXECUTION: 'An execution node is unreachable',
    DATA_CYCLE: 'Data dependencies form a cycle',
    UNSUPPORTED_GRAPH: 'The graph structure is unsupported',
    UNSUPPORTED_SOURCE_FEATURE: 'The Workflow uses an unsupported source feature',
    WAILS_NOT_READY: 'The desktop runtime is not ready',
    AUTOMATION_TARGET_SLOT_REQUIRED: 'An automation target must be selected',
    RECORDING_TARGET_UNAVAILABLE:
      'The recording target is unavailable. Check that the target window is running and still matches its selector.',
    RECORDING_MODE_REQUIRED: 'Choose simple or precise recording',
    RECORDING_CALIBRATION_REQUIRED:
      'Precise relative recording requires mouse calibration on the selected automation target',
    RECORDING_SESSION_BUSY: 'Finish or discard the current recording before starting another one',
    'recording.stop.failed':
      'Recording finalization failed. The recording stopped; record it again and use the operation ID to locate the cause in logs.',
    'recording.start.adapter_failed':
      'The recorder could not begin capture. Check the target window and record again.',
    'recording.start.failed':
      'Recording could not be prepared. Check the target window, recording hotkeys, and calibration, then retry.',
    'recording.start.target_reactivation_failed':
      'The same target window could not be reactivated when the countdown ended, so recording did not start. Confirm the target is still running and retry.',
    'recording.stop.unstructured_failure':
      'Recording finalization returned a legacy unstructured error. Restart Yotta and record again.',
    'recording.stop.result_invalid':
      'Recording finalization returned an invalid result. Restart Yotta and record again.',
    'recording.finalize.invalid_request': 'The recording {field} is invalid. Correct it and retry.',
    'recording.finalize.pending_unavailable':
      'The pending recording is no longer available. Keep the current input and record it again.',
    'recording.finalize.failed':
      'The recording could not be saved to {destination}. It is still pending, so you can retry saving.',
    'recording.finalize.result_invalid':
      'The recording was saved but returned an invalid result. Reload the library and retry only if it is absent.',
    'recording.events.unavailable': 'Raw events are available only for precise recordings.',
    'recording.events.invalid_page':
      'The recording event page request is invalid. Close the details and reopen them.',
    ASSET_QUERY_INVALID: 'The asset query is invalid; try again',
    'asset.template.capture_invalid':
      'The capture data is invalid. Select the capture region again.',
    'asset.template.load_failed':
      'The original visual template could not be read. Reopen the resource and retry.',
    'asset.template.not_found':
      'The original visual template no longer exists. Refresh the library.',
    'asset.template.save_failed':
      'The visual template could not be saved to the library. Retry, then use the operation ID to inspect logs.',
    'workflow.resource.image_create_failed':
      'The capture could not be added to this workflow. Retry, then use the operation ID to inspect logs.',
    UNKNOWN_ERROR: 'An unknown error occurred',
    OPERATION_ID: 'Operation ID: {id}',
    'system.unexpected': 'The operation did not complete. Try again.',
    'transport.unstructured_failure':
      'The {operation} call returned no valid error result. Try again or restart Yotta.',
    'hotkey.conflict':
      'This hotkey is already used by “{conflictLabel}”. Choose another combination.',
    'hotkey.reserved':
      'The {hotkey} combination is reserved by the system or editor. Choose another combination.',
    'hotkey.invalid': 'The {hotkey} combination is invalid. Record it again.',
    'hotkey.update_failed': 'The hotkey could not be saved. Keep the current setting and retry.',
    'hotkey.pause_failed':
      'Hotkey capture could not start. Close any other capture operation and retry.',
    'hotkey.resume_failed': 'Global hotkeys could not be restored. Reopen Hotkey settings.',
    'hotkey.registration_failed':
      'The {hotkey} hotkey could not be registered. Choose another combination or close the application using it.',
    'hotkey.rollback_failed':
      'The hotkey update failed and the previous binding could not be restored. Reopen Hotkey settings and configure it again.',
    'settings.invalid': 'The settings are invalid. Check the field you just changed.',
    'settings.update.invalid':
      'The settings update could not be parsed. Correct it and save again.',
    'settings.update_failed': 'The settings were not saved. Check disk access and retry.',
    'settings.update.committed_sync_failed':
      'The settings were saved, but the running features could not synchronize. Do not save again; restart Yotta to apply them.',
    'calibration.unavailable': 'Calibration is unavailable. Restart Yotta.',
    'calibration.start_failed':
      'Calibration could not start. Close applications that may be using input devices and retry.',
    'calibration.stop_failed':
      'Calibration could not stop. Close and reopen the calibration window.',
    'calibration.hotkey_watch_failed':
      'The calibration hotkey could not be monitored. Change it or close the application using it.',
    'schedule.not_found': 'Schedule {id} was not found. Refresh the schedule list.',
    'schedule.runner_unavailable': 'The schedule runner is not ready. Restart Yotta and retry.',
    'schedule.fire_failed': 'Schedule {id} could not start. Check the target workflow run state.',
    'schedule.update.invalid_patch': 'The schedule changes are invalid. Edit them and save again.',
    'schedule.save_failed': 'Schedule {id} was not saved. Check disk access and retry.',
    'schedule.delete_failed': 'Schedule {id} was not deleted. Check disk access and retry.',
    'schedule.committed_reload_failed':
      'The schedule change was saved, but the running scheduler could not reload it. Do not repeat the action; restart Yotta to synchronize it.',
    'input_clip.store_unavailable': 'The input recording library is unavailable. Restart Yotta.',
    'input_clip.invalid': 'The input recording is invalid. Record it again.',
    'input_clip.save_failed': 'Input recording {id} was not saved. Check disk access and retry.',
    'input_clip.load_failed':
      'Input recording {id} could not be read. Refresh the library and retry.',
    'input_clip.not_found': 'Input recording {id} was not found. Refresh the library.',
    'input_clip.corrupt': 'Input recording {id} is damaged. Record it again.',
    'input_clip.events.invalid_page':
      'The input event page request is invalid. Reopen the event details.',
    'input_clip.list_failed':
      'The input recording list could not be read. Check disk access and retry.',
    'input_clip.delete_failed':
      'Input recording {id} was not deleted. Check disk access and retry.',
    'input_clip.update_failed':
      'Input recording {id} was not updated. Check disk access and retry.',
    'macro.unavailable': 'The macro library is unavailable. Restart Yotta.',
    'macro.invalid': 'The macro is invalid. Check its name and actions, then retry.',
    'macro.load_failed': 'Macro {id} could not be read. Check disk access and retry.',
    'macro.identity_conflict':
      'Asset {id} already exists but is not a macro. Save with another name.',
    'macro.save_failed': 'Macro {id} was not saved. Check disk access and retry.',
    'macro.not_found': 'Macro {id} was not found. Refresh the library.',
    'macro.corrupt': 'Macro {id} is damaged and cannot be opened.',
    'macro.list_failed': 'The macro list could not be read. Check disk access and retry.',
    'macro.delete_failed': 'Macro {id} was not deleted. Check disk access and retry.',
    'ai.authoring.unavailable': 'AI proposals are unavailable. Restart Yotta and try again.',
    'ai.authoring.profile_not_found':
      'Model profile {slot} was not found. Choose the AI proposal model again.',
    'ai.authoring.tool_calling_required':
      'Model {slot} does not have Tool Calling enabled. Enable it in AI settings.',
    'ai.authoring.credential_unavailable':
      'The login or credential for model {slot} is unavailable. Check it in AI settings.',
    'ai.authoring.agent_unsupported':
      'Model {slot} does not support continuous tool conversations. Choose another model.',
    'ai.authoring.provider_failed':
      'The model failed during {stage} ({class}). Check the Codex login or model connection and retry.',
    'ai.authoring.failed':
      'This turn did not complete. Review the error above, then continue or retry.',
    'ai.authoring.budget_exhausted':
      'AI reached the tool-step limit before completing this proposal. Retry, be more specific, or raise the limit under Settings → AI Models → Proposal execution.',
    'ai.authoring.profile_invalid':
      'Model profile {slot} cannot be used for AI proposals. Save the profile again.',
    'ai.authoring.provider_unavailable':
      'Could not start model {slot} ({provider}). For Codex, ensure the Codex CLI is available on Yotta’s PATH.',
    'ai.authoring.run_unavailable':
      'The Run to diagnose could not be found. Select it again from this workflow’s timeline.',
    'ai.authoring.run_workflow_mismatch':
      'The selected Run belongs to another workflow. Start AI diagnosis again from this workflow’s timeline.',
    'ai.authoring.tool_input_invalid':
      'The model generated invalid arguments for {tool}, so no proposal was created. Try again.',
    'ai.authoring.conversation_not_found':
      'This AI conversation no longer exists. Create a new conversation.',
    'ai.authoring.conversation_capacity':
      'This conversation reached its history limit. Create a new conversation to continue.',
    'ai.credential.unavailable':
      'Credential storage for model {slot} is unavailable. Restart Yotta.',
    'ai.credential.save_failed': 'The credential for model {slot} was not saved. Retry.',
    'ai.credential.delete_failed': 'The credential for model {slot} was not deleted. Retry.',
    'ai.evaluation.unavailable': 'AI evaluation is unavailable. Restart Yotta.',
    'ai.evaluation.invalid':
      'Evaluation evidence for model {slot} is invalid. Run the evaluation again.',
    'ai.evaluation.apply_failed':
      'The evaluation result for model {slot} was not applied. Check the model configuration and retry.',
    'ai.evaluation.committed_sync_failed':
      'The evaluation result for model {slot} was saved, but runtime state did not synchronize. Do not apply it again; restart Yotta.',
    'workflow.revision.conflict':
      'Another operation updated this workflow. Reload the latest revision before editing.',
    'workflow.source.failed':
      'Workflow operation {operation} did not complete. Your current changes are preserved; retry.',
    'workflow.bundle.failed':
      'Workflow package operation {operation} did not complete. Check the file and disk access, then retry.',
    'workflow.run.failed':
      'Workflow run operation {operation} did not complete. Refresh the run state and retry.',
    'workflow.feature.unavailable': 'Workflow feature {feature} is unavailable. Restart Yotta.',
    'workflow.compile.failed':
      'Workflow validation did not complete. Fix any displayed diagnostics; if none appear, restart Yotta.',
    'workflow.draft.invalid':
      'The workflow draft contains errors. Fix the located node before saving.',
    'workflow.run_query.invalid': 'The run-state query is invalid. Refresh the workflow list.',
    'workflow.timeline.destination_required': 'Choose a destination for the run timeline export.',
    'workflow.timeline.export_failed':
      'The run timeline could not be exported. Check the destination and disk access, then retry.',
    'workflow.resource.image_version_missing':
      'This visual template has no version available to re-record. Reopen the resource and try again.',
    'workflow.resource.recapture_target_stale':
      'The template version to replace has changed. Select that version again before recording.',
    'workflow.resource.capture_result_invalid':
      'The capture did not return a valid template image. Capture it again.',
    'workflow.resource.capture_apply_failed':
      'The capture completed but could not be added to this workflow. Reopen the workflow and try again.',
    'automation.health.ready': 'The target identity and automation runtime are ready.',
    'automation.window_capture.no_foreground':
      'No foreground window was detected. Switch to the target window and capture again.',
    'automation.window_capture.metadata_failed':
      'The target window details could not be read. Confirm it is still running and retry.',
    'automation.window_capture.executable_failed':
      'The application behind the target window could not be identified. Reopen it and retry.',
    'automation.window_capture.unstructured_failure':
      'Window capture returned a legacy error. Restart Yotta and retry.',
    'tools.target_unavailable': 'The automation target service is unavailable. Restart Yotta.',
    'tools.target_resolve_failed':
      'Target {slot} could not be reached. Confirm that its application is running.',
    'tools.pixel_sample_failed':
      'A pixel could not be read from target {slot}. Confirm that its window is visible.',
    'tools.color_range.invalid': 'The color samples are invalid. Select the color region again.',
    'tools.mouse_position_failed':
      'The pointer position could not be read. Reopen the coordinate tool.',
    'tools.window_open_failed': 'The {window} window could not be opened. Retry or restart Yotta.',
    'tools.picker.invalid': 'Picker field {field} is invalid. Start the selection again.',
    'tools.picker.open_failed':
      'The picker could not open for target {slot}. Confirm that its application is running.',
    'tools.window_capture_unsupported': 'Hotkey window capture is not supported on this platform.',
    'tools.window_capture_start_failed':
      'Window capture could not start. Close another capture operation and retry.',
    'tools.window_capture_cancel_failed':
      'Window capture did not stop cleanly. Restart Yotta before capturing again.',
    'automation.health.unavailable': 'Automation health checks are currently unavailable.',
    'automation.health.invalid_profile':
      'Automation target {slot} has an invalid profile. Review and save its settings.',
    'automation.health.not_found':
      'Automation target {slot} was not found. Check the target-slot binding.',
    'asset.batch.duplicate': 'The asset batch contains a duplicate item.',
    'asset.not_found': 'Asset {guid} was not found.',
    'asset.load_failed': 'Asset {guid} could not be read. Check disk access and retry.',
    'asset.list_failed': 'The asset list could not be read. Check disk access and retry.',
    'asset.binding_failed':
      'The asset binding could not be resolved. Refresh the library and retry.',
    'asset.update_failed': 'Asset {guid} was not updated. Check disk access and retry.',
    'asset.delete_failed': 'Asset {guid} was not deleted. Check disk access and retry.',
    'asset.variant.last': 'Asset {guid} has only one variant. Delete the whole asset to remove it.',
    'asset.variant.remove_failed':
      'A variant of asset {guid} was not deleted. Check disk access and retry.',
    'asset.variant.not_found': 'Asset {guid} has no variant suitable for the current resolution.',
    'asset.variant.inconsistent':
      'Asset {guid} has inconsistent variant metadata. Import or capture it again.',
    'asset.preview.invalid': 'The asset preview is unsupported or damaged.',
    'asset.preview.failed': 'The asset preview could not be generated. Refresh and retry.',
    'asset.capture.unavailable': 'Screen capture is unavailable. Restart Yotta.',
    'asset.capture.failed':
      'Target {slot} could not be captured. Confirm that its window is running and visible.',
    'asset.target.failed':
      'Target {slot} state could not be read. Confirm that its application is running.',
    'asset.target.invalid_resolution':
      'Target {slot} returned an invalid resolution. Reopen its application.',
    'workflow.batch.duplicate': 'The Workflow batch contains a duplicate item.',
    'workflow.bundle.unavailable': 'Workflow export is currently unavailable.',
    'workflow.bundle.directory_required': 'Choose a Workflow export directory.',
    'workflow.bundle.destination_exists':
      'The export destination already exists. Choose another directory or filename.',
    'workflow.source.referenced':
      'The Workflow is still referenced by {references} local configurations and cannot be deleted.',
    'workflow.connection.invalid':
      'This connection cannot be created. Check port direction, channel, and data type.',
    'snippet.store.unavailable': 'Snippet storage is currently unavailable.',
    'snippet.load.invalid': 'Snippet file {file} could not be read.',
    'snippet.load.identity_mismatch':
      'Snippet file {file} has an identity that does not match its filename.',
    'snippet.invalid': 'The snippet is invalid. Check its name, shortcut, and node configuration.',
    'snippet.node_incompatible':
      'Node {nodeTypeId} in this snippet is incompatible with this Yotta version.',
    'snippet.shortcut_conflict':
      'Shortcut {shortcut} is already used by snippet “{name}”. Choose another combination.',
    'snippet.save_failed': 'Snippet {id} was not saved. Check disk access and retry.',
    'snippet.delete_failed': 'Snippet {id} was not deleted. Check disk access and retry.',
    'snippet.mark_used_failed':
      'Snippet {id} ran, but its usage record was not saved. Do not run it again.',
    'snippet.not_found': 'Snippet {id} was not found. Refresh the list.',
    'snippet.migration_save_failed':
      'Snippet {id} was upgraded, but the result was not saved. Check disk access and retry.',
    UNEXPECTED_CODE: 'The operation did not complete. Try again (code: {code}).',
    TRANSPORT_TIMEOUT: 'The request timed out; try again',
    TRANSPORT_UNAVAILABLE: 'The backend connection is unavailable; restart Yotta',
    math: {
      division_by_zero:
        'The divisor cannot be zero; check the B input of the Divide or Modulo node',
      domain_error:
        'The value is outside the real-number domain; check the Power or Square Root inputs',
      result_not_representable:
        'The result is not a storable finite number or exceeds the safe-integer range',
    },
    text: {
      invalid_regex: 'The regular expression is invalid; correct its syntax and try again',
    },
    conversion: {
      invalid_number:
        'The text is not a finite decimal number; remove whitespace and check its syntax and range',
      invalid_integer:
        'The value is not a safe integer; check its syntax and the +/-9007199254740991 range',
      invalid_boolean: 'The text is not a boolean; only lowercase true or false is accepted',
      blob_to_stream_failed:
        'The binary content could not be opened as a stream; check that the resource is still available',
      stream_to_blob_failed: 'The runtime stream could not be committed as durable binary content',
    },
    json: {
      invalid_document:
        'The JSON document is invalid or contains numbers outside the interoperable profile',
      invalid_path: 'The JSON path is invalid or exceeds the 64-step or 1024-byte budget',
    },
    ai: {
      generation_failed:
        'The AI model could not complete generation; check the selected model, endpoint, and request.',
    },
    collection: {
      index_out_of_range:
        'The list index is outside the available items; check List Length or the index value.',
    },
    control: {
      delay_failed: 'The delay could not complete; check cancellation and the requested duration.',
      switch_failed: 'The switch value or one of its cases is invalid for the resolved type.',
      thrown: 'The workflow was explicitly failed by a Fail workflow node.',
    },
    geometry: {
      unit_mismatch:
        'The points use different coordinate units; make both ratio coordinates or both pixel coordinates.',
    },
    inputclip: {
      invalid: 'The input recording clip is invalid or no longer available.',
    },
    macro: {
      invalid: 'The macro is invalid or no longer available.',
    },
    observability: {
      log_contract_violation: 'The workflow log payload violated its runtime contract.',
      log_write_failed: 'The workflow log entry could not be recorded.',
    },
    random: {
      empty_choice: 'Random Choice requires a non-empty typed list.',
      entropy_unavailable:
        'The host entropy source is unavailable; the random value was not produced.',
      invalid_probability: 'Probability must be a finite number from 0 through 1.',
      invalid_range: 'The random range is invalid or cannot be represented.',
    },
    state: {
      read_failed: 'The Run-local state value could not be read.',
      update_failed: 'The numeric state value could not be updated atomically.',
      write_failed: 'The Run-local state value could not be written.',
    },
    time: {
      observation_failed: 'The invocation time could not be recorded.',
      stopwatch_failed: 'The stopwatch start instant or elapsed value is invalid.',
    },
    vision: {
      analysis_failed: 'Image analysis could not complete.',
      color_range_invalid: 'The color range is invalid for the selected color space.',
      image_invalid: 'The image is invalid, unsupported, or outside the processing budget.',
      match_failed: 'Template matching could not complete.',
      region_invalid: 'The image region is invalid or lies outside the frame.',
      template_invalid:
        'The template image is invalid, unsupported, or outside the processing budget.',
    },
    admission: {
      target_unavailable:
        'A required target is unavailable. Check that it is configured and currently matches.',
      target_ambiguous: 'The target match is ambiguous',
      provider_incompatible: 'The capability provider is incompatible',
      unsupported_host: 'The current host does not support the required capability',
      credential_unavailable: 'A required credential is unavailable',
      credential_ambiguous: 'The credential match is ambiguous',
      consent_required: 'User consent is required before execution',
      policy_denied: 'Security policy denied this execution',
      policy_invalid: 'The security policy is invalid',
      persistence_failed: 'The grant record could not be saved',
      persistence_unconfirmed: 'Grant persistence could not be confirmed',
    },
    application: {
      invalid_request: 'The application-control request is invalid',
      launch_failed: 'The application failed to launch',
      terminate_failed: 'The application failed to terminate',
      unsupported_host: 'Application control is unsupported on this host',
      contract_violation: 'The application-control provider violated its contract',
    },
    automation: {
      invalid_request: 'The automation request is invalid',
      target_not_found: 'The automation target was not found',
      target_ambiguous: 'The automation target match is ambiguous',
      input_failed: 'The input operation failed',
      window_failed: 'The window operation failed',
      capture_failed: 'Capture failed',
      playback_failed: 'Input playback failed',
      playback_busy: 'Input playback is busy',
      unsupported_host: 'This automation operation is unsupported on the host',
      contract_violation: 'The automation provider violated its contract',
      dual_color_bar_not_found: 'The dual color bar was not found before the timeout.',
      observation_failed: 'The automation target could not produce the requested observation.',
    },
    network: {
      invalid_request: 'The network request is invalid',
      request_failed: 'The network request failed',
      response_too_large: 'The network response exceeds the size limit',
      invalid_response: 'The network response is invalid',
      contract_violation: 'The network provider violated its contract',
    },
    filesystem: {
      invalid_path: 'The file path is invalid',
      not_found: 'The file was not found',
      budget_exceeded: 'The filesystem budget was exceeded',
      is_directory: 'A file was expected, but the target is a directory',
      read_failed: 'The file could not be read',
      contract_violation: 'The filesystem provider violated its contract',
      decode_failed: 'The file could not be decoded with the selected text encoding.',
      invalid_json: 'The file does not contain one valid interoperable JSON document.',
      write_failed: 'The file could not be written to the workflow workspace.',
    },
    script: {
      source_invalid: 'The script source is invalid',
      guest_thrown: 'The script threw an error',
      deadline_exceeded: 'The script exceeded its deadline',
      stack_exceeded: 'The script exceeded its stack limit',
      contract_violation: 'The script result violated its contract',
      runner_protocol_violation: 'The script worker violated its protocol',
      runner_crashed: 'The script worker crashed',
      isolation_unavailable: 'Required script isolation is unavailable on this host',
    },
  },
}
