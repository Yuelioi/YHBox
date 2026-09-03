export default {
  node: {
    ai: {
      generate: {
        title: 'Generate AI text',
        description: 'Generates text from a prompt and optional image, including captured windows.',
      },
      extract: {
        title: 'Extract structured AI data',
        description:
          'Extracts the fields you choose from text or an image and returns connectable structured data.',
        config: {
          fields: {
            title: 'Output fields',
            description:
              'Add the values to extract; Yotta generates and validates the output structure.',
            field: 'Field {n}',
            name: 'Field name',
            type: 'Type',
            field_description: 'Field description (optional)',
            nullable: 'Allow an empty value when the content is missing',
            empty: 'Add at least one output field.',
            add: 'Add field',
            remove: 'Remove field {n}',
            types: {
              string: 'Text',
              number: 'Number',
              integer: 'Integer',
              boolean: 'Yes / no',
            },
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
        timeoutMilliseconds: {
          title: 'Timeout (milliseconds)',
          description:
            'Hard wall-time limit for this AI node attempt, from 1000 to 120000 milliseconds.',
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
        description:
          'Open durable binary content as a leased stream that is valid only for the current run.',
      },
      streamToBlob: {
        title: 'Stream to blob',
        description:
          'Consume a leased stream from the current run and commit it as durable binary content.',
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
        input: {
          input: {
            title: 'Input',
            description: 'Canonical JSON exposed to the guest as the isolated input value.',
          },
        },
        output: {
          result: {
            title: 'Result',
            description: 'Canonical JSON-compatible value returned by the guest.',
          },
        },
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
        input: {
          path: {
            title: 'Relative path',
            description: 'File path relative to the managed workflow workspace.',
          },
        },
        output: {
          text: { title: 'Text', description: 'Decoded file text.' },
          metadata: { title: 'File metadata', description: 'Canonical metadata for the file.' },
        },
      },
      readJSON: {
        title: 'Read workspace JSON',
        description:
          'Read and strictly parse one UTF-8 JSON document from the workflow file workspace.',
        input: {
          path: {
            title: 'Relative path',
            description: 'JSON file path relative to the managed workflow workspace.',
          },
        },
        output: {
          value: { title: 'JSON value', description: 'Canonical parsed JSON value.' },
          text: { title: 'Source text', description: 'Original decoded UTF-8 document text.' },
          metadata: { title: 'File metadata', description: 'Canonical metadata for the file.' },
        },
      },
      stat: {
        title: 'Inspect workspace file',
        description: 'Read canonical metadata without exposing an ambient host filesystem path.',
        input: {
          path: {
            title: 'Relative path',
            description: 'File or directory path relative to the managed workflow workspace.',
          },
        },
        output: {
          metadata: { title: 'File metadata', description: 'Canonical metadata for the path.' },
        },
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
        description: 'Read UTF-8 text from a configured HTTP base URL using a relative path.',
        input: {
          path: {
            title: 'Relative path',
            description: 'Path resolved against the configured HTTP base URL.',
          },
          query: {
            title: 'Query',
            description: 'JSON object whose values are arrays of query-string values.',
          },
        },
        output: {
          status: { title: 'Status', description: 'HTTP response status code.' },
          body: { title: 'Body', description: 'Bounded UTF-8 response body.' },
          'content-type': {
            title: 'Content type',
            description: 'Response Content-Type header, or an empty string when absent.',
          },
        },
        config: {
          slot: {
            title: 'HTTP target slot',
            description: 'Select the configured HTTP target for this request.',
          },
        },
      },
    },
    application: {
      launch: {
        title: 'Launch application',
        description: 'Launch the program path and arguments configured in the application slot.',
      },
      terminate: {
        title: 'Terminate application',
        description: 'Terminate processes that belong to the configured path and return the count.',
        output: {
          'terminated-count': {
            title: 'Terminated count',
            description: 'Number of matching processes terminated by this invocation.',
          },
        },
      },
      config: {
        slot: {
          title: 'Application slot',
          description: 'Select a configured desktop application.',
        },
      },
    },
    automation: {
      config: {
        slot: {
          title: 'Window target slot',
          description:
            'Select a target bound to an application path, window title, and window class.',
        },
      },
      clickPointer: {
        title: 'Click pointer',
        description: 'Perform one click inside the configured target.',
      },
      movePointer: {
        title: 'Move pointer',
        description:
          'Move to a target coordinate instantly, along a constant-speed line, or on a Bézier curve.',
      },
      getPointerPosition: {
        title: 'Get pointer position',
        description:
          'Read the current pointer position in a desktop target client area as a normalized point that can feed a move node.',
      },
      scrollPointer: {
        title: 'Scroll pointer',
        description: 'Scroll inside the configured target.',
      },
      dragPointer: {
        title: 'Drag pointer',
        description:
          'Press, move with the selected motion, and release inside the configured target.',
      },
      movePointerRelative: {
        title: 'Move pointer relatively',
        description: 'Send a relative pointer movement to the configured target.',
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
        description: 'Enter Unicode text into the configured target.',
      },
      activateWindow: {
        title: 'Activate target',
        description:
          'Resolve the configured target, then foreground its window or start its Android package.',
      },
      closeWindow: {
        title: 'Close window',
        description: 'Resolve the configured target and send a close request to its window.',
      },
      moveResizeWindow: {
        title: 'Move and resize window',
        description: 'Resolve the configured target and set its screen-pixel position and size.',
      },
      maximizeWindow: {
        title: 'Maximize window',
        description: 'Resolve the configured target and maximize its window.',
      },
      minimizeWindow: {
        title: 'Minimize window',
        description: 'Resolve the configured target and minimize its window.',
      },
      restoreWindow: {
        title: 'Restore window',
        description: 'Resolve the configured target and restore its window.',
      },
      getWindowState: {
        title: 'Read window state',
        description:
          'Read the configured target window state, foreground flag, position, and size.',
      },
      waitWindow: {
        title: 'Wait for window',
        description:
          'Wait up to the supplied timeout for the configured application path and window selector to match.',
      },
      waitWindowGone: {
        title: 'Wait for window to disappear',
        description:
          'Wait up to the supplied timeout for every matching target window to disappear.',
      },
      stopTargetApp: {
        title: 'Stop target app',
        description: 'Stop the configured Android package. Unsupported targets return a failure.',
      },
      captureWindow: {
        title: 'Capture window',
        description:
          'Capture the configured target through its selected backend and commit a durable Image BlobRef.',
      },
      controlDualColorBar: {
        title: 'Control dual color bar',
        description:
          'Capture a small region, track a dual color bar, and send left or right keys in one high-frequency node invocation.',
        input: {
          'inner-range': {
            title: 'Pointer color',
            description: 'RGB or HSV range for the narrow pointer.',
          },
          'outer-range': {
            title: 'Target color',
            description: 'RGB or HSV range for the wide target bar.',
          },
          region: {
            title: 'Capture region',
            description: 'Ratio or pixel rectangle captured at the target source.',
          },
          'inner-minimum-width': {
            title: 'Minimum pointer width',
            description: 'Minimum valid pointer cluster width in pixels.',
          },
          'inner-maximum-width': {
            title: 'Maximum pointer width',
            description: 'Maximum valid pointer cluster width; 0 selects automatically.',
          },
          'outer-minimum-width': {
            title: 'Minimum target width',
            description: 'Minimum valid target cluster width; 0 selects automatically.',
          },
          'band-height-ratio': {
            title: 'Band height ratio',
            description: 'Target scan band height relative to the capture region.',
          },
          'band-inner-height-ratio': {
            title: 'Pointer height ratio',
            description: 'Scan band height relative to the pointer height.',
          },
          'inner-confidence-weight': {
            title: 'Pointer confidence weight',
            description: 'Pointer detection contribution to total confidence.',
          },
          'outer-confidence-weight': {
            title: 'Target confidence weight',
            description: 'Target detection contribution to total confidence.',
          },
          'tolerance-ratio': {
            title: 'Width tolerance ratio',
            description: 'Target width multiplier used for the direction dead zone.',
          },
          'minimum-tolerance': {
            title: 'Minimum tolerance',
            description: 'Minimum direction dead zone in pixels.',
          },
          'left-keys': {
            title: 'Left keys',
            description: 'Keys sent when the pointer is to the right of the target.',
          },
          'right-keys': {
            title: 'Right keys',
            description: 'Keys sent when the pointer is to the left of the target.',
          },
          'hold-duration': {
            title: 'Key duration',
            description: 'Milliseconds to hold each directional key press.',
          },
          'neutral-duration': {
            title: 'Neutral interval',
            description: 'Milliseconds to wait while the pointer remains in the dead zone.',
          },
          'cycle-duration': {
            title: 'Minimum frame period',
            description:
              'Minimum milliseconds from one capture-control cycle start to the next; slow frames receive no extra delay.',
          },
          'maximum-iterations': {
            title: 'Maximum frames',
            description: 'Maximum number of frames processed by this invocation.',
          },
          'activation-keys': {
            title: 'Activation keys',
            description: 'Keys sent before detection and retried while the bar has not appeared.',
          },
          'activation-hold-duration': {
            title: 'Activation key duration',
            description: 'Milliseconds to hold each activation key press.',
          },
          'appearance-poll-duration': {
            title: 'Appearance poll interval',
            description: 'Milliseconds between captures while waiting for the bar.',
          },
          'activation-retry-duration': {
            title: 'Activation retry interval',
            description: 'Milliseconds before retrying activation while the bar is absent.',
          },
          'appearance-timeout': {
            title: 'Appearance timeout',
            description: 'Maximum milliseconds to wait for the bar; 0 disables waiting.',
          },
        },
        output: {
          frames: {
            title: 'Frames processed',
            description: 'Frames processed by this invocation.',
          },
          'left-actions': {
            title: 'Left actions',
            description: 'Number of left key actions sent.',
          },
          'right-actions': {
            title: 'Right actions',
            description: 'Number of right key actions sent.',
          },
          'neutral-actions': {
            title: 'Neutral actions',
            description: 'Number of frames inside the direction dead zone.',
          },
          'activation-actions': {
            title: 'Activation actions',
            description: 'Number of activation key actions sent.',
          },
        },
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
          'Check one supplied source image or capture fresh frames, then click the template match center.',
        input: {
          image: {
            title: 'Source image',
            description:
              'Optional; connect Capture Window to skip capture here. A fixed image is checked once and requires zero settle duration.',
          },
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
            description:
              'Maximum wait in milliseconds when no source image is supplied; 0 checks one frame.',
          },
          'poll-interval': {
            title: 'Poll interval',
            description: 'Milliseconds between fresh frame checks.',
          },
          'settle-duration': {
            title: 'Settle duration',
            description:
              'Milliseconds to wait and relocate after a live match; must be 0 with a source image.',
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
        title: 'Replay precise trajectory',
        description:
          'Replay keys, clicks, continuous motion, dragging, scrolling, and relative pointer movement from an InputClip with original timing.',
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
        input: {
          message: {
            title: 'Message',
            description:
              'Optional observable value; when connected, it overrides the configured message.',
          },
        },
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
        description: 'Wait up to 24 hours with cooperative Run cancellation, then emit Done.',
      },
      endBranch: {
        title: 'End branch',
        description: 'Explicitly finish this control-flow branch without emitting another signal.',
      },
      repeat: {
        title: 'Repeat',
        description:
          'Run an isolated activation 0–10,000 times. Body drains before the next iteration; Break and Continue target this exact region.',
      },
      forEach: {
        title: 'For each',
        description:
          'Run an isolated activation once per typed list item, up to 10,000 items, and expose the current Index and Item.',
      },
      retry: {
        title: 'Retry region',
        description:
          'Make 1–100 attempts and retry only failures explicitly routed back to this region. Completed and Exhausted are separate results.',
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
        description:
          'Exposes path, name, extension, media type, size, modified time, and directory flag.',
      },
    },
    builtin: {
      'collection-append': {
        title: 'List Append',
        description:
          'Add one same-typed item and return a new list; the input list is never mutated.',
      },
      'collection-contains': {
        title: 'List Contains',
        description:
          'Test whether a same-typed equatable value occurs in the list using canonical equality; no text coercion is performed.',
      },
      'collection-get': {
        title: 'List Get',
        description:
          'Return the item at a zero-based index. A negative or out-of-range index fails explicitly; use List Length when the index is uncertain.',
      },
      'collection-join': {
        title: 'Join',
        description:
          'Join a list of text values with a separator. Non-text lists are not accepted.',
      },
      'collection-length': {
        title: 'List Length',
        description: 'Return the number of items in a strongly typed list.',
      },
      'collection-slice': {
        title: 'List Slice',
        description:
          'Return a new list from Start for Count items. Negative Start clamps to 0; a negative Count takes the remainder; zero or an out-of-range Start returns an empty list.',
      },
      'collection-split': {
        title: 'Split',
        description:
          'Split text into a list by a separator. Empty text gives an empty list; an empty separator splits into individual characters (CJK-safe).',
      },
      'comparison-equal': {
        title: 'Equals',
        description:
          'Compares the canonical values of one exact type. It never silently turns different types into text.',
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
        title: 'Not equal',
        description:
          'Reports whether the canonical values of one exact type differ, without cross-type text coercion.',
      },
      'conversion-string-to-boolean': {
        title: 'To bool',
        description: 'Strictly parses lowercase true or false; every other value fails.',
      },
      'conversion-string-to-number': {
        title: 'To number',
        description:
          'Strictly parses a finite JSON decimal number. Whitespace, non-numeric text, and out-of-range values fail.',
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
          'Turns an observable inline JSON value into text. Strings keep their contents; other values use canonical JSON. Blobs, images, and runtime streams are not accepted.',
      },
      'geometry-make-point': {
        title: 'Make Point',
        description:
          'Build a typed coordinate from X, Y, and either ratio or px units. Ratio coordinates use the 0–1 frame space.',
      },
      'geometry-offset-point': {
        title: 'Offset Point',
        description:
          'Add horizontal and vertical offsets to a coordinate. Ratio points are clamped inside the screen; pixel points keep their pixel unit.',
      },
      'geometry-point-distance': {
        title: 'Point Distance',
        description:
          'Calculate straight-line distance between two points with the same unit; the result uses that unit and mixed units fail.',
      },
      'geometry-region-around-point': {
        title: 'ROI Around Point',
        description:
          'Create a centered ROI in the center point’s unit. Ratio regions are clamped inside the frame; px regions preserve pixel dimensions.',
      },
      'json-parse': {
        title: 'Parse JSON',
        description:
          'Parses and canonicalizes one JSON document. Trailing values, negative zero, and numbers outside the interoperable profile are rejected.',
      },
      'json-path': {
        title: 'JSON path',
        description:
          'Extracts fields or array items using $, .field, [index], and [*]. A missing result is null; wildcard results preserve missing positions with null.',
      },
      'json-stringify': {
        title: 'To JSON text',
        description:
          'Canonically serializes an observable inline JSON value. Blobs, images, and runtime streams are not accepted directly.',
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
          'Outputs When True or When False according to the condition. Both branches must use the same observable inline JSON type.',
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
        description: 'Flips an integer sign while preserving its type.',
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
        description: 'Divides A by B. A zero divisor or a non-finite result fails explicitly.',
      },
      'math-floor': { title: 'Floor', description: 'Round X down: 3.7 gives 3, -3.2 gives -4.' },
      'math-maximum': { title: 'Max', description: 'The larger of two numbers.' },
      'math-minimum': { title: 'Min', description: 'The smaller of two numbers.' },
      'math-modulo': {
        title: 'Modulo',
        description:
          'Gives the remainder of A divided by B; decimals are supported and a zero divisor fails.',
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
          'Raises Base to Exp. Fractional powers of a negative base report a domain error, non-finite results fail, and 0^0 is 1.',
      },
      'math-round': {
        title: 'Round',
        description:
          'Round X. Digits=0 rounds to integer; 2 keeps two decimals; -2 rounds to hundreds (12345 gives 12300). Digits is capped at +/-15 (beyond float precision).',
      },
      'math-square-root': {
        title: 'Sqrt',
        description:
          'Square root of X. Negative values report a domain error (wire an Abs node first if needed).',
      },
      'math-subtract': {
        title: 'Subtract',
        description: 'Subtracts B from A and gives the difference.',
      },
      'text-contains': {
        title: 'Contains',
        description:
          'Tells whether one text value contains the requested substring. Both inputs must be text, and matching is case-sensitive.',
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
          'Extract the first regular-expression match; with capture groups, group 1 is taken. No match gives an empty string, while an invalid pattern fails the run.',
      },
      'text-regex-match': {
        title: 'Regex Match',
        description:
          "Whether any part of the text matches the regular expression (search semantics: b matches abc). For a full match wrap the pattern in ^ and {'$'}. An invalid pattern fails the run.",
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
}
