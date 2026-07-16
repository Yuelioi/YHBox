export default {
  sidebar: {
    workflows: 'Workflows',
    workflow_edit: 'Edit workflow',
    schedules: 'Schedules',
    settings: 'Settings',
    about: 'About',
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
        'Start Yotta after Windows login. Writes registry HKCU\\...\\Run\\Yotta, no admin needed.',
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
      inputClip: {
        title: 'Input clip',
        description: 'A content-addressed recording carried only as a durable nominal BlobRef.',
      },
      pointer_button: {
        title: 'Pointer button',
        description: 'The left, right, or middle pointer button.',
      },
      key_code: {
        title: 'Key code',
        description: 'A canonical keyboard key injectable into an exact installed target.',
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
      },
      write: {
        title: 'Write state',
        description:
          'Write one exactly typed value to a Run-local state slot, then continue through Done.',
      },
      metadata: {
        title: 'State metadata',
        description:
          'Read a state slot revision and last-change time without exposing an untyped value.',
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
          'Launch the exact executable and fixed arguments sealed in Settings. Workflows cannot supply a path, arguments, or command line.',
      },
      terminate: {
        title: 'Terminate installed application',
        description:
          'Terminate only processes whose executable identity exactly matches the installed profile, and return the count.',
      },
      config: {
        slot: {
          title: 'Application slot',
          description:
            'Select an installed, digest-verified, and explicitly consented desktop application.',
        },
      },
    },
    automation: {
      config: {
        slot: {
          title: 'Window target slot',
          description:
            'Select an explicitly consented target pinned to an executable digest, window title, and window class.',
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
      typeText: {
        title: 'Type text',
        description:
          'Inject bounded Unicode text into the exact installed window without using the clipboard.',
      },
      activateWindow: {
        title: 'Activate window',
        description:
          'Reverify the installed application and unique window identity, then bring it to the foreground. Failure routes through Failed.',
      },
      captureWindow: {
        title: 'Capture window',
        description:
          'Reverify the exact installed window, capture it as PNG through the configured backend, and commit a durable Image BlobRef.',
      },
      playInputClip: {
        title: 'Play input clip',
        description:
          'Read a validated InputClip BlobRef and replay every event through one exclusive exact-target playback session.',
      },
    },
    observability: {
      log: {
        title: 'Write log',
        description:
          'Write one bounded, Run-attributed message and record only its digest in the action journal.',
        config: {
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
    cancel: 'Cancel',
    confirm: 'Confirm',
    copied: 'Copied',
    save: 'Save',
    delete: 'Delete',
    edit: 'Edit',
    close: 'Close',
    back: 'Back',
    copy: 'Copy',
    add: 'Add',
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
        'Strong-stop / calibrate / recording stop / recording pause will return to factory defaults.',
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
    RECORDING_TARGET_UNAVAILABLE: 'The recording target is unavailable',
    UNKNOWN_ERROR: 'An unknown error occurred',
    TRANSPORT_TIMEOUT: 'The request timed out; try again',
    TRANSPORT_UNAVAILABLE: 'The backend connection is unavailable; restart Yotta',
    admission: {
      target_unavailable: 'A required target is unavailable',
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
      identity_changed: 'The installed application identity changed',
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
    list: {
      title: 'Workflows',
      description: 'Every run compiles a saved 3.1 source into an immutable Program snapshot.',
      new_workflow: 'New workflow',
      name_placeholder: 'Workflow name',
      create: 'Create',
      loading: 'Loading workflows',
      empty_title: 'Create the first workflow',
      empty_description:
        'The host creates a strict source envelope and opens it in the generated node editor.',
      name: 'Name',
      revision: 'Revision',
      source_identity: 'Source identity',
      actions: 'Actions',
    },
    editor: {
      loading: 'Loading workflow editor',
      open_failed: 'Workflow could not be opened',
      back: 'Back to workflows',
      workflow_name: 'Workflow name',
      revision: 'rev {n}',
      unsaved: 'Unsaved',
      save_conflict: 'Save conflict: {message}',
      node_catalog: 'Node catalog',
      catalog_description: 'Click a node, or drag it onto the canvas.',
      discard_confirm: 'Discard unsaved workflow changes?',
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
      permission_confirm:
        'This candidate adds {n} permission requirement. Accept the exact candidate?',
      accepted_toast: 'AI proposal accepted',
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
      debug: 'Debug',
      compile: 'Compile',
      save: 'Save',
      stop: 'Stop',
      stop_all: 'Stop all',
      refresh: 'Refresh',
      undo: 'Undo',
      redo: 'Redo',
    },
    node: {
      disabled: 'Disabled',
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
      reference_only: 'This port accepts {carrier} references through a compatible connection.',
      select_clip: 'Select an input clip',
      select_template: 'Select an exact template variant',
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
      use_default: 'Use default',
      clear: 'Clear',
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
    },
    timeline: {
      run_status: 'Run {status}',
      empty: 'No attempts have been recorded yet.',
      attempt: 'attempt {n}',
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
      compile_diagnostics: 'Compile produced diagnostics',
      compile_succeeded: 'Compile succeeded',
      compile_failed: 'Compile failed',
      saved: 'Workflow saved',
      save_failed: 'Save failed',
      debug_started: 'Debug timeline started',
      debug_failed: 'Debug failed',
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
    delete_title: 'Delete schedule',
    delete_desc: 'Delete "{name}"? This cannot be undone.',
  },
  recordingHud: {
    title: 'Recording controls',
    subtitle: 'Control the active input capture',
    close_hint: 'Close and handle the active recording',
    preparing: 'Preparing',
    preparing_hint: 'Connecting to the recording service',
    countdown: 'Countdown',
    countdown_hint: 'Recording starts when the countdown ends',
    recording: 'Recording',
    paused: 'Paused',
    resuming: 'Resuming soon',
    resume_hint: 'Input capture resumes when the countdown ends',
    pause: 'Pause',
    resume: 'Resume',
    stop: 'Finish',
    stop_hint: 'Stop and save this recording ({key})',
    cancel: 'Cancel',
    cancel_confirm: 'Discard recording',
    shortcut_hint: '{stop} finish · {pause} pause/resume',
  },
  recordingSave: {
    title: 'Save recording',
    pending: 'Not in library',
    pending_hint: 'The recording joins your library and canvas only after you save it.',
    clip_type: 'InputClip recording',
    summary: '{duration} · {count} input events',
    name: 'Recording name',
    name_hint: 'Use a name that explains what the recording is for.',
    name_placeholder: 'e.g. Daily reward setup',
    name_required: 'Enter a recording name',
    description_placeholder: 'Add usage notes or important details',
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
      'Put frequently used workflows into a small launcher window and run them with one click. Open it with the show/hide hotkey.',
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
      title: 'Desktop applications run with your current user authority',
      hint: 'This capability is not a process sandbox. Install only a GUI application you explicitly selected, trust, and verified by digest. Workflows reference only its slot and cannot supply an executable, command line, environment, working directory, or PID.',
    },
    profiles: {
      title: 'Installed desktop applications',
      hint: 'Each profile pins one .exe content digest and fixed argument list. Launch never uses a shell; terminate matches the same file identity only.',
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
        'Select a trusted .exe and verify its digest first. Installation does not automatically grant workflow launch or terminate authority.',
    },
    consent: {
      title: 'Workflow application lifecycle consent',
      hint: 'Consent matches this exact slot, executable digest, and fixed arguments. Any edit revokes it immediately. Restart to install the new snapshot.',
      grant: 'Allow launch and terminate',
      revoke: 'Revoke consent',
    },
    picker: { title: 'Choose a Windows application to install' },
    confirm: {
      delete_title: 'Delete “{name}”?',
      delete_hint: 'The slot will be removed. Workflows that reference it will fail admission.',
    },
  },
  settingsAutomation: {
    security: {
      title: 'Window automation controls real windows, keyboard, and pointer',
      hint: 'Every target is pinned to an installed executable digest, exact window selector, input backend, and capture backend. Workflows reference only a slot and cannot supply an HWND, PID, process name, path, or backend. Activation or capture failure fails the node, and all held input is released on cancellation or failure.',
    },
    targets: {
      title: 'Installed window targets',
      hint: 'Provide one fixed target for activation, capture, click, move, scroll, drag, key chord, and text nodes. Multiple matching windows fail instead of selecting by Z order.',
      add: 'Install window target',
      workflow_allowed: 'Workflow allowed',
      consent_required: 'Consent required',
      name_label: 'Display name',
      slot_label: 'Installation slot',
      slot_hint: 'Workflows persist this identifier. It cannot be changed after saving.',
      application_label: 'Installed application',
      application_hint:
        'Reuses the SHA-256 executable identity from Applications. Launch arguments are not inherited.',
      backend_label: 'Fixed input backend',
      backend_hint:
        'SendInput foregrounds the target. PostMessage posts only to the exact window queue. Runtime never switches automatically.',
      capture_backend_label: 'Fixed capture backend',
      capture_backend_hint:
        'GDI and Windows Graphics Capture are explicit contracts. An unavailable backend fails installation; runtime never falls back.',
      mouse_counts_label: 'Mouse counts per 360°',
      mouse_counts_hint:
        'Hardware calibration used for exact relative-motion replay. Set 0 only when this target will never replay relative mouse events.',
      window_title_label: 'Exact window title (optional)',
      window_title_hint:
        'Case-sensitive full match. Contains, regex, and wildcard matching are not supported.',
      window_class_label: 'Exact window class (optional)',
      window_class_hint:
        'Full Win32 class-name match. When title and class are set, both must match.',
      timeout_label: 'Resolve timeout (milliseconds)',
      timeout_hint: 'Hard limit for waiting on one unique matching window, from 100 to 10000.',
      delete: 'Delete target',
      empty: 'No window targets installed',
      empty_hint:
        'Install a target before 3.1 window automation nodes can pass admission. Installation does not grant workflow use automatically.',
      no_applications: 'Install a desktop application first',
      no_applications_hint:
        'A window target must reference a content-verified .exe from the Applications page.',
      new_label: '{name} window',
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
      title: 'Workflow window automation consent',
      hint: 'Consent covers activation, capture, and atomic input operations for this target and matches its slot, executable digest, window selector, both backends, and timeout. Any edit revokes it. Restart to install the new snapshot.',
      grant: 'Allow window automation',
      revoke: 'Revoke consent',
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
      hint: 'Consent matches only the current profile contents. Changing the model, capabilities, or limit invalidates it immediately. Restart to install the new consent digest into the runtime.',
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
