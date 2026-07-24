export default {
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
      appearance_hint: 'Control editor information density and the Yotta display language.',
      behavior_hint: 'Choose how Yotta starts after sign-in and behaves when its window closes.',
      capture_diagnostics_title: 'Capture & diagnostics',
      capture_diagnostics_hint:
        'Keep the recommended defaults unless you are diagnosing compatibility or detection issues.',
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
      'Some content (templates/configs) requires app restart after language change.',
    language_changed_title: 'Language switched',
    language_changed_desc: 'UI updated immediately; templates/configs require app restart.',
    startup: {
      section_title: 'Startup & Close',
      autostart_label: 'Auto-start on login',
      autostart_hint:
        'Start Yotta after Windows login through a highest-privilege scheduled task without another UAC prompt.',
      tray_label: 'Close minimizes to tray',
      tray_hint:
        'Clicking close (×) hides to system tray instead of exiting. Right-click tray icon to force quit.',
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
      dump_debug_hint:
        'Bot detection async writes boxed PNGs to debug/captures/. For tuning / debugging detection issues. Applies immediately.',
      method_changed_title: 'Capture switched to {method}',
      method_changed_desc: 'Restart program to apply',
    },
    log: {
      section_title: 'Log',
      hint: 'Folding, file write, timestamps, line-wrap, autoscroll — settings live in the bottom log panel header gear.',
      enabled_label: 'Enable runtime logging',
      enabled_hint:
        'Stops log production at the source when disabled, reducing overhead during long automation runs.',
      level_label: 'Minimum level',
      level_hint: 'Keep this level and more severe messages. INFO is recommended for daily use.',
      live_label: 'Stream to the log panel',
      live_hint:
        'When disabled, logs can still be written to file without streaming to the main window.',
    },
    input: {
      title: 'Input calibration',
      intro:
        'Mouse hardware DPI affects cross-machine replay of relative-motion recordings (camera turns). Recording stores the local 360° count in InputClip metadata; playback scales by the target-to-source ratio.',
      record: {
        title: 'Recording config',
        hint: 'Config applies on the next recording.',
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
        commercial_hint:
          'Keep a profile per game or sensitivity so relative movement replays consistently across devices.',
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
        title: 'Input clip',
        description: 'A content-addressed recording carried only as a durable nominal BlobRef.',
      },
      pointer_button: {
        title: 'Pointer button',
        description: 'The left, right, or middle pointer button.',
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
  node: {
    ai: {
      generate: {
        title: 'Generate AI text',
        description:
          'Generates text through an installed model resolved by explicit model, credential, and consent slots.',
      },
      extract: {
        title: 'Extract structured AI data',
        description:
          'Requires an installed model to return JSON that validates against the supplied JSON Schema.',
        config: {
          schema: {
            title: 'Output JSON Schema',
            description:
              'The strict JSON Schema document used to constrain and validate structured model output.',
          },
        },
      },
      agent: {
        title: 'Run bounded AI agent',
        description:
          'Runs a provider-native tool loop with an exact ToolSet and host-enforced token, cost, time, iteration, call, and parallelism budgets.',
        config: {
          maxInputTokens: {
            title: 'Total input token budget',
            description: 'Maximum provider-reported input tokens across all turns.',
          },
          maxTotalOutputTokens: {
            title: 'Total output token budget',
            description: 'Maximum provider-reported output tokens across all turns.',
          },
          maxCost: {
            title: 'Cost budget',
            description: 'Maximum profile-priced billing microunits across all turns.',
          },
          maxWallTime: {
            title: 'Wall-time budget (ms)',
            description: 'Maximum host wall time for the complete tool loop.',
          },
          maxIterations: {
            title: 'Iteration budget',
            description: 'Maximum provider turns, including the initial turn.',
          },
          maxToolCalls: {
            title: 'Tool-call budget',
            description: 'Maximum exact ToolSet calls across the run.',
          },
          maxParallelism: {
            title: 'Parallel call budget',
            description: 'Maximum tool calls accepted from one provider turn.',
          },
        },
      },
      config: {
        slot: {
          title: 'Model slot',
          description:
            'References a stable installed AI model slot; ad hoc provider parameters are not accepted.',
        },
        temperature: {
          title: 'Temperature',
          description:
            'Sampling temperature from 0 to 2; when omitted, the installed model policy is used.',
        },
        maxOutputTokens: {
          title: 'Maximum output tokens',
          description:
            'Maximum tokens generated for this request; when omitted, the installation setting is used.',
        },
      },
    },
    vision: {
      matchTemplate: {
        title: 'Match template',
        description:
          'Find the best location of one immutable template image inside an explicit source image.',
        input: {
          image: {
            title: 'Source image',
            description: 'Image to search; connect Capture Window or bind an Image BlobRef.',
          },
          template: {
            title: 'Template image',
            description: 'Exact immutable template variant used for this workflow.',
          },
          region: {
            title: 'Search region',
            description: 'Ratio or pixel rectangle inside the source image.',
          },
          threshold: {
            title: 'Match threshold',
            description: 'Minimum normalized score from 0 to 1 considered matched.',
          },
        },
        output: {
          matched: {
            title: 'Matched',
            description: 'True when the best score reaches the threshold.',
          },
          score: { title: 'Score', description: 'Best normalized correlation score.' },
          center: { title: 'Center', description: 'Pixel center of the best candidate.' },
          bounds: { title: 'Bounds', description: 'Pixel bounds of the best candidate.' },
        },
      },
      findTemplateMatches: {
        title: 'Find template matches',
        description:
          'Return all local template matches after deterministic non-maximum suppression.',
        input: {
          image: { title: 'Source image', description: 'Image to search.' },
          template: {
            title: 'Template image',
            description: 'Exact immutable template variant used for this workflow.',
          },
          region: {
            title: 'Search region',
            description: 'Ratio or pixel rectangle inside the source image.',
          },
          threshold: {
            title: 'Match threshold',
            description: 'Minimum normalized score from 0 to 1.',
          },
          'minimum-distance': {
            title: 'Minimum distance',
            description:
              'Minimum pixel distance between retained match centers; 0 uses half the smaller template edge.',
          },
        },
        output: {
          matches: {
            title: 'Matches',
            description: 'Score-sorted list of typed template matches.',
          },
        },
      },
      compareImages: {
        title: 'Compare images',
        description:
          'Measure visual difference between two explicit images on the same logical region.',
        input: {
          before: { title: 'Before image', description: 'Baseline image.' },
          after: { title: 'After image', description: 'Image compared with the baseline.' },
          region: {
            title: 'Comparison region',
            description: 'Same ratio or pixel region resolved independently in both images.',
          },
          'grid-size': {
            title: 'Grid size',
            description: 'Box-average grid width and height, from 1 to 256.',
          },
          'cell-delta': {
            title: 'Cell delta',
            description: 'Per-channel 8-bit difference required to count a grid cell as changed.',
          },
        },
        output: {
          'changed-ratio': {
            title: 'Changed ratio',
            description:
              'Fraction of grid cells whose maximum channel delta exceeds the threshold.',
          },
          'mean-difference': {
            title: 'Mean difference',
            description: 'Mean absolute channel difference normalized to 0–1.',
          },
        },
      },
      decodeQR: {
        title: 'Decode QR codes',
        description: 'Decode every QR code found in an explicit image region.',
        input: {
          image: { title: 'Source image', description: 'Image containing QR codes.' },
          region: { title: 'Decode region', description: 'Ratio or pixel rectangle to decode.' },
        },
        output: { codes: { title: 'QR codes', description: 'Decoded text and locator points.' } },
      },
      analyzeColor: {
        title: 'Analyze color',
        description: 'Count pixels inside one explicit RGB or HSV range.',
        input: {
          image: { title: 'Source image', description: 'Image to analyze.' },
          range: { title: 'Color range', description: 'Inclusive RGB or HSV channel bounds.' },
          region: { title: 'Analysis region', description: 'Ratio or pixel rectangle to scan.' },
        },
        output: {
          'pixel-count': { title: 'Pixel count', description: 'Number of matching pixels.' },
          fraction: {
            title: 'Fraction',
            description: 'Matching pixels divided by all pixels in the region.',
          },
          centroid: {
            title: 'Centroid',
            description: 'Pixel centroid of matching pixels; zero when none match.',
          },
        },
      },
      findColorBlobs: {
        title: 'Find color blobs',
        description: 'Extract deterministic four-connected components from an RGB or HSV mask.',
        input: {
          image: { title: 'Source image', description: 'Image to analyze.' },
          range: { title: 'Color range', description: 'Inclusive RGB or HSV channel bounds.' },
          region: { title: 'Analysis region', description: 'Ratio or pixel rectangle to scan.' },
          'minimum-area': {
            title: 'Minimum area',
            description: 'Discard components smaller than this pixel count.',
          },
        },
        output: {
          blobs: {
            title: 'Color blobs',
            description: 'Components sorted by area, then vertical and horizontal position.',
          },
        },
      },
      trackDualColorBar: {
        title: 'Track dual-color bar',
        description:
          'Track a narrow cursor relative to a wide target bar using column clusters in an explicit image region.',
        input: {
          image: { title: 'Source image', description: 'Image to analyze.' },
          'inner-range': {
            title: 'Cursor color',
            description: 'RGB or HSV range for the narrow cursor.',
          },
          'outer-range': {
            title: 'Target color',
            description: 'RGB or HSV range for the wide target bar.',
          },
          region: {
            title: 'Tracking region',
            description: 'Ratio or pixel rectangle containing the two-color bar.',
          },
          'inner-minimum-width': {
            title: 'Minimum cursor width',
            description: 'Minimum valid cursor cluster width in pixels.',
          },
          'inner-maximum-width': {
            title: 'Maximum cursor width',
            description:
              'Maximum valid cursor cluster width in pixels; zero selects automatic sizing.',
          },
          'outer-minimum-width': {
            title: 'Minimum target width',
            description:
              'Minimum valid target cluster width in pixels; zero selects automatic sizing.',
          },
          'band-height-ratio': {
            title: 'Band height ratio',
            description: 'Target scan-band height relative to the region.',
          },
          'band-inner-height-ratio': {
            title: 'Cursor height ratio',
            description: 'Target scan-band height relative to the cursor height.',
          },
          'inner-confidence-weight': {
            title: 'Cursor confidence weight',
            description: 'Cursor contribution to combined confidence.',
          },
          'outer-confidence-weight': {
            title: 'Target confidence weight',
            description: 'Target contribution to combined confidence.',
          },
        },
        output: {
          found: {
            title: 'Found',
            description: 'Whether both cursor and target clusters are valid.',
          },
          'inner-x': { title: 'Cursor X', description: 'Cursor center X in source-image pixels.' },
          'outer-x': { title: 'Target X', description: 'Target center X in source-image pixels.' },
          'outer-width': { title: 'Target width', description: 'Target cluster width in pixels.' },
          confidence: { title: 'Confidence', description: 'Combined detection confidence.' },
          'inner-pixels': {
            title: 'Cursor pixels',
            description: 'Pixels matching the cursor range.',
          },
          'outer-pixels': {
            title: 'Target pixels',
            description: 'Pixels matching the target range.',
          },
        },
      },
    },
    text: {
      concat: {
        title: 'Concatenate',
        description:
          'Combine two strings into one value. This is a data function and has no control-flow output.',
      },
    },
    conversion: {
      blobToStream: {
        title: 'Blob to stream',
        description: 'Open durable binary content as a leased runtime stream.',
      },
      streamToBlob: {
        title: 'Stream to blob',
        description: 'Consume a leased runtime stream and commit it as durable binary content.',
        config: {
          mediaType: {
            title: 'Media type',
            description: 'MIME type recorded with the durable blob, for example image/png.',
          },
        },
      },
    },
    random: {
      integer: {
        title: 'Random integer',
        description:
          'Observe a cryptographically seeded integer sample and persist it in the Run record.',
      },
      number: {
        title: 'Random number',
        description: 'Observe a finite number sample and persist it in the Run record.',
      },
      boolean: {
        title: 'Random boolean',
        description: 'Observe a true/false sample using an explicit probability from zero to one.',
      },
      choice: {
        title: 'Random choice',
        description:
          'Observe one unbiased item from a non-empty typed list and persist the result.',
      },
    },
    time: {
      observe: {
        title: 'Observe time',
        description:
          'Capture the host-provided invocation time as Unix milliseconds and persist it in the Run record.',
      },
      stopwatchStart: {
        title: 'Start stopwatch',
        description: 'Record this invocation as an explicit typed start instant.',
        output: {
          'started-at': {
            title: 'Started at',
            description: 'Unix millisecond instant recorded by this start.',
          },
        },
      },
      stopwatchRead: {
        title: 'Read stopwatch',
        description:
          'Read elapsed milliseconds from an explicit start instant without ambient process timers.',
        input: {
          'started-at': {
            title: 'Started at',
            description: 'Connect the instant from Start stopwatch.',
          },
        },
        output: {
          elapsed: { title: 'Elapsed', description: 'Milliseconds elapsed since the start.' },
        },
      },
      stopwatchStop: {
        title: 'Stop stopwatch',
        description:
          'Record final elapsed milliseconds from an explicit start instant, then continue.',
        input: {
          'started-at': {
            title: 'Started at',
            description: 'Connect the instant from Start stopwatch.',
          },
        },
        output: {
          elapsed: { title: 'Final elapsed', description: 'Total milliseconds recorded at stop.' },
        },
      },
    },
    state: {
      variable: {
        title: 'Workflow state variable',
        description: 'Select one explicitly declared typed state slot from this Workflow.',
      },
      read: {
        title: 'Read state',
        description:
          'Read the current value of a typed Run-local state slot. This is a recorded data effect with no control-flow output.',
        output: {
          result: {
            title: 'Current value',
            description: 'Current value with the exact state type.',
          },
        },
      },
      write: {
        title: 'Write state',
        description:
          'Write one exactly typed value to a Run-local state slot, then continue through Done.',
        input: {
          value: { title: 'New value', description: 'Same-type value to write to the state slot.' },
        },
        output: {
          result: {
            title: 'Written value',
            description: 'State value after the successful write.',
          },
        },
      },
      metadata: {
        title: 'State metadata',
        description:
          'Read a state slot revision and last-change time without exposing an untyped value.',
        output: {
          revision: {
            title: 'Revision',
            description: 'Revision incremented after each successful change.',
          },
          'changed-at': {
            title: 'Last changed',
            description: 'Unix millisecond time of the last change.',
          },
        },
      },
      lastChange: {
        title: 'State last change',
        description:
          'Read the Unix-millisecond instant of the state slot’s last successful update.',
        output: {
          'changed-at': {
            title: 'Last changed',
            description: 'Unix millisecond time of the last change.',
          },
        },
      },
      increment: {
        title: 'Increment state',
        description:
          'Atomically add to an Integer or Number while holding the Run State slot lock.',
        input: {
          delta: {
            title: 'Delta',
            description: 'Same-type numeric value to add to current state.',
          },
        },
        output: {
          result: {
            title: 'Updated value',
            description: 'State value after the atomic increment.',
          },
        },
      },
    },
    script: {
      execute: {
        title: 'Run script',
        description:
          'Execute JavaScript in a one-shot isolated worker with canonical JSON input and output.',
        config: {
          source: {
            title: 'JavaScript source',
            description:
              'Return one JSON-compatible value. The guest receives only an isolated copy of input.',
          },
          timeoutMilliseconds: {
            title: 'Timeout (milliseconds)',
            description: 'Hard wall-time limit for this isolated script attempt.',
          },
        },
      },
    },
    filesystem: {
      readText: {
        title: 'Read workspace text',
        description: 'Read bounded text from the Yotta-managed workflow file workspace.',
      },
      readJSON: {
        title: 'Read workspace JSON',
        description:
          'Read and strictly parse one UTF-8 JSON document from the workflow file workspace.',
      },
      stat: {
        title: 'Inspect workspace file',
        description: 'Read canonical metadata without exposing an ambient host filesystem path.',
      },
      loadImage: {
        title: 'Load workspace image',
        description:
          'Read a PNG in chunks from managed workflow files and commit a durable Image BlobRef.',
        input: {
          path: {
            title: 'Relative path',
            description: 'PNG path relative to managed workflow files.',
          },
        },
        output: {
          image: { title: 'Image', description: 'Validated and durable Image BlobRef.' },
          metadata: {
            title: 'File metadata',
            description: 'Canonical metadata for the loaded file.',
          },
        },
      },
      saveImage: {
        title: 'Save image to workspace',
        description:
          'Write a durable Image BlobRef in chunks to the managed workflow file workspace.',
        input: {
          image: { title: 'Image', description: 'Durable Image BlobRef to write.' },
          path: {
            title: 'Relative path',
            description: 'PNG path relative to managed workflow files.',
          },
        },
        output: {
          metadata: {
            title: 'File metadata',
            description: 'Canonical metadata for the written file.',
          },
        },
        config: {
          overwrite: {
            title: 'Overwrite existing file',
            description:
              'Replace an existing regular workflow file without following symbolic links.',
          },
        },
      },
      config: {
        encoding: {
          title: 'Text encoding',
          description: 'Decode as UTF-8, GBK, or detect UTF-8 before falling back to GBK.',
        },
        maxBytes: {
          title: 'Maximum bytes',
          description: 'Reject the file before reading when it exceeds this bounded byte budget.',
        },
      },
    },
    network: {
      httpGet: {
        title: 'HTTP GET',
        description:
          'Read UTF-8 text from one explicitly installed origin using a relative path. Redirects, cookies, credentials, proxies, and arbitrary hosts are unavailable.',
        config: {
          slot: {
            title: 'HTTP origin slot',
            description: 'Select the exact installed and consented HTTP origin for this request.',
          },
        },
      },
    },
    application: {
      launch: {
        title: 'Launch installed application',
        description:
          'Launch the user-authorized executable and fixed arguments from Settings. Workflows cannot supply a path, arguments, or command line.',
      },
      terminate: {
        title: 'Terminate installed application',
        description:
          'Terminate only processes that belong to the user-authorized installation path, and return the count.',
      },
      config: {
        slot: {
          title: 'Application slot',
          description: 'Select an installed and explicitly consented desktop application.',
        },
      },
    },
    automation: {
      config: {
        slot: {
          title: 'Window target slot',
          description:
            'Select an explicitly consented target bound to an authorized application path, window title, and window class.',
        },
      },
      clickPointer: {
        title: 'Click pointer',
        description: 'Perform one atomic click inside the exact installed window.',
      },
      movePointer: {
        title: 'Move pointer',
        description: 'Move the pointer to a coordinate inside the exact installed window.',
      },
      scrollPointer: {
        title: 'Scroll pointer',
        description: 'Perform a bounded scroll inside the exact installed window.',
      },
      dragPointer: {
        title: 'Drag pointer',
        description:
          'Perform a cancellable press, move, and release inside the exact installed window.',
      },
      movePointerRelative: {
        title: 'Move pointer relatively',
        description: 'Send a bounded relative pointer movement to the exact installed window.',
      },
      pressKeys: {
        title: 'Press key chord',
        description: 'Atomically press and reverse-release a canonical set of keys.',
      },
      holdKeys: {
        title: 'Hold keys',
        description:
          'Press canonical keys and return a Run-owned lease. Connect a release node; terminal cleanup also releases it.',
      },
      holdPointerButton: {
        title: 'Hold pointer button',
        description:
          'Hold a pointer button at the target coordinate and return a Run-owned lease with fail-safe cleanup.',
      },
      releaseHeldInput: {
        title: 'Release held input',
        description:
          'Consume a held-input lease and release every key or pointer button owned by it.',
      },
      typeText: {
        title: 'Type text',
        description:
          'Inject bounded Unicode text into the exact installed window without using the clipboard.',
      },
      activateWindow: {
        title: 'Activate target',
        description:
          'Reverify the installed target, then foreground the exact desktop window or start the installed Android package. Failure routes through Failed.',
      },
      closeWindow: {
        title: 'Close window',
        description:
          'Resolve the exact installed target again and send a close request to that window.',
      },
      moveResizeWindow: {
        title: 'Move and resize window',
        description:
          'Resolve the exact installed target again and set its screen-pixel position and size.',
      },
      maximizeWindow: {
        title: 'Maximize window',
        description: 'Resolve the exact installed target again and maximize its window.',
      },
      minimizeWindow: {
        title: 'Minimize window',
        description: 'Resolve the exact installed target again and minimize its window.',
      },
      restoreWindow: {
        title: 'Restore window',
        description: 'Resolve the exact installed target again and restore its window.',
      },
      getWindowState: {
        title: 'Read window state',
        description:
          'Read the exact installed target window state, foreground flag, screen position, and size.',
      },
      waitWindow: {
        title: 'Wait for window',
        description:
          'Wait up to the supplied timeout for the authorized application path and window selector to match.',
      },
      waitWindowGone: {
        title: 'Wait for window to disappear',
        description:
          'Wait up to the supplied timeout for every matching installed-target window to disappear.',
      },
      stopTargetApp: {
        title: 'Stop target app',
        description:
          'Reverify the installed target and force-stop its exact Android package. Unsupported desktop adapters fail admission.',
      },
      captureWindow: {
        title: 'Capture window',
        description:
          'Reverify the exact installed window, capture it as PNG through the configured backend, and commit a durable Image BlobRef.',
      },
      waitTemplate: {
        title: 'Wait for template',
        description:
          'Capture fresh frames from an exact window until the template appears or time expires.',
        input: {
          template: { title: 'Template image', description: 'Durable template image to wait for.' },
          region: {
            title: 'Search region',
            description: 'Region of the captured window to search.',
          },
          threshold: {
            title: 'Match threshold',
            description: 'Score from 0 to 1 required to report a match.',
          },
          timeout: {
            title: 'Wait timeout',
            description: 'Maximum wait in milliseconds; 0 checks one frame.',
          },
          'poll-interval': {
            title: 'Poll interval',
            description: 'Milliseconds between fresh frame checks.',
          },
          'settle-duration': {
            title: 'Settle duration',
            description: 'Milliseconds to wait and relocate after the first match.',
          },
        },
        output: {
          matched: {
            title: 'Matched',
            description: 'Whether the template met the threshold in the final frame.',
          },
          score: { title: 'Match score', description: 'Template score from the final frame.' },
          center: {
            title: 'Center',
            description: 'Pixel center of the matched bounds in the capture.',
          },
          bounds: {
            title: 'Matched bounds',
            description: 'Pixel bounds of the match in the capture.',
          },
        },
      },
      clickTemplate: {
        title: 'Click template',
        description:
          'Wait for a stable template match, then click its center in the same exact window.',
        input: {
          template: {
            title: 'Template image',
            description: 'Durable template image to wait for and click.',
          },
          region: {
            title: 'Search region',
            description: 'Region of the captured window to search.',
          },
          threshold: {
            title: 'Match threshold',
            description: 'Score from 0 to 1 required before clicking.',
          },
          timeout: {
            title: 'Wait timeout',
            description: 'Maximum wait in milliseconds; 0 checks one frame.',
          },
          'poll-interval': {
            title: 'Poll interval',
            description: 'Milliseconds between fresh frame checks.',
          },
          'settle-duration': {
            title: 'Settle duration',
            description: 'Milliseconds to wait and relocate after the first match.',
          },
          button: {
            title: 'Pointer button',
            description: 'Use the left, right, or middle pointer button.',
          },
          'hold-duration': {
            title: 'Hold duration',
            description: 'Milliseconds between button press and release.',
          },
        },
        output: {
          matched: {
            title: 'Matched',
            description: 'Whether the final frame met the threshold before clicking.',
          },
          score: { title: 'Match score', description: 'Final template score before clicking.' },
          center: {
            title: 'Click center',
            description: 'Pixel center converted to the exact-window click position.',
          },
          bounds: {
            title: 'Matched bounds',
            description: 'Pixel bounds of the match before clicking.',
          },
        },
      },
      waitTemplateGone: {
        title: 'Wait for template to disappear',
        description:
          'Capture fresh frames from an exact window until the template disappears or time expires.',
        input: {
          template: {
            title: 'Template image',
            description: 'Durable template image expected to disappear.',
          },
          region: {
            title: 'Search region',
            description: 'Region of the captured window to search.',
          },
          threshold: {
            title: 'Match threshold',
            description: 'A lower score means the template has disappeared.',
          },
          timeout: {
            title: 'Wait timeout',
            description: 'Maximum wait in milliseconds; 0 checks one frame.',
          },
          'poll-interval': {
            title: 'Poll interval',
            description: 'Milliseconds between fresh frame checks.',
          },
        },
        output: {
          matched: {
            title: 'Still matched',
            description: 'Whether the template still met the threshold in the final frame.',
          },
          score: { title: 'Match score', description: 'Template score from the final frame.' },
          center: {
            title: 'Last center',
            description: 'Pixel center from the final template match.',
          },
          bounds: {
            title: 'Last matched bounds',
            description: 'Pixel bounds from the final template match.',
          },
        },
      },
      waitStable: {
        title: 'Wait for stable frame',
        description:
          'Capture the exact target until a region stays below the change threshold for the stable duration.',
        input: {
          region: {
            title: 'Observation region',
            description: 'Region of the exact target frame to compare.',
          },
          threshold: {
            title: 'Change threshold',
            description: 'A changed-cell ratio at or below this value is stable.',
          },
          timeout: {
            title: 'Wait timeout',
            description: 'Maximum milliseconds to wait for stability.',
          },
          'poll-interval': {
            title: 'Poll interval',
            description: 'Milliseconds between fresh captures.',
          },
          'grid-size': {
            title: 'Sample grid',
            description: 'Bounded downsample cell count on each axis.',
          },
          'cell-delta': {
            title: 'Cell delta',
            description: 'Color difference that marks one grid cell changed.',
          },
          'stable-duration': {
            title: 'Stable duration',
            description: 'Milliseconds the region must remain stable.',
          },
        },
        output: {
          'changed-ratio': {
            title: 'Changed ratio',
            description: 'Ratio of changed cells in the final frame pair.',
          },
          'mean-difference': {
            title: 'Mean difference',
            description: 'Mean grid color difference in the final frame pair.',
          },
        },
      },
      waitChange: {
        title: 'Wait for frame change',
        description:
          'Capture the exact target until a region changes from its baseline by the requested threshold.',
        input: {
          region: {
            title: 'Observation region',
            description: 'Region of the exact target frame to compare.',
          },
          threshold: {
            title: 'Change threshold',
            description: 'Report changed when the changed-cell ratio reaches this value.',
          },
          timeout: {
            title: 'Wait timeout',
            description: 'Maximum milliseconds to wait for a change.',
          },
          'poll-interval': {
            title: 'Poll interval',
            description: 'Milliseconds between fresh captures.',
          },
          'grid-size': {
            title: 'Sample grid',
            description: 'Bounded downsample cell count on each axis.',
          },
          'cell-delta': {
            title: 'Cell delta',
            description: 'Color difference that marks one grid cell changed.',
          },
        },
        output: {
          'changed-ratio': {
            title: 'Changed ratio',
            description: 'Ratio of cells changed from the baseline frame.',
          },
          'mean-difference': {
            title: 'Mean difference',
            description: 'Mean grid color difference from the baseline frame.',
          },
        },
      },
      playInputClip: {
        title: 'Play input clip',
        description:
          'Read a validated InputClip BlobRef and replay every event through one exclusive exact-target playback session.',
      },
      playMacro: {
        title: 'Play keyboard macro',
        description:
          'Validate an atomic macro and preserve its true down, up, and sleep order in one exclusive exact-target session.',
      },
    },
    observability: {
      log: {
        title: 'Write log',
        description:
          'Write one bounded, Run-attributed message and record only its digest in the action journal.',
        config: {
          message: {
            title: 'Message',
            description:
              'Set the message in the Inspector; a connected input overrides this value.',
          },
          level: {
            title: 'Log level',
            description: 'Select debug, info, warning, or error severity.',
          },
        },
      },
    },
    event: {
      runStarted: {
        title: 'Run started',
        description: 'Emit Started exactly once when this Program Run begins.',
      },
    },
    control: {
      throw: {
        title: 'Fail workflow',
        description:
          'Terminate this branch with the stable control.thrown failure code and no success output.',
      },
      branch: {
        title: 'Branch',
        description: 'Continue through exactly one of the True or False routes.',
      },
      delay: {
        title: 'Delay',
        description:
          'Wait for a bounded duration with cooperative Run cancellation, then emit Done.',
      },
      endBranch: {
        title: 'End branch',
        description: 'Explicitly finish this control-flow branch without emitting another signal.',
      },
      repeat: {
        title: 'Repeat',
        description:
          'Run an isolated activation a bounded number of times. Body drains before the next iteration; Break and Continue target this exact region.',
      },
      forEach: {
        title: 'For each',
        description:
          'Run an isolated activation once per typed list item and expose the current Index and Item.',
      },
      retry: {
        title: 'Retry region',
        description:
          'Retry only failures explicitly routed back to this region. Completed and Exhausted are separate control results.',
      },
      switch: {
        title: 'Typed switch',
        description:
          'Compare a configurable number of same-typed cases in order and emit the first matching route or Default.',
        config: {
          caseCount: {
            title: 'Case count',
            description:
              'Creates 1–32 stable case inputs and matching execution routes; reducing the count removes out-of-range connections.',
          },
        },
        input: {
          value: {
            title: 'Match value',
            description: 'Typed value that selects the control-flow output.',
          },
          'case-1': { title: 'Case 1', description: 'First optional value of the same type.' },
          'case-2': { title: 'Case 2', description: 'Second optional value of the same type.' },
          'case-3': { title: 'Case 3', description: 'Third optional value of the same type.' },
          'case-4': { title: 'Case 4', description: 'Fourth optional value of the same type.' },
          'case-5': { title: 'Case 5', description: 'Fifth optional value of the same type.' },
          'case-6': { title: 'Case 6', description: 'Sixth optional value of the same type.' },
          'case-7': { title: 'Case 7', description: 'Seventh optional value of the same type.' },
          'case-8': { title: 'Case 8', description: 'Eighth optional value of the same type.' },
        },
      },
    },

    structure: {
      breakPoint: {
        title: 'Break point',
        description: 'Exposes typed X, Y, and unit outputs from a point.',
      },
      breakRegion: {
        title: 'Break region',
        description: 'Exposes typed position, size, and unit outputs from a region.',
      },
      breakTemplateMatch: {
        title: 'Break template match',
        description: 'Exposes score, center, and bounds from a template match.',
      },
      breakQRCode: {
        title: 'Break QR code',
        description: 'Exposes decoded text and the list of location points.',
      },
      breakColorBlob: {
        title: 'Break color blob',
        description: 'Exposes area, center, and bounds from a color blob.',
      },
      breakFileMetadata: {
        title: 'Break file metadata',
        description: 'Exposes path, name, media type, size, modified time, and directory flag.',
      },
    },
    builtin: {
      'collection-append': {
        title: 'List Append',
        description:
          'Add one item to the end, producing a NEW list (the original is unchanged; to accumulate, store it back with Set Variable using type "any").',
      },
      'collection-contains': {
        title: 'List Contains',
        description:
          'Whether the list has an item equal to Value. Same rules as the Equals node: same types compare directly, different types compare as text.',
      },
      'collection-get': {
        title: 'List Get',
        description:
          'Take the item at Index (counting from 0). Out-of-range gives null — an item can itself be null, so check List Length first to tell them apart.',
      },
      'collection-join': {
        title: 'Join',
        description:
          'Join list items into one piece of text with a separator. Non-text items are converted automatically.',
      },
      'collection-length': {
        title: 'List Length',
        description: 'Count the items in a list. A non-list counts as 0.',
      },
      'collection-slice': {
        title: 'List Slice',
        description:
          'Take Count items starting at Start, as a new list. Count -1 (default) takes to the end, 0 gives an empty list. Out-of-range Start gives an empty list.',
      },
      'collection-split': {
        title: 'Split',
        description:
          'Split text into a list by a separator. Empty text gives an empty list; an empty separator splits into individual characters (CJK-safe).',
      },
      'comparison-equal': {
        title: 'Equals',
        description:
          'Tells whether two values are equal, giving true/false. Accepts any type; same types compare directly, while different types are both turned into text first (so the number 1 and the text "1" count as equal).',
      },
      'comparison-greater-or-equal': {
        title: 'Greater or equal',
        description: 'Tells whether A ≥ B, giving true/false.',
      },
      'comparison-greater-than': {
        title: 'Greater than',
        description: 'Tells whether A > B, giving true/false.',
      },
      'comparison-less-or-equal': {
        title: 'Less or equal',
        description: 'Tells whether A ≤ B, giving true/false.',
      },
      'comparison-less-than': {
        title: 'Less than',
        description: 'Tells whether A < B, giving true/false.',
      },
      'comparison-not-equal': {
        title: 'Not equals',
        description:
          'Tells whether two values are not equal, giving true/false. Compares the same way as Equals: same types directly, different types both turned into text first.',
      },
      'conversion-string-to-boolean': {
        title: 'To bool',
        description:
          'Turns a value into true/false. An empty value, the number 0, and empty text count as false; everything else counts as true.',
      },
      'conversion-string-to-number': {
        title: 'To number',
        description:
          'Turns a value into a number, e.g. the text "12.5" becomes 12.5 and true becomes 1. Anything that cannot convert (like plain letters) gives 0.',
      },
      'conversion-string-to-integer': {
        title: 'Text to integer',
        description:
          'Strictly parses a safe integer. Decimals, whitespace, and out-of-range values fail.',
      },
      'conversion-truncate-to-integer': {
        title: 'Truncate to integer',
        description:
          'Drops the fractional part and checks the safe integer range; -1.9 becomes -1.',
      },
      'conversion-floor-to-integer': {
        title: 'Floor to integer',
        description: 'Rounds toward negative infinity and checks the safe integer range.',
      },
      'conversion-ceiling-to-integer': {
        title: 'Ceil to integer',
        description: 'Rounds toward positive infinity and checks the safe integer range.',
      },
      'conversion-round-to-integer': {
        title: 'Round to integer',
        description:
          'Rounds to the nearest integer, ties away from zero, and checks the safe range.',
      },
      'conversion-to-string': {
        title: 'To string',
        description:
          'Turns any value into text. Numbers, true/false, points and more all convert; an empty value becomes "null".',
      },
      'geometry-make-point': {
        title: 'Make Point',
        description:
          'Build a coordinate value from two numbers (X, Y) and a unit. Wire it to nodes that need a coordinate such as Click or Scroll. Use it when you need to compute coordinates dynamically.',
      },
      'geometry-offset-point': {
        title: 'Offset Point',
        description:
          'Add horizontal and vertical offsets to a coordinate. Ratio points are clamped inside the screen; pixel points keep their pixel unit.',
      },
      'geometry-point-distance': {
        title: 'Point Distance',
        description:
          'Calculate the straight-line distance between two coordinates. Useful for checking whether a detection is close enough to a target point.',
      },
      'geometry-region-around-point': {
        title: 'ROI Around Point',
        description:
          'Create an ROI for screenshot or detection around a center point. Width and height are percentages and the result is clamped inside the screen.',
      },
      'json-parse': {
        title: 'Parse JSON',
        description:
          'Parses JSON text into a structured value that can feed JsonPath, Fetch request bodies, or other JSON inputs.',
      },
      'json-path': {
        title: 'JSON path',
        description:
          'Extracts fields or array items from a JSON value. Supports $, .field, [index], and [*], for example $.items[0].url or $.items[*].url.',
      },
      'json-stringify': {
        title: 'To JSON text',
        description: 'Serializes any value to JSON text for logging, API calls, or file output.',
      },
      'logic-and': {
        title: 'Logical AND',
        description:
          'Gives true only when both conditions are true, otherwise false. An unconnected input defaults to true so it does not affect the result.',
      },
      'logic-not': {
        title: 'Logical NOT',
        description: 'Flips true/false: true becomes false and false becomes true.',
      },
      'logic-or': {
        title: 'Logical OR',
        description:
          'Gives true when at least one condition is true, and false only when both are false.',
      },
      'logic-select': {
        title: 'Select (ternary)',
        description:
          'Looks at the condition and picks one of two values to output: A when the condition is true, B when it is false. A and B can be any type.',
      },
      'math-absolute': {
        title: 'Abs',
        description: 'Absolute value of X (negatives become positive).',
      },
      'math-add': { title: 'Add', description: 'Adds two numbers and gives the sum.' },
      'math-integer-add': {
        title: 'Integer add',
        description: 'Adds two integers and preserves the integer type; overflow fails.',
      },
      'math-integer-subtract': {
        title: 'Integer subtract',
        description: 'Subtracts integer B from A and preserves the integer type; overflow fails.',
      },
      'math-integer-multiply': {
        title: 'Integer multiply',
        description: 'Multiplies two integers and preserves the integer type; overflow fails.',
      },
      'math-integer-modulo': {
        title: 'Integer modulo',
        description: 'Computes an integer remainder; a zero divisor fails.',
      },
      'math-integer-negate': {
        title: 'Integer negate',
        description: 'Flips an integer sign while preserving its type; overflow fails.',
      },
      'math-integer-absolute': {
        title: 'Integer absolute',
        description: 'Returns an integer absolute value while preserving its type.',
      },
      'math-integer-minimum': {
        title: 'Minimum integer',
        description: 'Returns the smaller integer and preserves the integer type.',
      },
      'math-integer-maximum': {
        title: 'Maximum integer',
        description: 'Returns the larger integer and preserves the integer type.',
      },
      'math-integer-clamp': {
        title: 'Clamp integer',
        description: 'Clamps an integer between minimum and maximum while preserving its type.',
      },
      'math-ceiling': { title: 'Ceil', description: 'Round X up: 3.2 gives 4, -3.7 gives -3.' },
      'math-clamp': {
        title: 'Clamp',
        description:
          'Limit X to the Min~Max range: below Min gives Min, above Max gives Max, otherwise X. Min/Max swap automatically if reversed.',
      },
      'math-divide': {
        title: 'Divide',
        description:
          'Divides A by B and gives the quotient. When the divisor is 0 the result is NaN (not a number).',
      },
      'math-floor': { title: 'Floor', description: 'Round X down: 3.7 gives 3, -3.2 gives -4.' },
      'math-maximum': { title: 'Max', description: 'The larger of two numbers.' },
      'math-minimum': { title: 'Min', description: 'The smaller of two numbers.' },
      'math-modulo': {
        title: 'Modulo',
        description: 'Gives the remainder of A divided by B; works with decimals too.',
      },
      'math-multiply': {
        title: 'Multiply',
        description: 'Multiplies two numbers and gives the product.',
      },
      'math-negate': {
        title: 'Negate',
        description: 'Flips the sign of a number: positive becomes negative and vice versa.',
      },
      'math-power': {
        title: 'Pow',
        description:
          'Base raised to Exp. Math conventions apply: negative base with fractional exponent gives NaN, 0 to a negative power gives Infinity, 0^0 gives 1.',
      },
      'math-round': {
        title: 'Round',
        description:
          'Round X. Digits=0 rounds to integer; 2 keeps two decimals; -2 rounds to hundreds (12345 gives 12300). Digits is capped at +/-15 (beyond float precision).',
      },
      'math-square-root': {
        title: 'Sqrt',
        description: 'Square root of X. Negative X gives NaN (wire an Abs node first if needed).',
      },
      'math-subtract': {
        title: 'Subtract',
        description: 'Subtracts B from A and gives the difference.',
      },
      'text-contains': {
        title: 'Contains',
        description:
          'Tells whether the needle text appears somewhere inside the haystack, giving true/false. Case-sensitive; non-text inputs are turned into text first.',
      },
      'text-ends-with': {
        title: 'Ends With',
        description: 'Whether the text ends with Suffix. Empty Suffix is always true.',
      },
      'text-index-of': {
        title: 'Index Of',
        description:
          'Position of the first occurrence of Sub in the text (counting from 0, a CJK character counts as 1). -1 when not found. To just test "contains", use the Contains node.',
      },
      'text-length': {
        title: 'String length',
        description:
          'Counts how long a piece of text is and gives the character count. A CJK character counts as 1, matching the position semantics of Substring and Index Of.',
      },
      'text-lowercase': { title: 'To Lower', description: 'Convert letters to lowercase.' },
      'text-regex-extract': {
        title: 'Regex Extract',
        description:
          'Extract the first match of the regular expression; with capture groups, group 1 is taken. No match or an invalid pattern gives an empty string (invalid patterns also log a warning).',
      },
      'text-regex-match': {
        title: 'Regex Match',
        description:
          "Whether any part of the text matches the regular expression (search semantics: b matches abc). For a full match wrap the pattern in ^ and {'$'}. An invalid pattern always gives false and logs a warning.",
      },
      'text-replace': {
        title: 'Replace',
        description:
          'Replace Old with New in the text. With "Replace all" on it replaces every occurrence, off only the first. Empty Old returns the text unchanged.',
      },
      'text-starts-with': {
        title: 'Starts With',
        description: 'Whether the text starts with Prefix. Empty Prefix is always true.',
      },
      'text-substring': {
        title: 'Substring',
        description:
          'Take Length characters starting at Start (a CJK character counts as 1). Length -1 (default) takes to the end, 0 gives an empty string. Out-of-range Start gives an empty string.',
      },
      'text-trim': {
        title: 'Trim',
        description: 'Remove spaces, newlines and tabs from both ends of the text.',
      },
      'text-uppercase': { title: 'To Upper', description: 'Convert letters to uppercase.' },
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
    group_hint: {
      system: 'Affects Yotta-wide execution and system tools.',
      recording: 'Captured by the low-level keyboard hook and not forwarded to the target app.',
      action: 'Triggers an independent action directly.',
      schedule: 'Provides a manual trigger for scheduled work.',
      editor: 'Active only while the workflow editor has focus.',
    },
    status: {
      register_failed: 'Registration failed',
      unbound: 'Unbound',
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
        'Strong-stop / calibrate / recording start / recording stop / recording pause will return to factory defaults.',
      reset_ok: 'Reset',
    },
    label: {
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
  // backend ValidationError.Code → user-facing message.
  // Params: {param} placeholders use vue-i18n named-interpolation. Missing keys fall back to backend Message field.
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
    NODE_CONTRACT_MISMATCH: 'The node contract digest does not match',
    UNKNOWN_PORT: 'An edge references an unknown port',
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
    ASSET_QUERY_INVALID: 'The asset query is invalid; try again',
    UNKNOWN_ERROR: 'An unknown error occurred',
    TRANSPORT_TIMEOUT: 'The request timed out; try again',
    TRANSPORT_UNAVAILABLE: 'The backend connection is unavailable; restart Yotta',
    admission: {
      target_unavailable:
        'A required target is unavailable. Check that it is installed, authorized, and currently matches.',
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
      identity_changed: 'The installed application is currently unavailable',
      launch_failed: 'The application failed to launch',
      terminate_failed: 'The application failed to terminate',
      unsupported_host: 'Application control is unsupported on this host',
      contract_violation: 'The application-control provider violated its contract',
    },
    automation: {
      invalid_request: 'The automation request is invalid',
      identity_changed: 'The automation target identity changed',
      target_not_found: 'The automation target was not found',
      target_ambiguous: 'The automation target match is ambiguous',
      input_failed: 'The input operation failed',
      window_failed: 'The window operation failed',
      capture_failed: 'Capture failed',
      playback_failed: 'Input playback failed',
      playback_busy: 'Input playback is busy',
      unsupported_host: 'This automation capability is unsupported on the host',
      contract_violation: 'The automation provider violated its contract',
    },
    network: {
      invalid_request: 'The network request is invalid',
      resolution_denied: 'The destination was denied by network policy',
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
  workflow: {
    workspace_tools: 'Workspace tools',
    sidebar: {
      resize_workspace: 'Resize workspace sidebar',
      resize_inspector: 'Resize properties sidebar',
      hide_inspector: 'Hide properties sidebar',
      show_inspector: 'Show properties sidebar',
    },
    list: {
      eyebrow: 'WORKFLOW SOURCES',
      title: 'Workflows',
      description: 'Every run compiles a saved Workflow Source into an immutable Program snapshot.',
      management_description: 'Search, filter, and manage Workflow Sources in bulk.',
      new_workflow: 'New workflow',
      library_actions: 'Workflow library actions',
      name_placeholder: 'Workflow name',
      create: 'Create',
      template_label: 'Starting template',
      template_generic: 'Generic',
      template_windows: 'Windows automation',
      template_android: 'Android automation',
      template_browser: 'Browser automation',
      template_cross_target: 'Cross-target automation',
      loading: 'Loading workflows',
      empty_title: 'Create the first workflow',
      empty_description:
        'The host creates a strict source envelope and opens it in the generated node editor.',
      name: 'Name',
      revision: 'Revision',
      source_identity: 'Source identity',
      actions: 'Actions',
      row_actions: 'More actions for workflow “{name}”',
      category: 'Category',
      tags: 'Tags',
      nodes: 'Nodes',
      columns: 'Columns',
      reset_columns: 'Restore default columns',
      no_description: 'No description',
      unclassified: 'Unclassified',
      no_tags: 'No tags',
      search_label: 'Search workflows',
      search_placeholder: 'Search by name or workflow ID',
      search_all_placeholder: 'Search name, description, category, tags, or workflow ID',
      search_action: 'Search',
      clear_search: 'Clear search',
      all_categories: 'All categories',
      all_tags: 'All tags',
      reset_filters: 'Reset filters',
      no_results_title: 'No matching workflows',
      no_results_description: 'Try a different search term. Existing workflows are unchanged.',
      sort_label: 'Sort',
      sort_name_asc: 'Name A–Z',
      sort_name_desc: 'Name Z–A',
      sort_nodes_desc: 'Most nodes first',
      sort_revision_desc: 'Highest revision first',
      sort_created_desc: 'Recently created first',
      sort_updated_desc: 'Recently modified first',
      created_at: 'Created',
      updated_at: 'Modified',
      created_any: 'Any creation date',
      created_today: 'Created today',
      created_days: 'Created in {n} days',
      updated_any: 'Any modification date',
      updated_today: 'Modified today',
      updated_days: 'Modified in {n} days',
      page_size_label: 'Per page',
      page_summary: 'Page {page} of {pages}; {total} workflows',
      result_range: 'Showing {start}–{end} of {total} workflows',
      per_page: 'Per page',
      previous_page: 'Previous',
      next_page: 'Next',
      select_page: 'Select every workflow on this page',
      select_named: 'Select workflow “{name}”',
      selected_count: '{n} workflows selected',
      selection_scope_hint:
        'Selection persists across pages; Select all affects only the current page.',
      clear_selection: 'Clear selection',
      batch_edit: 'Batch category/tags',
      batch_edit_title: 'Edit category and tags for {n} workflows',
      batch_update_result: 'Updated {updated} workflows; failed {failed}.',
      create_title: 'Create workflow',
      edit_metadata_title: 'Edit workflow information',
      edit_metadata: 'Edit information',
      description_label: 'Description',
      description_placeholder: 'Describe what this workflow does and when it should be used',
      category_placeholder: 'Select or create a category',
      tags_placeholder: 'Select or create tags',
      import_source: 'Import source bundle',
      export_source: 'Export',
      export_selected: 'Export selected',
      replace_source: 'Replace source',
      export_named: 'Export workflow “{name}”',
      replace_named: 'Replace workflow “{name}” from a source bundle',
      import_title: 'Import “{name}”?',
      import_result: 'Imported “{name}” as a new workflow.',
      export_result: 'Exported {n} Workflow Source bundle.',
      export_batch_result: 'Exported {exported}; failed {failed}.',
      replace_title: 'Replace the source of “{name}”?',
      replace_description:
        'The target workflow ID is preserved and concurrent edits are guarded by exact revision and source hash.',
      replace_result: 'Updated the target workflow with the source from “{name}”.',
      bundle_description:
        'Source bundle: {name}, revision {revision}, {blobs} referenced assets ({bytes} bytes). Installations, secrets, and machine-local target details are not imported.',
      delete_selected: 'Delete selected',
      delete_title: 'Delete {n} workflows?',
      delete_description:
        '{deletable} unreferenced workflows will be deleted; {blocked} referenced workflows are blocked. Historical run records are retained.',
      delete_all_blocked: 'Every selected workflow is referenced or running. Nothing was deleted.',
      delete_result: 'Deleted {deleted}; failed {failed}; blocked by references {blocked}.',
      reference_schedule: 'Referenced by schedule “{name}”',
      reference_launcher: 'Referenced by launcher item “{name}”',
      reference_active_run: 'Run “{name}” is queued or running',
      recovery_title: '{n} corrupt workflow sources were isolated',
      recovery_description:
        'The rest of the workspace remains available. Repair or delete only the affected object; the whole data directory is not involved.',
      recovery_repair: 'Repair',
      recovery_delete: 'Delete corrupt object',
      recovery_repair_title: 'Repair workflow source',
      recovery_repair_description:
        'Edit the original JSON and submit it. Yotta validates the complete current Workflow contract before returning it to the workflow library.',
      recovery_source_json: 'Workflow Source JSON to repair',
      recovery_validate_repair: 'Validate and repair',
      recovery_delete_title: 'Delete corrupt object “{name}”?',
      recovery_delete_description:
        'Only the isolated corrupt source is deleted. Other workflows, assets, and run records are unchanged.',
    },
    template: {
      windows: {
        hint: 'Windows onboarding template: use one Workflow Source and bind automation nodes to installed desktop-window targets.',
      },
      android: {
        hint: 'Android onboarding template: use the same nodes and runtime, binding supported operations to installed android-device targets.',
      },
      browser: {
        hint: 'Browser onboarding template: reuse the same Workflow Source and generic input nodes, bound to an installed browser-cdp page target.',
      },
      'cross-target': {
        hint: 'Cross-target onboarding template: one Workflow Source can bind separate branches to Windows, Android, and browser target slots.',
      },
      configure_targets: 'Configure targets',
    },
    editor: {
      loading: 'Loading workflow editor',
      open_failed: 'Workflow could not be opened',
      back: 'Back to workflows',
      workflow_name: 'Workflow name',
      settings: 'Workflow settings',
      tags_hint: 'Separate tags with commas',
      revision: 'rev {n}',
      unsaved: 'Unsaved',
      save_conflict: 'Save conflict: {message}',
      node_catalog: 'Node catalog',
      catalog_description: 'Click a node, or drag it onto the canvas.',
      discard_title: 'Discard workflow changes?',
      discard_confirm: 'Discard unsaved workflow changes?',
      discard_action: 'Discard changes',
      leave_title: 'Leave workflow?',
      leave_confirm: 'This workflow has unsaved changes. Save before leaving, or discard them.',
      save_and_exit: 'Save and leave',
    },
    node_search: {
      action: 'Find node',
      shortcut: 'Find a canvas node (Ctrl/⌘+F)',
      title: 'Find canvas node',
      placeholder: 'Search node names, IDs, types, or graphs',
      empty: 'Enter a query to search nodes in this Workflow Source.',
      no_results: 'No matching canvas nodes',
      result_count: '{n} results',
    },
    catalog: {
      search_placeholder: 'Search node names, types, or tags',
      no_results: 'No matching nodes',
      category: {
        ai: 'AI',
        application: 'Application',
        automation: 'Automation',
        collection: 'Collection',
        comparison: 'Comparison',
        control: 'Control flow',
        conversion: 'Conversion',
        event: 'Event',
        geometry: 'Geometry',
        io: 'Data and files',
        json: 'JSON',
        logic: 'Logic',
        math: 'Math',
        network: 'Network',
        random: 'Random',
        script: 'Script',
        state: 'State',
        text: 'Text',
        time: 'Time',
        vision: 'Vision',
        other: 'Other',
      },
    },
    snippets: {
      title: 'Snippets',
      hint: 'Reuse configured nodes. Right-click a canvas node to save it.',
      search: 'Search name, description, category, tag, or node type',
      all_categories: 'All categories',
      all_tags: 'All tags',
      empty: 'No snippets yet',
      empty_hint: 'Right-click a configured canvas node to save it as a reusable template.',
      count: '{count} items shown',
      corrupt: '{count} corrupt snippet files were isolated; other snippets remain available.',
      corrupt_hint: 'Fix or remove the listed files under data/snippets, then restart Yotta.',
      edit: 'Edit snippet',
      delete: 'Delete snippet',
      create_title: 'Save as snippet',
      edit_title: 'Edit snippet',
      name: 'Name',
      name_placeholder: 'For example: Confirm button in login window',
      description: 'Description',
      category: 'Category',
      tags: 'Tags',
      tags_placeholder: 'Separate multiple tags with commas',
      shortcut: 'Canvas shortcut',
      shortcut_hint: 'Works only on the workflow canvas. Use modifiers or F1–F12.',
      shortcut_reserved: 'This shortcut is reserved by the editor.',
      shortcut_duplicate: 'This shortcut is already assigned to another snippet.',
      shortcut_invalid: 'Use a modified key combination or F1–F12.',
      payload: 'Node template',
      payload_hint:
        'Stores the node contract version, configuration, and input bindings. Node IDs, canvas position, grants, and runtime handles are excluded.',
      delete_title: 'Delete snippet?',
      delete_hint: '“{name}” cannot be restored after deletion.',
      load_failed: 'Could not load snippet',
      save_failed: 'Could not save snippet',
      delete_failed: 'Could not delete snippet',
      insert_failed: 'Could not insert snippet',
      usage_failed: 'Could not update snippet usage metadata',
      node_unavailable: 'The required node type is not available in the current catalog.',
      contract_changed: 'The node contract changed. Reconfigure the node and update this snippet.',
    },
    quick_add: {
      title: 'Quick add',
      search: 'Search nodes or snippets',
      categories: 'Categories',
      all: 'All',
      empty: 'No matching nodes or snippets',
      hint: '↑↓ Select · Enter Add · Esc Close',
      count: '{n} items',
    },
    connection: {
      title: 'Add and connect a node',
      compatible_hint: 'Only nodes that can complete this connection are shown.',
      all_hint: 'All nodes are shown. Incompatible choices are added without a connection.',
      search: 'Search candidate nodes',
      via_port: 'Auto-connect through {port}',
      add_only: 'Add without connecting',
      no_results: 'No matching compatible nodes',
      show_all: 'Show all nodes',
      show_compatible: 'Show compatible nodes',
      show_more: 'Show {remaining} more',
      match_exact: 'Exact',
      match_assignable: 'Compatible',
      match_generic_bind: 'Inferred',
      conversion_lossless: 'Lossless conversion',
      conversion_lossy: 'Lossy conversion',
      conversion_parser: 'May fail',
      conversion_title: 'Choose an explicit conversion',
      conversion_hint:
        '{source} cannot connect directly to {target}. Choosing a strategy inserts a visible node into the graph.',
      conversion_cost: 'Conversion cost {cost}',
      conversion_failed: 'Could not insert the conversion node',
      issue: {
        direction: 'Connections must run from an output to an input',
        channel: 'The port channels differ',
        port: 'The port is missing or cannot be connected',
        type: 'The data types are incompatible',
        type_detail: '{source} cannot connect to {target}',
        carrier: 'The data carrier classes differ',
        'resource-lease': 'The resource operations cannot be transferred safely',
        instruction: 'The target control node does not accept this entry',
      },
    },
    selection: {
      count: '{count} nodes selected',
      copy: 'Copy',
      cut: 'Cut',
      paste: 'Paste',
      duplicate: 'Duplicate nodes',
      remove: 'Remove selected nodes',
      arrange: 'Arrange',
      align_left: 'Align left',
      align_right: 'Align right',
      align_top: 'Align top',
      align_bottom: 'Align bottom',
      align_horizontal: 'Align horizontal centers',
      align_vertical: 'Align vertical centers',
      distribute_horizontal: 'Distribute horizontally',
      distribute_vertical: 'Distribute vertically',
      layout_lr: 'Auto-layout left to right',
      layout_tb: 'Auto-layout top to bottom',
      layout_failed: 'Auto-layout failed',
      clipboard_failed: 'Could not read the workflow clipboard',
      collapse: 'Collapse to subgraph',
      collapse_rejected: 'Cannot collapse this selection',
      collapse_multiple_entry:
        'The selection has {count} distinct entries. Merge them into one execution entry first; a conflicting edge is selected.',
      collapse_incoming_error:
        'An error channel cannot enter a subgraph. Handle it outside or reshape the control flow first; the conflicting edge is selected.',
      collapse_invalid:
        'The current selection cannot form a subgraph. Select only executable nodes or existing graph calls.',
      collapse_input_type:
        'The data input type at the selection boundary could not be determined. Check the node contracts.',
      collapse_output_type:
        'The data output type at the selection boundary could not be determined. Check the node contracts.',
      collapse_missing_boundary:
        'The selection needs exactly one execution entry and at least one execution or error exit.',
      collapse_unknown:
        'An internal error occurred while creating the subgraph. Retry or check the current selection.',
    },
    canvas: {
      help: 'Canvas gestures',
      marquee_key: 'Left drag',
      marquee: 'Marquee-select nodes',
      add_selection: 'Add a marquee selection',
      toggle_selection: 'Toggle one node',
      pan: 'Pan the canvas',
      delete_clear: 'Delete selection / clear selection',
      clear_run_trace: 'Clear the latest run trace',
      show_minimap: 'Show minimap',
      hide_minimap: 'Hide minimap',
    },
    graphs: {
      all: 'Graphs',
      manager: 'Subgraph manager',
      manager_hint: 'Manage shared definitions, call locations, and interface health.',
      search: 'Search names, IDs, or call locations',
      search_empty: 'No matching graph definitions.',
      main: 'Main graph',
      new: 'New subgraph',
      rename: 'Rename subgraph',
      rename_definition: 'Rename subgraph definition',
      duplicate_definition: 'Duplicate subgraph definition',
      add_call: 'Add call',
      call_inspector: 'Subgraph call',
      call_actions: 'Call actions',
      duplicate_call: 'Duplicate call (share definition)',
      fork_definition: 'Create independent definition copy',
      expand_call: 'Expand call',
      expand_call_title: 'Expand this call into the current graph?',
      expand_call_hint:
        'Internal nodes, calls, annotations, and edges will be inlined atomically. The shared definition remains available, and Undo restores the call.',
      call_count: '{count} calls',
      call_locations: 'Call locations',
      call_hint: 'Inputs can be connected on the canvas or bound here for this call site.',
      no_call_inputs: 'This subgraph has no data inputs.',
      open: 'Open subgraph',
      infer_interface: 'Infer interface',
      infer_interface_hint:
        'Preview the difference between open internal endpoints and the current interface before applying it.',
      infer_interface_title: 'Apply inferred interface?',
      infer_interface_preview:
        'This will add {added} items and remove {removed}; matching endpoints keep their stable IDs and names.',
      infer_interface_confirm: 'Apply interface changes',
      infer_interface_blocked: 'Cannot apply inferred interface',
      infer_interface_blocked_hint:
        '{count} call references still use interface items that would be removed.',
      infer_not_subgraph: 'Open a subgraph first.',
      infer_multiple_entries:
        'There are multiple unconnected execution entries. Connect them into one entry first.',
      infer_missing_endpoints: 'Add a node with an unconnected in entry and signal exit first.',
      boundary_entry: 'Subgraph entry',
      boundary_exit: 'Subgraph exit',
      boundary_output: 'Subgraph data outputs',
      boundary_authoring: 'Interface projection',
      interface_title: 'Subgraph interface',
      interface_hint:
        'Explicitly publish open internal endpoints. Names can change while stable IDs remain fixed; canvas boundaries, call nodes, and the compiler read the same Source.',
      interface_empty: 'There are no data ports or named exits yet.',
      interface_unbound: 'Not bound',
      interface_section_empty: 'No interface items of this kind are published.',
      interface_name: 'Interface display name',
      bind_entry: 'Bind subgraph entry',
      unbind_entry: 'Unbind subgraph entry',
      add_interface_item: 'Add {item}',
      no_available_endpoint: 'No open internal endpoint is available',
      move_interface_up: 'Move interface item up',
      move_interface_down: 'Move interface item down',
      remove_interface_item: 'Remove interface item',
      remove_interface_blocked: 'Interface item is still referenced',
      interface_referenced: '{count} call bindings or edges still reference this interface item.',
      interface_summary: '{inputs} inputs · {outputs} outputs · {exits} exits',
      interface_unhealthy: 'The interface is not fully bound',
      add_comment: 'Add comment',
      default_name: 'Subgraph',
      delete_call: 'Delete this call',
      delete_definition: 'Delete subgraph definition',
      delete_definition_referenced:
        '{count} calls still reference it. Locate and resolve those calls first.',
      delete_definition_cascade: 'Delete definition and all calls',
      delete_definition_cascade_hint: 'This also removes all {count} calls of the definition.',
      delete_definition_cascade_title: 'Delete the definition and all calls?',
      delete_definition_cascade_confirm:
        '“{name}”, its {count} calls, and edges attached to those calls will be removed atomically. You can Undo before saving.',
      delete_title: 'Delete subgraph definition?',
      delete_hint:
        'Definition “{name}” will be removed from the Workflow Source and cannot be restored after saving.',
    },
    reroute: {
      add: 'Add reroute point',
      clear: 'Clear reroute points',
    },
    empty_canvas: {
      title: 'Start from the run root',
      description:
        'Every Run begins at Run started. Add the root, then connect action nodes from the catalog.',
      add_start: 'Add Run started',
      subgraph_title: 'Build a reusable subgraph',
      subgraph_description:
        'Add action nodes, then publish the entry, data ports, and named exits from the interface panel.',
    },
    state_panel: {
      key_chord_type: 'Key chord',
      initial_value: 'Initial value',
      initial_value_hint: 'This value initializes the state at the start of every Run.',
      title: 'Workflow state',
      hint: 'This belongs to the workflow, not the selected node.',
      empty: 'No Run state variables yet. Add one only when values must persist across nodes.',
      search: 'Search state name or type',
      no_results: 'No matching state variables',
      drag_hint: 'Drag to the canvas for Read; hold Alt while dragging for Write.',
      insert_read: 'Insert a Read node for {name}',
      insert_write: 'Insert a Write node for {name}',
      insert_last_change: 'Insert Last change for {name}',
      insert_increment: 'Insert atomic Increment for {name}',
      actions: 'More actions for state {name}',
      insert_failed: 'Could not insert the state reference',
      locate_references: 'Locate references to {name}; {count} total',
      remove_referenced: 'Remove or rebind every reference before deleting this state variable',
      type_change: 'Change the type of state {name}',
      type_change_referenced: 'Review every reference and affected edge before changing this type',
      type_change_impact:
        'This change rechecks {count} Read/Write references across graphs; Yotta will not disconnect them silently.',
      type_change_blocked:
        'This change would break {count} existing data connections. Insert conversions or adjust those edges first.',
      type_change_safe:
        'Existing edges can migrate directly. The authoritative Compiler will verify the atomic save.',
      type_change_conversion: 'Conversion required',
      type_change_incompatible: 'Incompatible',
      show_more: 'Show {remaining} more',
      promote_title: 'Promote to Run state',
      promote_action: 'Promote to state',
      promote_hint:
        'Create a {type} state and insert a connected write node. The whole edit can be undone once.',
      promote_candidate_hint: 'Create same-type state and connect a write node',
      promote_invalid_name: 'The state name is invalid',
      promote_duplicate_name: 'A state with this name already exists',
      promote_failed: 'Could not promote the output to state',
    },
    ai: {
      open: 'AI proposal',
      title: 'AI workflow proposal',
      hint: 'The model can prepare and compile a candidate. Nothing changes until you accept it.',
      close: 'Close AI proposal panel',
      profile: 'Model profile',
      request: 'Requested change',
      request_help: 'Describe the outcome. Inspected workflow data remains untrusted to the model.',
      request_placeholder: 'Add a text node, connect it, and keep existing nodes unchanged.',
      save_first: 'Save or discard local edits before starting an AI proposal.',
      no_profile: 'No evaluated tool-calling profile is available. Configure one in AI settings.',
      propose: 'Prepare proposal',
      retry: 'Prepare another proposal',
      review: 'Review candidate',
      revision: 'Revision',
      candidate: 'Candidate hash',
      changes: 'Normalized changes',
      redacted: 'Sensitive value is redacted',
      diagnostics: 'Compiler diagnostics',
      no_diagnostics: 'No compiler diagnostics',
      permissions: 'Permission delta',
      no_permission_change: 'No capability, credential, or target changes',
      audit: 'Redacted audit trace',
      audit_redaction:
        'Trace entries retain identities, digests, sizes, decisions, and usage. Raw prompts, credentials, and sensitive values are omitted.',
      turns: 'turns',
      tool_calls: 'tool calls',
      reject: 'Reject',
      accept: 'Accept exact candidate',
      permission_confirm_title: 'Accept added permissions?',
      permission_confirm:
        'This candidate adds {n} permission requirement. Accept the exact candidate?',
      refresh_failed: 'Could not reload the accepted workflow',
      status: {
        proposed: 'Awaiting review',
        accepted: 'Accepted',
        rejected: 'Rejected',
        stale: 'Revision changed',
      },
    },
    action: {
      edit: 'Edit',
      edit_named: 'Edit workflow “{name}”',
      run: 'Run',
      run_named: 'Run workflow “{name}”',
      delete_named: 'Delete workflow “{name}”',
      run_timeline: 'Run and inspect timeline',
      compile: 'Compile',
      compile_succeeded: 'Compiled',
      save: 'Save',
      saved: 'Saved',
      stop: 'Stop',
      stop_all: 'Stop all',
      refresh: 'Refresh',
      undo: 'Undo',
      redo: 'Redo',
    },
    node: {
      disabled: 'Disabled',
      show_optional_inputs: 'Show {n} optional inputs',
      hide_optional_inputs: 'Hide optional inputs',
      run_running: 'Running',
      run_waiting: 'Waiting',
      run_succeeded: 'Succeeded',
      run_failed: 'Failed',
      run_cancelled: 'Cancelled',
      run_routed: 'Routed',
    },
    node_menu: {
      title: 'Node actions',
      enable: 'Enable node',
      disable: 'Disable node',
      visual_template: 'Visual template',
      choose_template: 'Choose from library',
      capture_template: 'Capture new template',
    },
    diagnostics: {
      title: 'Compiler diagnostics',
      summary: '{n} items; select one to locate its node',
      badge: '{n} diagnostics',
      close: 'Close compiler diagnostics',
      error: 'Errors',
      warning: 'Warnings',
      info: 'Information',
      fix_set_field: 'Compiler-declared fix: set field',
      fix_remove_field: 'Compiler-declared fix: remove field',
      locate_failed: 'Could not locate the diagnostic',
    },
    debug: {
      title: 'Debugger',
      start: 'Debug',
      current: 'Paused here',
      waiting: 'Waiting for the first node',
      step: 'Step',
      continue: 'Continue',
      pause: 'Pause',
      close: 'Close debugger',
      add_breakpoint: 'Add breakpoint',
      remove_breakpoint: 'Remove breakpoint',
      inputs: 'Input snapshot',
      outputs: 'Output snapshot',
      state: 'Run state',
      queue: 'Pending queue',
      empty: 'None',
      no_runtime_data: 'This node has no input, output, Run state, or queued data to inspect.',
      runtime_data: 'Runtime data',
      technical_details: 'Technical details',
      will_execute: 'Will execute next (not executed yet)',
      just_executed: 'Just executed',
      next_queued: 'Queued next',
      current_position: 'Executing',
      end_position: 'End position',
      paused_before: 'Paused before “{node}” executes.',
      pause_pending_hint: 'Pause requested; waiting for the current node to finish.',
      running_hint: 'Running until a breakpoint is reached or the pause request takes effect.',
      finished: 'The debug run completed.',
      finished_failed: 'The debug run ended with an error. Open Logs for details.',
      finished_cancelled: 'The debug run was stopped.',
      redacted: 'content redacted',
      status_running: 'Running',
      status_pause_pending: 'Pause pending',
      status_paused: 'Paused',
      status_completed: 'Completed',
    },
    resources: {
      title: 'Workspace resources',
      hint: 'Record, capture, search, and bind resources beside the canvas without leaving this workflow.',
      open_library: 'Open the full library',
      edit: 'Edit resource “{name}”',
      use: 'Use resource “{name}”',
      empty: 'No resources in this category',
      empty_hint: 'Record or capture here; new resources appear in this list immediately.',
      capture_hint:
        'Choose the target for this capture. The visual template is saved and added to the current canvas.',
    },
    recording: {
      start: 'Record',
      record_macro: 'Record macro',
      record_precise: 'Precise recording',
      pause: 'Pause recording',
      resume: 'Resume recording',
      finish: 'Finish recording',
      start_title: 'Record workflow actions',
      mode: 'Recording mode',
      target: 'Automation target',
      start_hint:
        'The target is armed first. Switch to it and press the start shortcut; input capture begins after a 3-second countdown. The recording is saved as a resource and creates its playback node.',
      macro_title: 'Record editable macro',
      macro_hint:
        'Record atomic key down, key up, click, and sleep actions for keyboard and pointer macros.',
      precise_title: 'Record precise trajectory',
      precise_hint:
        'Preserve motion, dragging, raw mouse deltas, and complete timing as an InputClip.',
      start_failed: 'Could not start recording',
      open_calibration: 'Open input calibration',
      control_failed: 'Recording control failed',
      preview_title: 'Review recording',
      result_mode: 'Generated form',
      mode_simple: 'Editable atomic macro',
      mode_precise: 'Precise · full trajectory clip',
      action_summary: 'Keys {keys} · clicks {clicks} · moves {moves} · scrolls {scrolls}',
      trajectory_hint:
        'Precise recording is stored as one InputClip and is not expanded into canvas nodes.',
      leave_title: 'End the active recording?',
      leave_hint: 'Leaving the editor cancels the unfinished recording.',
      leave_action: 'Cancel recording and leave',
    },
    inspector: {
      title: 'Inspector',
      no_selection: 'No selection',
      remove_node: 'Remove node',
      select_hint: 'Select a node to edit its generated contract fields.',
      label: 'Label',
      label_placeholder: 'Optional display label',
      configuration: 'Configuration',
      inputs: 'Inputs',
      group_required: 'Required first',
      group_common: 'Common settings',
      group_advanced: 'Advanced settings',
      group_output: 'Outputs',
      reference_only: 'This port accepts {carrier} references through a compatible connection.',
      select_clip: 'Select an input clip',
      select_macro: 'Select a keyboard macro',
      select_template: 'Select an exact template variant',
      resource_missing: 'The library record no longer exists',
      resource_stale: 'Unavailable',
      select_target: 'Select an installed target',
      search_target: 'Search installed targets',
      no_installed_target: 'No compatible target is installed. Configure one in Settings first.',
      configure_target: 'Open installation settings',
      target_inherited: 'Inherited from workflow',
      target_overridden: 'Node override',
      target_missing: 'Target missing',
      restore_inherited: 'Restore inheritance',
      advanced: 'Advanced information',
      color_rgb: 'RGB',
      color_hsv: 'HSV',
      color_red: 'Red',
      color_green: 'Green',
      color_blue: 'Blue',
      color_hue: 'Hue',
      color_saturation: 'Saturation',
      color_value: 'Value',
      color_minimum: 'Minimum',
      color_maximum: 'Maximum',
      color_channels: 'Advanced channel ranges',
      color_sample_hint: 'The swatch is a preview; the saved value remains a strict RGB/HSV range.',
      color_unsampled_hint: 'The full range matches every color. Sample the target before running.',
      pick_color: 'Sample target',
      pick_point: 'Pick from target',
      pick_region: 'Frame on target',
      pick_failed: 'Screen picking failed',
      picker_target_required:
        'Choose the workflow default target to pick from screen, or enter a value manually.',
      region_width: 'Width',
      region_height: 'Height',
      duration_ms: 'Milliseconds (ms)',
      duration_s: 'Seconds (s)',
      duration_min: 'Minutes (min)',
      use_default: 'Use default',
      clear: 'Clear',
      record_key_chord: 'Click to record a key chord',
      record_key_chord_active: 'Press the key chord…',
      record_key_chord_hint:
        'Click, then press the key or chord to send. Yotta presses the keys in order and releases them in reverse.',
      capabilities: 'Capabilities',
      observed_status: 'Observed status',
      status_hint:
        'Status events appear in the Run timeline. They are not connectable graph ports.',
      required_value: 'Required value or connection',
      optional_value: 'Optional value',
      point_x: 'X',
      point_y: 'Y',
      point_ratio: 'Ratio',
      point_px: 'Pixels',
      state_title: 'Run state',
      state_hint: 'Typed slots are initialized separately for every Run.',
      state_name_placeholder: 'State name',
      state_add: 'Add state variable',
      state_remove: 'Remove state variable {name}',
      playback_calibration: 'Playback turn calibration',
      playback_source_counts: 'Recorded counts/360',
      playback_source_recorded: 'Fixed in the InputClip',
      playback_target_counts: 'Target counts/360',
      playback_target_custom: 'Target override',
      playback_target_active: 'Active calibration profile',
      playback_target_missing: 'Target calibration missing',
      playback_target_unsupported: 'Target does not support relative turns',
      playback_calibration_formula:
        'Relative mouse motion is converted automatically from the recorded to the local target counts/360; local calibration is not stored in the workflow.',
      playback_metadata_unavailable: 'The InputClip recording calibration could not be read.',
    },
    target_default: {
      label: 'Default automation target',
      placeholder: 'Choose once for target nodes',
      none: 'No default target',
      clear: 'Clear default automation target',
    },
    timeline: {
      run_status: 'Run {status}',
      empty: 'No attempts have been recorded yet.',
      attempt: 'attempt {n}',
      close: 'Close Run timeline',
      open: 'Run timeline',
      page: 'Page {page} of {pages} · {total} entries',
      older: 'Older',
      newer: 'Newer',
      active_attempt: 'Current node',
      executing: 'Executing',
      timeout_budget: 'timeout {value}',
      unhandled_route: 'The “{route}” output is not connected; this Run ends here',
      status: {
        'automation.template.waiting': 'Waiting for a template match',
        'automation.template.matched': 'Template matched',
        'automation.template.timeout': 'Template wait timed out',
      },
    },
    workbench: {
      diagnostics: 'Diagnostics',
      logs: 'Logs',
      timeline: 'Timeline',
      debug: 'Debug',
      close: 'Collapse runtime workbench',
      open: 'Expand runtime workbench',
      unavailable: 'No data is available for this view.',
    },
    status: {
      program: 'Program {status}',
      ready: 'Ready',
    },
    toast: {
      list_failed: 'Could not load workflows',
      create_failed: 'Could not create workflow',
      not_started: 'Workflow did not start',
      queued: 'Program queued',
      run_failed: 'Run failed',
      no_run_created: 'No Run was created.',
      edit_rejected: 'Edit rejected',
      compile_failed: 'Compile failed',
      save_failed: 'Save failed',
      debug_failed: 'Debug operation failed',
      stop_failed: 'Stop failed',
      refresh_failed: 'Timeline refresh failed',
    },
  },
  schedule: {
    workspace: {
      eyebrow: 'SCHEDULE CONTROL',
      title: 'Schedule control',
      description:
        'Manage automatic triggers, execution order, and failure policy from one operational view.',
      editor_description:
        'Configure the trigger, target workflows, and runtime boundaries. The preview reflects the saved behavior.',
      summary: 'Schedule overview',
      total: 'All schedules',
      enabled: 'Enabled',
      automatic: 'Automatic',
      targets: 'Total targets',
      showing: 'Showing {n} schedules',
      error_policy: 'Failure policy',
      no_targets: 'No target workflows selected',
      no_targets_hint:
        'Add at least one workflow. Targets run in order after the schedule triggers.',
      more_targets: '{names} and +{n}',
      never_run: 'Never triggered',
      last_run: 'Last run {value}',
      basics_hint: 'Schedule identity and availability.',
      trigger_hint: 'Controls when this schedule enters the execution queue.',
      policy_hint: 'Bound runtime and choose what happens after a target fails.',
      preview: 'Behavior preview',
      trigger_preview_hint: 'Runs with this rule after saving',
      timeout_minutes: '{n} minutes',
      no_timeout: 'No timeout',
    },
    title: 'Schedule',
    search_placeholder: 'Search schedules...',
    status_filter: 'Filter schedule status',
    filter: {
      all: 'All statuses',
      enabled: 'Enabled only',
      disabled: 'Disabled only',
    },
    status: {
      queued: 'Queued',
    },
    more_action: 'More actions for “{name}”',
    create: 'New schedule',
    back_to_list: 'Back to list',
    name_label: 'Name',
    enabled_label: 'Enabled',
    enable: 'Enable',
    disable: 'Disable',
    basics_section: 'Basics',
    targets_section: 'Target workflows',
    targets_hint: 'Run these workflows in order when triggered.',
    trigger_section: 'Trigger',
    trigger_kind_label: 'Trigger type',
    cron_subkind_label: 'Cadence',
    daily_at_label: 'At',
    on_error_label: 'On error',
    timeout_label: 'Timeout',
    timeout_hint: 'In minutes, 0 = no limit',
    add_workflow: 'Add workflow',
    target_n: 'Target workflow {n}',
    move_up: 'Move target up',
    move_down: 'Move target down',
    remove_target: 'Remove target',
    edit_action: 'Edit schedule “{name}”',
    delete_action: 'Delete schedule “{name}”',
    interval_label: 'Every N minutes',
    hotkey_label: 'Hotkey',
    limit_label: 'Limit',
    minutes: 'minutes',
    empty: 'No schedules yet',
    empty_desc:
      'Schedules bind cron / hotkey / once-at-startup, then run the listed workflows sequentially.',
    table: {
      caption: 'Schedule list',
      name: 'Name',
      trigger: 'Trigger',
      count: 'Workflows',
      last: 'Last triggered',
      enabled: 'Enabled',
      actions: 'Actions',
    },
    trigger: {
      manual: 'Manual only',
      cron: 'Schedule (Cron)',
      once: 'Once on startup',
      hotkey: 'Hotkey',
      daily: 'Daily',
      interval: 'Interval',
    },
    error_mode: {
      stop: 'Stop on any error',
      continue: 'Continue to next',
    },
    display: {
      daily: 'Daily {at}',
      interval: 'Every {mins}m',
      hotkey: 'Hotkey {key}',
    },
    workflow_unnamed: '(Untitled)',
    workflow_unbound: 'Unbound (manual only)',
    create_default_name: 'Schedule {n}',
    validation: {
      name: 'Enter a schedule name.',
      targets: 'Add at least one target workflow.',
      target: 'Select a target workflow.',
      daily: 'Enter a valid time in HH:MM format.',
      interval: 'The interval must be greater than 0 minutes.',
      hotkey: 'Enter a hotkey combination.',
    },
    delete_title: 'Delete schedule',
    delete_desc: 'Delete "{name}"? This cannot be undone.',
  },
  batchMetadata: {
    selection_hint: 'Selection persists across pages',
    description: 'Only fields you explicitly change are updated. Other metadata stays unchanged.',
    keep: 'Keep unchanged',
    set: 'Set',
    clear: 'Clear',
    add: 'Add',
    remove: 'Remove',
    replace: 'Replace',
    keep_hint: 'Keep the current value for every selected item',
    category_clear_hint: 'Remove the category from selected items',
    tags_clear_hint: 'Remove every tag from selected items',
    apply: 'Apply changes',
  },
  assets: {
    eyebrow: 'WORKFLOW ASSETS',
    title: 'Library and recording',
    description:
      'Manage input recordings and visual templates that bind directly to nodes. Content digests pin resources into workflows.',
    target_placeholder: 'Select an automation target',
    action_target_hint:
      'Choose the automation target for this action only; persistent target configuration does not occupy the library header.',
    asset_types: 'Asset types',
    library_actions: 'Library actions',
    view_list: 'List view',
    view_grid: 'Grid view',
    columns: {
      asset: 'Asset',
      organization: 'Category and tags',
      details: 'Asset details',
      created: 'Created',
    },
    tabs: {
      macros: 'Keyboard macros',
      clips: 'Precise recordings',
      templates: 'Visual templates',
    },
    search_placeholder: 'Search names, categories, or tags',
    search_all_placeholder: 'Search name, description, category, tags, or asset ID',
    search_action: 'Filter',
    all_categories: 'All categories',
    all_tags: 'All tags',
    reset_filters: 'Reset filters',
    category_filter: 'Filter category',
    tags_filter: 'Filter tags (comma separated)',
    sort_name_asc: 'Name A–Z',
    sort_name_desc: 'Name Z–A',
    sort_created_desc: 'Newest created',
    page_summary: 'Page {page} of {pages}; {total} assets',
    result_range: 'Showing {start}–{end} of {total} assets',
    per_page: 'Per page',
    unclassified: 'Unclassified',
    no_tags: 'No tags',
    select_page: 'Select every asset on this page',
    select_named: 'Select asset “{name}”',
    selected_count: '{n} assets selected',
    clear_selection: 'Clear selection',
    batch_edit: 'Batch category/tags',
    batch_edit_title: 'Edit category and tags for {n} assets',
    batch_delete: 'Batch delete',
    batch_delete_title: 'Delete {n} assets?',
    batch_update_result: 'Updated {updated}; failed {failed}.',
    batch_delete_result: 'Deleted {deleted}; failed {failed}.',
    cleanup_action: 'Clean unreferenced data',
    cleanup_title: 'Clean unreferenced Blobs?',
    cleanup_description:
      'This will reclaim {count} unreachable objects ({bytes} bytes). Confirmation rechecks every asset, Workflow Source, Program, and historical Run reference; changed references make the preview stale.',
    cleanup_none: 'There is no unreferenced data to clean.',
    cleanup_result: 'Reclaimed {count} unreferenced objects.',
    cleanup_failed: 'Asset cleanup failed. Preview again if durable references just changed.',
    no_results: 'No matching resources',
    no_results_hint: 'Try a different search term.',
    asset_actions: 'Manage “{name}”',
    edit_title: 'Edit resource details',
    tags_hint: 'Separate tags with commas',
    load_failed: 'Could not load the library',
    save_failed: 'Could not save resource details',
    delete_title: 'Delete “{name}”?',
    delete_description:
      'The library record will be deleted. Immutable content references already stored in workflows remain valid.',
    delete_failed: 'Could not delete the resource',
    preview_unavailable: 'Preview unavailable',
    recording: {
      title: 'Input recording',
      hint: 'Keyboard macros and precise trajectories are separate resources.',
      mode: 'Recording mode',
      active_hint: 'Capturing target {target}. Pause or finish here or in the floating controls.',
      waiting_hint:
        'The target is ready. Switch to it and press the start shortcut, or start from the floating controls.',
      countdown_hint: 'Input capture begins after the 3-second countdown.',
      start: 'Start recording',
      record_macro: 'Record keyboard macro',
      record_precise: 'Record precise trajectory',
      ready: 'Ready',
      finalizing: 'Finalizing',
      save_to_library: 'Save to library',
      start_failed: 'Could not start recording',
      control_failed: 'Recording control failed',
    },
    macros: {
      nav_hint: 'Editable down, up, click, and sleep actions',
      empty: 'No keyboard macros yet',
      empty_hint: 'Select a target and record an editable macro, or add actions manually later.',
      library_meta: 'Atomic macro · {bytes} bytes',
      base_resolution: 'Capture base resolution',
      edit_actions: 'Edit macro actions',
      load_failed: 'Could not load the keyboard macro',
      save_failed: 'Could not save the keyboard macro',
      create_blank: 'New blank macro',
      create_title: 'Create blank keyboard macro',
      create_hint:
        'Create a blank resource at the current target resolution, then add atomic actions in the macro editor.',
      resolution_unavailable:
        'The target resolution is unavailable. Open and bind the target window first.',
    },
    clips: {
      nav_hint: 'Full trajectories, dragging, and mouse turning',
      empty: 'No precise recordings yet',
      empty_hint: 'Select an automation target and record the complete input trajectory.',
      meta: '{duration} · {count} events · {mode}',
      library_meta: 'Input recording · {bytes} bytes',
      open_workbench: 'Open precise recording workbench',
      load_failed: 'Could not load the precise recording',
    },
    templates: {
      nav_hint: 'Match, wait, and click templates',
      empty: 'No visual templates yet',
      empty_hint:
        'Capture and crop a template from an installed target, then select it in matching nodes.',
      capture: 'Capture template',
      capture_failed: 'Could not create the visual template',
      meta: '{count} resolution variants',
      manage_variants: 'Manage resolution variants',
      recapture: 'Add or recapture this resolution',
      remove_variant: 'Delete this resolution variant',
      remove_variant_title: 'Delete the {resolution} variant?',
      remove_variant_description:
        'Only this variant is removed. Immutable Blob references already pinned in workflows are unaffected.',
      last_variant_hint:
        'The final variant cannot be removed alone. Delete the whole asset if it is no longer needed.',
    },
  },
  assetPicker: {
    template_title: 'Select visual template',
    macro_title: 'Select keyboard macro',
    clip_title: 'Select input recording',
    search_placeholder: 'Search name, description, category, tags, or ID',
    category_placeholder: 'Category',
    tags_placeholder: 'Tags (comma separated)',
    sort_recent: 'Recently used',
    filters: 'Filters',
    selection_instruction: 'Click to select; double-click to use',
    result_count: '{count} assets',
    current: 'Current binding',
    selected: 'Selected',
    selected_asset: 'Selected: {name}',
    select_hint: 'Select a resource or template variant first',
    select_clip: 'Select recording',
    use_template: 'Use this template',
    capture_template: 'Capture new template',
    use_macro: 'Use this macro',
    use_clip: 'Use this recording',
    replace: 'Replace resource',
    change: 'Change',
    open_library: 'Search and select from the asset library',
    clip_size: 'Input recording · {size} bytes',
    macro_size: 'Keyboard macro · {size} bytes',
    empty: 'No matching resources',
    empty_hint: 'Adjust the filters. Newly created resources appear immediately in this process.',
  },
  recordingHud: {
    title: 'Recording controls',
    subtitle: 'Control the active input capture',
    close_hint: 'Close controls; this cancels preparation but only hides an active recording',
    preparing: 'Preparing',
    preparing_hint: 'Connecting to the recording service',
    waiting: 'Waiting to start',
    waiting_hint: 'Switch to the target window, then press {key} to begin a 3-second countdown',
    start_countdown: 'Start countdown',
    cancel_countdown: 'Cancel countdown',
    countdown: 'Countdown',
    countdown_hint: 'Recording starts when the countdown ends',
    recording: 'Recording',
    paused: 'Paused',
    resuming: 'Resuming soon',
    resume_hint: 'Input capture resumes when the countdown ends',
    pause: 'Pause',
    resume: 'Resume',
    stop: 'Finish',
    stop_hint: 'Stop and review this recording ({key})',
    cancel: 'Cancel',
    cancel_confirm: 'Discard recording',
    shortcut_hint: '{start} start · {pause} pause/resume · {stop} finish',
  },
  recordingEditor: {
    title: 'Edit recorded actions',
    summary: '{count} actions; reorder them or adjust delay and duration',
    add_keys: 'Add keys',
    add_click: 'Add click',
    add_scroll: 'Add scroll',
    kind_keys: 'Keys',
    kind_click: 'Click',
    kind_scroll: 'Scroll',
    delay_ms: 'Delay before action (ms)',
    duration_ms: 'Duration (ms)',
    keys: 'Key chord',
    keys_placeholder: 'Ctrl + C',
    button: 'Mouse button',
    button_left: 'Left',
    button_middle: 'Middle',
    button_right: 'Right',
    notches: 'Scroll notches',
    point_x: 'X (percent)',
    point_y: 'Y (percent)',
    move_up: 'Move up',
    move_down: 'Move down',
    empty: 'The recording has no editable actions. Add one above.',
    precise_hint:
      'Precise recording preserves the full pointer path and original timing. The path is stored as a folded segment and replayed by InputClip.',
    editing_unavailable:
      'This recording exceeds the action editing budget. Its original events will be preserved when saved.',
    page: 'Page {page} of {pages}',
  },
  macroEditor: {
    title: 'Edit keyboard macro',
    summary: '{count} atomic actions · {duration}',
    search: 'Search actions or keys',
    add: 'Add action',
    action: 'Action',
    parameters: 'Parameters',
    state_after: 'Held after',
    select_visible: 'Select visible actions',
    select_action: 'Select action {n}',
    action_menu: 'Actions for row {n}',
    press_key: 'Focus, then press one key',
    press_key_hint: 'Each row records one down or up event',
    kind_key_down: 'Key down',
    kind_key_up: 'Key up',
    kind_mouse_down: 'Mouse down',
    kind_mouse_up: 'Mouse up',
    kind_click: 'Click',
    kind_scroll: 'Scroll',
    kind_sleep: 'Sleep',
    button_left: 'Left',
    button_middle: 'Middle',
    button_right: 'Right',
    duplicate: 'Duplicate action',
    state_none: 'None',
    empty: 'This macro has no actions',
    empty_hint: 'Start with Add action. Every down action must have a matching up action.',
    no_results: 'No matching actions',
    selected: '{count} actions selected',
    delete_selected: 'Delete selected',
    error_key_down: 'Row {n}: {key} is already held.',
    error_key_up: 'Row {n}: {key} is not held and cannot be released.',
    error_button_down: 'Row {n}: this mouse button is already held.',
    error_button_up: 'Row {n}: this mouse button is not held.',
    error_click_held: 'Row {n}: the {button} mouse button is already held and cannot be clicked.',
    error_held_end: 'A key or mouse button is still held when the macro ends.',
  },
  preciseWorkbench: {
    title: 'Trim precise recording',
    summary: '{duration} · {count} input events',
    recording_details: 'Recording details',
    timeline_title: 'Event timeline',
    timeline_hint:
      'This shows event distribution only and does not control the target. Click to move the nearest trim boundary.',
    timeline_aria: 'Click to choose the nearest trim boundary',
    play_preview: 'Preview timeline',
    pause_preview: 'Pause preview',
    raw_events: 'Raw events',
    raw_events_hint: 'Diagnostic data only; individual events are not editable here.',
    raw_previous: 'Previous raw event page',
    raw_next: 'Next raw event page',
    resolution: 'Capture resolution',
    mouse_mode: 'Mouse mode',
    counts_360: '360° counts',
    source_counts_360: 'Recorded counts/360',
    track_count: 'Active tracks',
    calibration_warning:
      'This recording contains relative mouse motion without valid counts360. Cross-device replay may be inaccurate.',
    track_keyboard: 'Keyboard',
    track_mouse_buttons: 'Mouse buttons',
    track_absolute_motion: 'Absolute motion',
    track_relative_motion: 'Relative camera',
    track_scroll: 'Scroll',
    no_tracks: 'No input tracks are available.',
    trim_start: 'Trim start',
    trim_end: 'Trim end',
    milliseconds: 'milliseconds',
    trim_hint:
      'Trim at any time. Held keys and mouse buttons are completed automatically at the boundaries to prevent stuck input.',
    event_none: 'Unknown',
    event_key_down: 'KeyDown',
    event_key_up: 'KeyUp',
    event_mouse_down: 'MouseDown',
    event_mouse_up: 'MouseUp',
    event_move: 'MouseMove',
    event_raw_delta: 'RawDelta',
    event_scroll: 'Scroll',
    event_unknown: 'Unknown',
  },
  recordingSave: {
    title: 'Save recording',
    pending: 'Not in library',
    pending_hint: 'Saving adds the resource to the library without changing the current workflow.',
    macro_type: 'Editable keyboard macro',
    clip_type: 'InputClip recording',
    mode_simple: 'Simple recording',
    mode_precise: 'Precise recording',
    summary: '{duration} · {count} input events',
    name: 'Recording name',
    name_hint: 'Use a name that explains what the recording is for.',
    name_placeholder: 'e.g. Daily reward setup',
    name_required: 'Enter a recording name',
    description_placeholder: 'Add usage notes or important details',
    category_placeholder: 'Select a category or type to create one',
    tags_placeholder: 'Select tags or type to create them',
    tags_hint: 'Multiple tags supported',
    save_add: 'Save and add to canvas',
    save_replace: 'Save and replace',
    save_failed: 'Could not save recording',
    discard: 'Discard recording',
    discard_confirm: 'Click again to discard',
    discard_confirm_hint: 'Click again to permanently discard this recording.',
    discard_failed: 'Could not discard recording',
  },
  calibration: {
    title: 'Mouse DPI calibration',
    switch_game_hint: 'Switch to game, press',
    start_label: 'Start',
    countdown_desc:
      '3-second countdown after press; final pose adjustment during countdown; auto-accumulate when done',
    ready_status: 'About to start accumulating, get ready',
    f8_shortcut: 'Press {hk} early = start now',
    recording_status: 'Accumulating · spin 360° in place',
    vertical_hint: '(vertical, for reference)',
    press_f8_stop: 'After turning, press {hk} to stop',
    recorded_label: 'Recorded',
    save_or_retest: 'Click "Save" below to write the local baseline; or press {hk} to retest',
    save_with_value: 'Save ({value})',
    service_failed: 'Calibration service failed to start (port in use?)',
    hud: {
      subtitle: 'Measure the mouse input baseline for this device',
      waiting: 'Waiting to start',
      countdown: 'Prepare to calibrate',
      press_to_start: 'Switch to your target game/app, press {hk} to start calibration',
      stage: {
        waiting: 'Standby',
        countingDown: 'Countdown',
        accumulating: 'Measuring',
        done: 'Measurement complete',
      },
    },
  },
  hotkeyInput: {
    click_to_set: 'Click to set hotkey',
    clear: 'Clear hotkey',
    instruction: 'Press combo · Backspace clears · Esc cancels',
    press_key: 'Press any key...',
    key_example: 'e.g. W / Space',
    click_to_record: 'Click to record key',
    click_to_cancel: 'Cancel recording',
  },
  settingsTab: {
    general: 'General',
    hotkeys: 'Hotkeys',
    input_calibration: 'Input & calibration',
    launcher: 'Floating launcher',
    ai: 'AI connections',
    network: 'Network capabilities',
    applications: 'Application capabilities',
    automation: 'Automation targets',
  },
  settingsCenter: {
    eyebrow: 'YOTTA SETTINGS',
    local_hint: 'Device and workspace preferences',
    search_placeholder: 'Search settings themes',
    clear_search: 'Clear settings search',
    themes_label: 'Settings themes',
    no_results: 'No matching settings themes',
    restart_required: 'Applies after restart',
    theme: {
      general: 'Interface language, startup behavior, and app-wide defaults',
      hotkeys: 'Review, search, and manage global and editor shortcuts',
      input: 'Configure recorded input semantics and maintain game calibration profiles',
      launcher: 'Arrange the floating launcher content, appearance, and quick actions',
      ai: 'Manage AI service endpoints, credentials, and connection health',
      network: 'Install exact HTTP origins and control workflow network consent',
      applications: 'Install exact desktop applications and control launch and terminate consent',
      automation:
        'Bind input nodes to exact installed application windows and control workflow consent',
    },
    save: {
      automatic: 'Saved automatically on this device',
      saving: 'Saving',
      saved: 'Saved on this device',
      failed: 'Save failed',
    },
  },
  iconPicker: {
    search_placeholder: 'Search icons...',
    no_match: 'No matching icons',
    search_hint: 'Searching all Tabler icons · showing up to 120',
  },
  hudShell: {
    close: 'Close',
  },
  mouseHud: {
    title: 'Mouse locator',
    subtitle: 'Live screen and target-window coordinates',
    target_ready: 'Target connected',
    screen_only: 'Screen coordinates only',
    screen: 'Screen',
    client: 'Window',
    ratio: 'Ratio',
    copy_ratio: 'Copy ratio',
    pick_color: 'Read color',
    copy_hex: 'Copy HEX value',
    outside_target: 'The pointer is outside the target window.',
    no_target: 'No target window detected. Only screen coordinates are available.',
  },
  screenPicker: {
    cancel_close: 'Cancel and close',
    fit: 'Fit to window',
    actual: 'Actual size',
    zoom_out: 'Zoom out',
    zoom_in: 'Zoom in',
    recapture: 'Capture again',
    gesture_hint: 'Wheel to zoom · hold Space to pan',
    capturing: 'Capturing screen…',
    extracting: 'Analyzing colors…',
    cursor: 'Cursor',
    cursor_hint: 'Live position and color',
    point: 'Point',
    point_hint: 'Click the canvas to set coordinates',
    color: 'Color',
    region: 'Region',
    color_hint: 'Click for one color or drag to analyze a region',
    region_hint: 'Drag on the canvas to select a rectangle',
    save_crop: 'Save cropped region',
    save_full: 'Save full capture',
    recapture_crop: 'Recapture cropped region',
    recapture_full: 'Recapture full image',
    mode: {
      point: 'Pick screen point',
      rect: 'Select screen region',
      template_save: 'Capture new template',
      template_recapture: 'Recapture template',
      color: 'Pick screen color',
    },
    hint: {
      point: 'Click the canvas to choose an exact position',
      rect: 'Drag on the canvas to choose a rectangular region',
      template_save: 'Optionally drag a crop; save the full capture when no region is selected',
      template_recapture:
        'Save a new resolution variant of the same asset; references follow automatically',
      color: 'Click for a single color or drag to analyze a target-color region',
    },
    status: {
      capturing: 'Capturing',
      saving: 'Saving',
      extracting: 'Analyzing',
      selected: 'Selected',
      ready: 'Waiting for selection',
    },
    template: {
      title: 'Template details',
      name: 'Name',
      name_placeholder: 'Enter a recognizable template name',
      category: 'Category',
      category_placeholder: 'Select or create a category',
      tags: 'Tags',
      tags_placeholder: 'Enter a tag and press Enter',
      remove_tag: 'Remove tag “{tag}”',
      crop_hint: 'Save the full capture, or drag on the canvas first to crop it.',
    },
  },
  floatingLauncher: {
    title: 'Launcher',
    subtitle: 'Run frequent automations quickly',
    item_count: '{n} entries',
    hide: 'Hide (hotkey can show it again)',
    pin: 'Pin window',
    unpin: 'Pinned · click to unpin',
    empty: 'Not configured yet — add items in Main app → Settings → Launcher',
    configure: 'Configure now',
    run: 'Run “{name}”',
    resize: 'Drag to resize',
    search_placeholder: 'Search automations…',
    search_aria: 'Search automations in the launcher',
    no_results: 'No matching automations',
    stale_count: '{n} to clean up',
    stale_hint: '{n} entries are stale. Clean them up in Settings.',
    stale_item: 'The linked workflow no longer exists',
    running: 'Starting',
    success: 'Run submitted',
    failed: 'Launch failed',
  },
  settingsLauncher: {
    title: 'Floating launcher',
    intro:
      'Put frequent workflows into a small launcher and run them with one click. Open it from the main window, this page, or the global hotkey.',
    access_title: 'Open & shortcut',
    access_hint:
      'Open the launcher now. The global hotkey is a shortcut, not the only entry point; repeated opens focus the existing window.',
    open_now: 'Open launcher',
    hotkey_title: 'Current global hotkey',
    hotkey_active:
      'The hotkey is registered and can show or hide the launcher from another foreground window.',
    hotkey_unbound:
      'No hotkey is bound. You can still open the launcher from the main window or this page.',
    hotkey_failed:
      'The hotkey failed to register or conflicts with another binding. Resolve it under Hotkeys.',
    configure_hotkey: 'Configure hotkey',
    display_label: 'Command layout',
    appearance_title: 'Appearance & display',
    display_hint:
      'Choose a command density. The list is informative; the compact grid suits icon-first entries.',
    display_both: 'Command list (recommended)',
    display_icon: 'Compact icon grid',
    display_text: 'Text-only list',
    health_title: 'Configuration health',
    health_hint:
      'Checks launcher entries against stored workflows. Cleanup removes stale references only; it never deletes workflows.',
    health_available: 'Available',
    health_stale: 'Stale entries',
    health_attention: 'Needs attention',
    health_normal: 'Healthy',
    health_ready: 'Launcher configuration is healthy',
    health_ready_hint:
      'Every entry resolves to a workflow and there are no stale references to clean up.',
    stale_title: '{n} stale workflow entries found',
    cleanup_scope: 'Stale entries that will be cleaned up',
    cleanup_stale: 'Clean {n}',
    undo_cleanup: 'Undo cleanup',
    layout_title: 'Command layout',
    layout_hint:
      'Organize commands with headings and separators. Vertical separators stay vertical in the compact grid and become spacing in lists. Drag to reorder.',
    empty: 'Empty launcher — add blocks below.',
    pick_icon: 'Pick icon',
    clear_icon: 'Clear icon',
    label_placeholder: 'Heading text, e.g. Combat',
    hsep: 'Horizontal separator',
    vsep: 'Vertical separator',
    delete_block: 'Delete this block',
    move_up: 'Move block up',
    move_down: 'Move block down',
    from_workflow: 'From workflow: {name}',
    add_workflow: '+ Workflow',
    label_block: 'Text heading',
    deleted_workflow: '(deleted workflow)',
    library_title: 'Add content block',
    preview_title: 'Live preview',
    live_badge: 'Live',
    preview_empty: 'Add content to preview it here',
    untitled_label: 'Untitled heading',
  },
  settingsNetwork: {
    security: {
      title: 'Workflows never receive arbitrary network access',
      hint: 'Each installation pins one exact scheme, host, and port. Nodes can supply only a relative path and query; redirects, proxies, cookies, credentials, and request headers are disabled.',
    },
    origins: {
      title: 'Installed HTTP origins',
      hint: 'Use a stable slot to bind workflows to an exact origin and bounded response budget.',
      add: 'Install origin',
      unnamed: 'Unnamed origin',
      workflow_allowed: 'Workflow allowed',
      consent_required: 'Consent required',
      private_enabled: 'Private network enabled',
      origin_missing: 'Origin not specified',
      name_label: 'Display name',
      name_placeholder: 'For example: Production status API',
      slot_label: 'Installation slot',
      slot_hint: 'Workflows persist this identifier. It cannot be changed after saving.',
      origin_label: 'Exact origin',
      origin_hint:
        'Scheme, host, and optional port only. Public origins require HTTPS. Paths, query strings, fragments, and user information are rejected.',
      byte_limit_label: 'Maximum response bytes',
      byte_limit_hint: 'Responses larger than this limit fail before entering workflow data.',
      timeout_label: 'Timeout (milliseconds)',
      timeout_hint: 'Hard limit covering connection, response headers, and body reading.',
      delete: 'Delete origin',
      empty: 'No HTTP origins installed',
      empty_hint:
        'Install an exact origin before an HTTP GET node can pass admission. Installation does not grant workflow use automatically.',
      new_label: 'New HTTP origin',
    },
    private: {
      title: 'Allow private and loopback destinations',
      hint: 'Keep disabled for public APIs. DNS results are checked again at connection time to prevent rebinding into local networks.',
      warning:
        'This origin may reach services on this computer or private network. Enable only for a target you control and trust.',
    },
    consent: {
      title: 'Workflow network consent',
      hint: 'Consent matches this exact slot and profile digest. Editing the origin, private-network policy, timeout, or byte limit revokes it. Restart to install the new snapshot.',
      grant: 'Allow current origin',
      revoke: 'Revoke consent',
    },
    confirm: {
      delete_title: 'Delete “{name}”?',
      delete_hint:
        'The installation slot will be removed. Workflows that reference it will fail admission.',
    },
  },
  settingsApplications: {
    security: {
      title: 'Desktop applications inherit Yotta administrator privileges',
      hint: 'This capability is not a process sandbox. Install only a GUI application you explicitly selected and trust. Workflows reference only its slot and cannot supply an executable, command line, environment, working directory, or PID.',
    },
    profiles: {
      title: 'Installed desktop applications',
      hint: 'Each profile fixes one .exe installation path and argument list. Normal updates at that path retain authorization. Launch never uses a shell.',
      add: 'Select and install application',
      workflow_allowed: 'Workflow allowed',
      consent_required: 'Consent required',
      name_label: 'Display name',
      slot_label: 'Installation slot',
      slot_hint: 'Workflows persist this identifier. It cannot be changed after saving.',
      executable_label: 'Exact executable',
      executable_hint: 'Regular .exe files only. Shell, PowerShell, and script hosts are rejected.',
      replace: 'Choose another',
      arguments_label: 'Fixed arguments',
      arguments_hint:
        'One argument per line, sealed as a distinct argv element. Workflows cannot modify them at run time.',
      arguments_placeholder: '--project\nD:\\Projects\\fixed.aep',
      delete: 'Delete application',
      empty: 'No desktop applications installed',
      empty_hint:
        'Select a trusted .exe first. Installation does not automatically grant workflow launch or terminate authority.',
      cancelled: 'No desktop application selected',
      cancelled_hint:
        'The file picker was cancelled. No installation was created or changed; you can choose again when ready.',
    },
    consent: {
      title: 'Workflow application lifecycle consent',
      hint: 'Consent matches this slot, installation path, and fixed arguments. Changing the path or arguments revokes it; a normal update at the same path does not.',
      grant: 'Allow launch and terminate',
      revoke: 'Revoke consent',
    },
    picker: {
      title: 'Choose a Windows application to install',
      inspect_failed: 'Could not read the executable identity of the selected application.',
    },
    confirm: {
      delete_title: 'Delete “{name}”?',
      delete_hint: 'The slot will be removed. Workflows that reference it will fail admission.',
    },
  },
  settingsAutomation: {
    security: {
      title: 'Automation controls real desktops, devices, or browser pages',
      hint: 'Every target is pinned to an adapter-verified exact identity. Workflows reference only a slot and cannot supply native handles, paths, device serials, debugger addresses, or backends. Identity drift, offline targets, and operation failures fail the node.',
    },
    targets: {
      title: 'Installed automation targets',
      hint: 'Install exact Windows window, Android ADB device, or Browser CDP page identities behind stable workflow slots. Each adapter exposes only the operations it supports.',
      add: 'Install window target',
      add_windows: 'Windows target',
      add_android: 'Android target',
      add_browser: 'Browser target',
      workflow_allowed: 'Workflow allowed',
      consent_required: 'Consent required',
      name_label: 'Display name',
      slot_label: 'Installation slot',
      slot_hint: 'Workflows persist this identifier. It cannot be changed after saving.',
      application_label: 'Installed application',
      application_hint:
        'Reuses the user-authorized installation path from Applications. Launch arguments are not inherited.',
      backend_label: 'Fixed input backend',
      backend_hint:
        'SendInput foregrounds the target. PostMessage posts only to the exact window queue. Runtime never switches automatically.',
      capture_backend_label: 'Fixed capture backend',
      capture_backend_hint:
        'GDI and Windows Graphics Capture are explicit contracts. An unavailable backend fails installation; runtime never falls back.',
      mouse_counts_label: 'Mouse counts per 360°',
      mouse_counts_hint:
        'Precise recording follows the active Input & calibration profile. A target override takes priority and supplies relative-mouse playback scaling.',
      mouse_counts_use_active: 'Follow active mouse calibration',
      mouse_counts_custom: 'Use a custom value for this target',
      mouse_counts_following:
        'Following “{name}”: {n} counts/360°. Switching or recalibrating the active profile applies to the next recording.',
      mouse_counts_missing:
        'There is no valid active mouse calibration. Precise relative recording will remain unavailable.',
      open_calibration: 'Manage calibration profiles',
      window_title_match_label: 'Window title match mode',
      window_title_match_hint:
        'Captured windows use exact matching and preserve the title verbatim, including surrounding spaces. Explicitly select regex for dynamic titles.',
      window_title_match_exact: 'Exact match',
      window_title_match_regex: 'Regex match',
      window_selection_label: 'When multiple windows match',
      window_selection_hint:
        'Require a unique fixed window for safety, or explicitly follow the current topmost match for dynamic multi-window apps.',
      window_selection_unique: 'Require a unique match',
      window_selection_topmost: 'Use the current topmost match',
      preview_matches: 'Check current window matches',
      window_title_label: 'Window title',
      window_title_hint:
        'Exact mode is case-sensitive and character-for-character. Regex mode uses RE2 syntax.',
      window_class_label: 'Exact window class',
      window_class_hint:
        'Full Win32 class-name match. When title and class are set, both must match.',
      timeout_label: 'Resolve timeout (milliseconds)',
      timeout_hint: 'Hard limit for waiting on one unique matching window, from 100 to 10000.',
      delete: 'Delete target',
      empty: 'No automation targets installed',
      empty_hint:
        'Install a Windows, Android, or browser target before automation nodes can pass admission. Installation does not grant workflow use automatically.',
      no_applications: 'Install a desktop application first',
      no_applications_hint:
        'A window target must reference a user-installed .exe from the Applications page.',
      new_label: '{name} window',
      new_blank_label: 'New window target',
      duplicate: 'Duplicate target',
      copy_label: '{name} copy',
    },
    android: {
      new_blank_label: 'New Android target',
      discovery_hint:
        'Yotta discovers devices through the configured or bundled ADB. Saving pins serial, product, model, and device identity; reconnecting a different device under the same serial fails closed.',
      refresh: 'Refresh devices',
      none_found:
        'No ADB devices found. Start an emulator or connect a device and authorize USB debugging.',
      device_label: 'ADB device',
      device_hint:
        'Only devices in the ready “device” state with a complete identity can be installed.',
      package_label: 'Android package',
      package_hint:
        'Search launchable apps from the selected device, or enter an exact package manually.',
      refresh_apps: 'Refresh applications',
      apps_none_found:
        'No third-party launchable application was found. You can still enter an exact package manually.',
      app_search: 'Search application name or package',
      app_unselected: 'Select an application',
      manual_package: 'Exact package, for example com.example.app',
      foreground_app: 'Foreground: {name} · {package}',
      identity_label: 'Pinned device identity',
      state_label: 'Runtime health',
      not_checked: 'Not checked',
      check_health: 'Check',
      unselected: 'No device selected',
    },
    browser: {
      new_blank_label: 'New browser target',
      discovery_hint:
        'Start Chrome or Edge with an explicit remote-debugging-port, then select one page from a loopback CDP endpoint. Yotta pins the endpoint, page ID, and WebSocket identity. A browser restart or identity change requires reinstalling the page.',
      refresh: 'Discover pages',
      none_found:
        'No installable pages found. Confirm the browser was started with an explicit debugging port and the endpoint uses 127.0.0.1 or ::1.',
      endpoint_label: 'Loopback CDP endpoint',
      endpoint_hint:
        'Only an HTTP origin with a literal loopback IP is accepted, for example http://127.0.0.1:9222.',
      page_label: 'Exact page',
      page_hint:
        'The page ID and WebSocket URL enter the installed identity. Titles and URLs are never fuzzy matched.',
      url_label: 'URL at discovery',
      state_label: 'Runtime health',
      not_checked: 'Not checked',
      check_health: 'Check',
      unselected: 'No page selected',
    },
    capture: {
      hint: 'Temporarily enable the global capture key, switch to the target window, then press F9 (or the configured key). A successful capture binds and immediately authorizes that exact identity; the key is then released without reaching the target app.',
      start: 'Capture and enable window',
      start_failed: 'Could not start window capture. Try again.',
      cancel: 'Cancel capture',
      cancelled: 'Window capture cancelled. The target was not changed.',
      timeout: 'Window capture timed out. Try again.',
      incomplete: 'The capture result is missing the executable, window title, or window class.',
      inspect_failed: 'Could not verify the captured window executable identity.',
      application_missing:
        'The captured window application is not installed. Select and install its .exe under Desktop applications first.',
      install_title: 'Install “{name}”, bind this window, and allow automation?',
      install_hint:
        'Yotta will install and authorize this executable path and create the window target. Normal updates at the same path will not require reauthorization: {path}',
      install_confirm: 'Install, bind, and allow',
      install_cancelled: 'Installation cancelled. The captured identity was not saved.',
      installed_and_completed:
        'Installed the window application, bound “{name}” to the exact window, and enabled it immediately.',
      application_ambiguous:
        'Multiple installation records match this executable. Remove the duplicate record and try again.',
      save_failed: 'The capture result could not be saved as a complete window target.',
      completed: 'Bound “{name}” to the exact window and enabled it immediately.',
    },
    backend: {
      sendinput: 'SendInput · foreground system input',
      postmessage: 'PostMessage · exact window messages',
    },
    captureBackend: {
      gdi: 'GDI · exact window pixels',
      wgc: 'WGC · Windows Graphics Capture',
    },
    consent: {
      title: 'Workflow automation consent',
      hint: 'Consent covers only the operations declared by this adapter and exactly matches the slot and complete target identity. Any edit revokes it; reauthorization applies in the current process.',
      grant: 'Allow automation',
      revoke: 'Revoke consent',
    },
    bulk: {
      grant: 'Authorize all',
      revoke: 'Revoke all',
      grant_title: 'Authorize all current desktop automation installations?',
      grant_hint:
        'This authorizes every currently installed desktop application and automation target in one action. New installations or identity changes still require fresh consent and do not bypass capability, admission, or arm boundaries.',
      revoke_title: 'Revoke all desktop automation consent?',
      revoke_hint:
        'Workflow consent will be cleared from every installed desktop application and automation target.',
      granted:
        'Authorized all current desktop applications and automation targets. New or changed items still require consent.',
      revoked: 'Revoked all desktop application and automation target consent.',
    },
    confirm: {
      delete_title: 'Delete “{name}”?',
      delete_hint:
        'The target slot will be removed. Workflows that reference it will fail admission.',
    },
  },
  settingsAI: {
    title: 'AI Models',
    intro: 'Install explicit model profiles for workflows to call through stable slots.',
    security: {
      title: 'Credentials and model configuration are stored separately',
      hint: 'API keys are written only to the operating system credential store, never to settings.json, and are never displayed again. Model profiles use provider-native protocols.',
    },
    provider: {
      openai_responses: 'OpenAI Responses',
      anthropic_messages: 'Anthropic Messages',
    },
    profiles: {
      title: 'Installed models',
      hint: 'Each profile binds an immutable slot, a provider-native protocol, an exact model name, and declared capabilities. Workflows reference only the slot, never an endpoint or key.',
      add: 'Install model',
      unnamed: 'Unnamed model',
      credential_saved: 'Key saved',
      workflow_allowed: 'Workflow allowed',
      consent_required: 'Consent required',
      model_missing: 'Model not specified',
      name_label: 'Display name',
      slot_label: 'Installation slot',
      slot_hint: 'Workflows persist this identifier. It cannot be changed after saving.',
      provider_label: 'Native provider protocol',
      model_label: 'Exact model name',
      model_hint: 'No discovery or alias substitution. This exact name is sent to the provider.',
      endpoint_label: 'Exact API endpoint',
      endpoint_hint:
        'The complete provider-native request URL, including the Responses or Messages path. Yotta never probes or falls back to another protocol.',
      endpoint_reset: 'Restore official endpoint',
      local_http_title: 'Allow a local HTTP endpoint',
      local_http_hint:
        'Only localhost, 127.0.0.0/8, or ::1 is allowed. Remote endpoints still require HTTPS.',
      local_http_warning:
        'HTTP is unencrypted. Enable it only for a local test service you control.',
      label_placeholder: 'For example: Primary reasoning model',
      model_placeholder: 'For example: gpt-5.1 or claude-opus-4-1',
      max_tokens_label: 'Maximum output tokens',
      max_tokens_hint: 'The output limit allowed for one generation with this profile.',
      apikey_label: 'API key',
      apikey_hint: 'Credentials are isolated by installation slot.',
      apikey_replace_placeholder: 'Key saved, enter a new key to replace it',
      apikey_placeholder: 'Enter the provider API key',
      reveal: 'Show or hide key',
      apikey_secure_hint:
        'After saving, the UI reads only the presence status and never reads the key back.',
      apikey_remove: 'Remove key',
      test: 'Test',
      delete: 'Delete model',
      test_ok: 'Native request succeeded. Resolved model: {model}; finish: {finish}',
      empty: 'No AI models installed',
      empty_hint:
        'Install a model before nodes can generate through a stable slot. Installation does not grant workflow access automatically.',
      new_label: 'New model',
    },
    evaluation: {
      unverified: 'Not evaluated',
      approved: 'Evaluation approved',
      rejected: 'Evaluation rejected',
    },
    capabilities: {
      title: 'Declared capabilities',
      hint: 'Declare only verified capabilities. Declarations enter the installation digest and admission checks; failures never trigger an automatic downgrade.',
      structured_output: 'Structured output',
      structured_output_hint: 'Allow nodes to request provider-native structured results.',
      tool_calling: 'Tool calling',
      tool_calling_hint: 'Allow the model to initiate tool calls.',
      parallel_tools: 'Parallel tool calls',
      parallel_tools_hint:
        'Allow one response to contain parallel tool calls. Tool calling is also required.',
      background: 'Background generation',
      background_hint: 'Allow provider-native asynchronous or background execution.',
      zero_retention: 'Zero retention',
      zero_retention_hint:
        'Declare only when the provider and exact model configuration satisfy zero-retention requirements.',
    },
    pricing: {
      title: 'Pinned token pricing',
      hint: 'Required for tool calling so the host can enforce cost budgets. Enter billing microunits per one million tokens; these values enter the profile digest.',
      input: 'Input',
      cache_read: 'Cache read',
      output: 'Output',
    },
    consent: {
      title: 'Workflow usage consent',
      hint: 'Consent matches only the current profile contents. Changing the endpoint, model, capabilities, or limit invalidates it immediately. Restart to install the new consent digest into the runtime.',
      grant: 'Allow current profile',
      revoke: 'Revoke consent',
    },
    confirm: {
      delete_title: 'Delete “{name}”?',
      delete_profile:
        'The installation slot and its saved key will be removed. Workflows that reference the slot will fail admission.',
      delete_key_title: 'Remove the saved API key?',
      delete_key_hint:
        'The model profile remains, but every generation request through this slot will fail until a new key is saved.',
    },
  },
  about: {
    tagline: 'Compose nodes. Run automatically.',
    concepts: {
      title: 'Core concepts',
      workflow: {
        name: 'Workflow',
        desc: 'The only editable orchestration document. A Source stores typed graphs, configuration, state declarations, and exact NodeRefs, then commits a server-normalized revision.',
      },
      catalog: {
        name: 'Node Catalog',
        desc: 'An immutable snapshot of Node Contracts, data types, capability requirements, and implementation locks. UI, AI, CLI, and docs share its Authoring Projection.',
      },
      program_run: {
        name: 'Program and Run',
        desc: 'Compilation turns a Source into a content-addressed immutable Program. Every execution is independently admitted, cancellable, and recorded as an auditable Run.',
      },
      installation: {
        name: 'Installation and Target',
        desc: 'Models, applications, automation targets, and plugins install under stable slots. Workflows declare slots and capabilities without persisting native handles or bypassing consent.',
      },
    },
    section_author: 'Author · Links',
    label_author: 'Author',
    label_source: 'Source',
    label_site: 'Site',
    section_stack: 'Tech stack',
    section_thanks: 'Acknowledgements',
    label_icon: 'App icon',
    icon_credit:
      'Icon sourced from public works on Pixiv; copyright belongs to the original author. For personal use only; please contact for replacement if requested.',
  },
}
