export default {
  node: {
    ai: {
      generate: {
        title: 'AI 生成文本',
        description: '根据提示词和可选图像生成文本；可直接连接窗口截图。',
      },
      extract: {
        title: 'AI 提取结构化数据',
        description: '从文本或图像中提取指定字段，并返回可直接连接的结构化数据。',
        config: {
          fields: {
            title: '输出字段',
            description: '添加需要提取的字段；Yotta 会自动生成并校验输出结构。',
            field: '字段 {n}',
            name: '字段名',
            type: '类型',
            field_description: '字段说明（可选）',
            nullable: '内容缺失时允许返回空值',
            empty: '至少添加一个输出字段。',
            add: '添加字段',
            remove: '删除字段 {n}',
            types: {
              string: '文本',
              number: '数字',
              integer: '整数',
              boolean: '是/否',
            },
          },
        },
      },
      config: {
        slot: {
          title: '模型 slot',
          description: '引用设置中已安装 AI 模型的稳定 slot，不接受临时 provider 参数。',
        },
        temperature: {
          title: '温度',
          description: '0 到 2 的采样温度；不填写时使用已安装模型的默认策略。',
        },
        maxOutputTokens: {
          title: '最大输出 token',
          description: '本次请求允许生成的最大 token 数；不填写时使用安装配置。',
        },
        timeoutMilliseconds: {
          title: '超时（毫秒）',
          description: '本次 AI 节点尝试的硬性墙钟时间上限，范围 1000–120000 毫秒。',
        },
      },
    },
    vision: {
      matchTemplate: {
        title: '匹配模板',
        description: '在显式源图像中查找一个不可变模板图像的最佳位置。',
        input: {
          image: {
            title: '源图像',
            description: '待搜索图像；可连接“捕获窗口”或绑定 Image BlobRef。',
          },
          template: { title: '模板图像', description: '此工作流固定使用的精确、不可变模板变体。' },
          region: { title: '搜索区域', description: '源图像内的比例或像素矩形。' },
          threshold: { title: '匹配阈值', description: '0 到 1；最佳归一化分数达到它才算命中。' },
        },
        output: {
          matched: { title: '是否命中', description: '最佳分数达到阈值时为 true。' },
          score: { title: '分数', description: '最佳归一化相关分数。' },
          center: { title: '中心点', description: '最佳候选的像素中心。' },
          bounds: { title: '边界', description: '最佳候选的像素矩形。' },
        },
      },
      findTemplateMatches: {
        title: '查找全部模板',
        description: '经过确定性非极大值抑制后返回全部局部模板命中。',
        input: {
          image: { title: '源图像', description: '待搜索图像。' },
          template: { title: '模板图像', description: '此工作流固定使用的精确、不可变模板变体。' },
          region: { title: '搜索区域', description: '源图像内的比例或像素矩形。' },
          threshold: { title: '匹配阈值', description: '0 到 1 的最低归一化分数。' },
          'minimum-distance': {
            title: '最小距离',
            description: '保留命中中心之间的最小像素距离；0 表示模板较短边的一半。',
          },
        },
        output: { matches: { title: '命中列表', description: '按分数排序的强类型模板命中。' } },
      },
      compareImages: {
        title: '比较图像',
        description: '在同一逻辑区域上测量两张显式图像的视觉差异。',
        input: {
          before: { title: '之前图像', description: '基准图像。' },
          after: { title: '之后图像', description: '与基准比较的图像。' },
          region: { title: '比较区域', description: '在两张图中分别解析的同一比例或像素区域。' },
          'grid-size': { title: '网格大小', description: '方框平均网格的宽和高，范围 1 到 256。' },
          'cell-delta': { title: '单格差值', description: '某网格计为变化所需的 8 位通道差阈值。' },
        },
        output: {
          'changed-ratio': { title: '变化比例', description: '最大通道差超过阈值的网格占比。' },
          'mean-difference': { title: '平均差异', description: '归一化到 0–1 的平均绝对通道差。' },
        },
      },
      decodeQR: {
        title: '解码二维码',
        description: '解码显式图像区域内发现的全部二维码。',
        input: {
          image: { title: '源图像', description: '包含二维码的图像。' },
          region: { title: '解码区域', description: '要解码的比例或像素矩形。' },
        },
        output: { codes: { title: '二维码', description: '解码文本与定位点列表。' } },
      },
      analyzeColor: {
        title: '分析颜色',
        description: '统计落在显式 RGB 或 HSV 范围内的像素。',
        input: {
          image: { title: '源图像', description: '待分析图像。' },
          range: { title: '颜色范围', description: '包含边界的 RGB 或 HSV 通道上下限。' },
          region: { title: '分析区域', description: '要扫描的比例或像素矩形。' },
        },
        output: {
          'pixel-count': { title: '像素数', description: '匹配像素数量。' },
          fraction: { title: '占比', description: '匹配像素数除以区域总像素数。' },
          centroid: { title: '质心', description: '匹配像素的像素质心；无匹配时为零点。' },
        },
      },
      findColorBlobs: {
        title: '查找颜色连通域',
        description: '从 RGB 或 HSV 掩码提取确定性的四邻接连通分量。',
        input: {
          image: { title: '源图像', description: '待分析图像。' },
          range: { title: '颜色范围', description: '包含边界的 RGB 或 HSV 通道上下限。' },
          region: { title: '分析区域', description: '要扫描的比例或像素矩形。' },
          'minimum-area': { title: '最小面积', description: '丢弃像素数小于此值的连通分量。' },
        },
        output: {
          blobs: { title: '颜色连通域', description: '按面积、纵向位置、横向位置稳定排序。' },
        },
      },
      trackDualColorBar: {
        title: '追踪双色条',
        description: '在显式图像区域内按列聚类，追踪窄指针相对宽目标条的位置。',
        input: {
          image: { title: '源图像', description: '待分析图像。' },
          'inner-range': { title: '指针颜色', description: '窄指针的 RGB 或 HSV 范围。' },
          'outer-range': { title: '目标条颜色', description: '宽目标条的 RGB 或 HSV 范围。' },
          region: { title: '追踪区域', description: '包含双色条的比例或像素矩形。' },
          'inner-minimum-width': {
            title: '指针最小宽度',
            description: '有效指针列簇的最小像素宽度。',
          },
          'inner-maximum-width': {
            title: '指针最大宽度',
            description: '有效指针列簇的最大像素宽度；0 表示自动。',
          },
          'outer-minimum-width': {
            title: '目标条最小宽度',
            description: '有效目标条的最小像素宽度；0 表示自动。',
          },
          'band-height-ratio': {
            title: '扫描带高度比例',
            description: '目标条扫描带相对区域高度的比例。',
          },
          'band-inner-height-ratio': {
            title: '指针高度比例',
            description: '扫描带相对指针高度的比例。',
          },
          'inner-confidence-weight': {
            title: '指针置信权重',
            description: '指针检测对总置信度的权重。',
          },
          'outer-confidence-weight': {
            title: '目标条置信权重',
            description: '目标条检测对总置信度的权重。',
          },
        },
        output: {
          found: { title: '已找到', description: '指针和目标条是否同时有效。' },
          'inner-x': { title: '指针 X', description: '指针中心的源图像像素 X。' },
          'outer-x': { title: '目标条 X', description: '目标条中心的源图像像素 X。' },
          'outer-width': { title: '目标条宽度', description: '目标条列簇的像素宽度。' },
          confidence: { title: '置信度', description: '组合检测置信度。' },
          'inner-pixels': { title: '指针像素数', description: '命中指针颜色的像素数。' },
          'outer-pixels': { title: '目标条像素数', description: '命中目标条颜色的像素数。' },
        },
      },
    },
    text: {
      concat: {
        title: '拼接',
        description: '把两个字符串合成一个值。它是数据函数，不声明控制流出口。',
      },
    },
    conversion: {
      blobToStream: {
        title: 'Blob 转流',
        description: '把持久二进制内容打开为带租约的运行时流；该流只在当前运行内有效。',
      },
      streamToBlob: {
        title: '流转 Blob',
        description: '消费当前运行内的带租约流，并提交为可持久保存的二进制内容。',
        config: {
          mediaType: {
            title: '媒体类型',
            description: '写入持久 Blob 的 MIME 类型，例如 image/png。',
          },
        },
      },
    },
    random: {
      integer: {
        title: '随机整数',
        description: '从宿主加密熵源观测一个整数样本，并把结果持久化到 Run 记录。',
      },
      number: {
        title: '随机数值',
        description: '观测一个有限数值样本，并把结果持久化到 Run 记录。',
      },
      boolean: {
        title: '随机布尔值',
        description: '按零到一之间的显式概率观测一个 true/false 样本。',
      },
      choice: {
        title: '随机取值',
        description: '从非空的强类型列表中等概率观测一个元素，并持久化结果。',
      },
    },
    time: {
      observe: {
        title: '观测时间',
        description: '捕获宿主提供的调用时间（Unix 毫秒），并持久化到 Run 记录。',
      },
      stopwatchStart: {
        title: '启动秒表',
        description: '记录本次调用的开始时间，并输出可显式连接的强类型时间点。',
        output: {
          'started-at': { title: '开始时间', description: '本次启动时记录的 Unix 毫秒时间点。' },
        },
      },
      stopwatchRead: {
        title: '读取秒表',
        description: '根据开始时间读取当前经过的毫秒数，不访问进程全局计时器。',
        input: {
          'started-at': { title: '开始时间', description: '连接「启动秒表」输出的开始时间。' },
        },
        output: { elapsed: { title: '已用时间', description: '从开始至今经过的毫秒数。' } },
      },
      stopwatchStop: {
        title: '停止秒表',
        description: '根据开始时间记录最终经过的毫秒数，然后从「完成」继续。',
        input: {
          'started-at': { title: '开始时间', description: '连接「启动秒表」输出的开始时间。' },
        },
        output: { elapsed: { title: '最终用时', description: '停止时记录的总毫秒数。' } },
      },
    },
    state: {
      variable: {
        title: '工作流状态变量',
        description: '选择这个工作流中显式声明的一个强类型状态槽。',
      },
      read: {
        title: '读取状态',
        description:
          '读取 Run 本地强类型状态槽的当前值。它是 recorded 数据 effect，没有控制流出口。',
        output: { result: { title: '当前值', description: '状态槽中类型完全一致的当前值。' } },
      },
      write: {
        title: '写入状态',
        description: '向 Run 本地状态槽写入类型完全一致的值，然后从「完成」继续。',
        input: { value: { title: '新值', description: '要写入状态槽的同类型值。' } },
        output: { result: { title: '已写入值', description: '成功写入后的状态值。' } },
      },
      metadata: {
        title: '状态元数据',
        description: '读取状态槽的 revision 和最后修改时间，不暴露无类型值。',
        output: {
          revision: { title: '修订号', description: '每次成功修改后递增的状态修订号。' },
          'changed-at': { title: '最后变化', description: '最后一次修改的 Unix 毫秒时间。' },
        },
      },
      lastChange: {
        title: '状态最后变化',
        description: '直接读取状态槽最后一次成功修改的 Unix 毫秒时间。',
        output: {
          'changed-at': { title: '最后变化', description: '最后一次修改的 Unix 毫秒时间。' },
        },
      },
      increment: {
        title: '增加状态',
        description: '在 Run State 锁内原子增加 Integer 或 Number，避免读取后写入的竞态。',
        input: { delta: { title: '增量', description: '加到状态当前值上的同类型数值。' } },
        output: { result: { title: '更新后值', description: '原子增加后的状态值。' } },
      },
    },
    script: {
      execute: {
        title: '执行脚本',
        description: '在一次性隔离 worker 中执行 JavaScript，输入和输出均为规范化 JSON。',
        input: {
          input: { title: '输入', description: '以隔离 input 值提供给 guest 的规范化 JSON。' },
        },
        output: {
          result: { title: '结果', description: 'guest 返回的规范化 JSON 兼容值。' },
        },
        config: {
          source: {
            title: 'JavaScript 源码',
            description: '返回一个 JSON 兼容值；guest 只能看到 input 的隔离副本。',
          },
          timeoutMilliseconds: {
            title: '超时（毫秒）',
            description: '本次隔离脚本尝试的硬性墙钟时间上限。',
          },
        },
      },
    },
    filesystem: {
      readText: {
        title: '读取工作区文本',
        description: '从 Yotta 管理的工作流文件区中有界读取文本。',
        input: {
          path: { title: '相对路径', description: '相对于工作流文件区的文件路径。' },
        },
        output: {
          text: { title: '文本', description: '解码后的文件文本。' },
          metadata: { title: '文件信息', description: '该文件的规范化元数据。' },
        },
      },
      readJSON: {
        title: '读取工作区 JSON',
        description: '从工作流文件区读取并严格解析单个 UTF-8 JSON 文档。',
        input: {
          path: { title: '相对路径', description: '相对于工作流文件区的 JSON 文件路径。' },
        },
        output: {
          value: { title: 'JSON 值', description: '解析后的规范化 JSON 值。' },
          text: { title: '源文本', description: '原始解码后的 UTF-8 文档文本。' },
          metadata: { title: '文件信息', description: '该文件的规范化元数据。' },
        },
      },
      stat: {
        title: '检查工作区文件',
        description: '读取规范化元数据，不暴露宿主文件系统的任意路径权限。',
        input: {
          path: { title: '相对路径', description: '相对于工作流文件区的文件或目录路径。' },
        },
        output: {
          metadata: { title: '文件信息', description: '该路径的规范化元数据。' },
        },
      },
      loadImage: {
        title: '加载工作区图片',
        description: '从 Yotta 管理的工作流文件区分块读取 PNG，并提交为持久 Image BlobRef。',
        input: { path: { title: '相对路径', description: '工作流文件区内的 PNG 相对路径。' } },
        output: {
          image: { title: '图片', description: '已验证并持久化的 Image BlobRef。' },
          metadata: { title: '文件信息', description: '加载文件的规范化元数据。' },
        },
      },
      saveImage: {
        title: '保存图片到工作区',
        description: '把持久 Image BlobRef 分块写入 Yotta 管理的工作流文件区。',
        input: {
          image: { title: '图片', description: '要写入的持久 Image BlobRef。' },
          path: { title: '相对路径', description: '工作流文件区内的 PNG 相对路径。' },
        },
        output: { metadata: { title: '文件信息', description: '已写入文件的规范化元数据。' } },
        config: {
          overwrite: {
            title: '覆盖现有文件',
            description: '允许替换工作流文件区中同名的普通文件；不会跟随符号链接。',
          },
        },
      },
      config: {
        encoding: {
          title: '文本编码',
          description: '按 UTF-8、GBK 解码，或先检测 UTF-8 再回退到 GBK。',
        },
        maxBytes: {
          title: '最大字节数',
          description: '文件超过这个有界预算时，在读取前拒绝执行。',
        },
      },
    },
    network: {
      httpGet: {
        title: 'HTTP GET',
        description: '使用相对路径，从配置的 HTTP 基础 URL 读取 UTF-8 文本。',
        input: {
          path: { title: '相对路径', description: '相对于已配置 HTTP 基础 URL 解析的路径。' },
          query: { title: '查询参数', description: '值均为查询字符串数组的 JSON 对象。' },
        },
        output: {
          status: { title: '状态码', description: 'HTTP 响应状态码。' },
          body: { title: '响应正文', description: '有界的 UTF-8 响应正文。' },
          'content-type': {
            title: '内容类型',
            description: '响应的 Content-Type 头；未提供时为空字符串。',
          },
        },
        config: {
          slot: {
            title: 'HTTP 目标槽位',
            description: '为这次请求选择已配置的 HTTP 目标。',
          },
        },
      },
    },
    application: {
      launch: {
        title: '启动应用',
        description: '直接使用应用槽位中配置的程序路径和参数启动。',
      },
      terminate: {
        title: '终止应用',
        description: '终止与配置路径对应的进程，并返回终止数量。',
        output: {
          'terminated-count': {
            title: '终止数量',
            description: '本次调用终止的匹配进程数量。',
          },
        },
      },
      config: {
        slot: {
          title: '应用槽位',
          description: '选择已经配置的桌面应用。',
        },
      },
    },
    automation: {
      config: {
        slot: {
          title: '窗口目标槽位',
          description: '选择已绑定应用路径、窗口标题与窗口类的目标。',
        },
      },
      clickPointer: { title: '点击指针', description: '在配置目标内执行一次点击。' },
      movePointer: {
        title: '移动指针',
        description: '以即时、匀速直线或贝塞尔曲线移动到目标坐标。',
      },
      scrollPointer: { title: '滚动指针', description: '在配置目标内执行滚动。' },
      dragPointer: {
        title: '拖拽指针',
        description: '按指定移动方式执行按下、移动和释放。',
      },
      movePointerRelative: {
        title: '相对移动指针',
        description: '向配置目标发送相对位移。',
      },
      pressKeys: { title: '按下组合键', description: '原子按下并反向释放一组规范化按键。' },
      holdKeys: {
        title: '按住按键',
        description: '按下规范化按键并输出 Run 级租约；必须连接释放节点，异常结束也会自动释放。',
      },
      holdPointerButton: {
        title: '按住指针按键',
        description: '在目标坐标按住鼠标键并输出 Run 级租约；异常结束时自动释放。',
      },
      releaseHeldInput: {
        title: '释放按住输入',
        description: '消费按住输入租约并立即释放该租约拥有的所有按键或鼠标键。',
      },
      typeText: {
        title: '输入文本',
        description: '向配置目标输入 Unicode 文本。',
      },
      activateWindow: {
        title: '激活目标',
        description: '解析当前配置目标，然后置前桌面窗口或启动 Android 包。',
      },
      closeWindow: {
        title: '关闭窗口',
        description: '解析当前配置目标并向窗口发送关闭请求。',
      },
      moveResizeWindow: {
        title: '移动并调整窗口',
        description: '解析当前配置目标，并按屏幕像素设置窗口位置与尺寸。',
      },
      maximizeWindow: { title: '最大化窗口', description: '解析当前配置目标并最大化窗口。' },
      minimizeWindow: { title: '最小化窗口', description: '解析当前配置目标并最小化窗口。' },
      restoreWindow: { title: '还原窗口', description: '解析当前配置目标并还原窗口。' },
      getWindowState: {
        title: '读取窗口状态',
        description: '读取配置目标当前的窗口状态、前台标记、屏幕位置与尺寸。',
      },
      waitWindow: {
        title: '等待窗口出现',
        description: '在给定时限内按已配置应用路径和窗口选择器等待窗口出现。',
      },
      waitWindowGone: {
        title: '等待窗口消失',
        description: '在给定时限内等待配置目标的匹配窗口全部消失。',
      },
      stopTargetApp: {
        title: '停止目标应用',
        description: '停止配置目标中的 Android 包；不支持该操作的目标会返回失败。',
      },
      captureWindow: {
        title: '截取窗口',
        description: '通过配置的截图后端截取目标画面，并提交持久 Image BlobRef。',
      },
      controlDualColorBar: {
        title: '高频控制双色条',
        description: '在单次节点调用内循环抓取指定区域、追踪双色条并按偏差发送左右按键。',
        input: {
          'inner-range': { title: '指针颜色', description: '窄指针的 RGB 或 HSV 范围。' },
          'outer-range': { title: '目标条颜色', description: '宽目标条的 RGB 或 HSV 范围。' },
          region: { title: '抓取区域', description: '从目标窗口源头抓取的比例或像素矩形。' },
          'inner-minimum-width': {
            title: '指针最小宽度',
            description: '有效指针列簇的最小像素宽度。',
          },
          'inner-maximum-width': {
            title: '指针最大宽度',
            description: '有效指针列簇的最大像素宽度；0 表示自动。',
          },
          'outer-minimum-width': {
            title: '目标条最小宽度',
            description: '有效目标条列簇的最小像素宽度；0 表示自动。',
          },
          'band-height-ratio': {
            title: '扫描带高度比例',
            description: '目标条扫描带相对抓取区域高度的比例。',
          },
          'band-inner-height-ratio': {
            title: '指针高度比例',
            description: '扫描带相对指针高度的比例。',
          },
          'inner-confidence-weight': {
            title: '指针置信权重',
            description: '指针检测对总置信度的权重。',
          },
          'outer-confidence-weight': {
            title: '目标条置信权重',
            description: '目标条检测对总置信度的权重。',
          },
          'tolerance-ratio': {
            title: '宽度容差比例',
            description: '目标条宽度乘以该比例作为方向死区。',
          },
          'minimum-tolerance': { title: '最小容差', description: '方向死区的最小像素值。' },
          'left-keys': { title: '左方向键', description: '指针位于目标右侧时发送的按键。' },
          'right-keys': { title: '右方向键', description: '指针位于目标左侧时发送的按键。' },
          'hold-duration': { title: '按键时长', description: '每次方向按键保持的毫秒数。' },
          'neutral-duration': { title: '中性间隔', description: '指针位于死区内时等待的毫秒数。' },
          'cycle-duration': {
            title: '最小帧周期',
            description: '每轮截图和控制从开始到下一轮开始的最短毫秒数；慢帧不会额外等待。',
          },
          'maximum-iterations': { title: '最大帧数', description: '本次控制最多处理的画面帧数。' },
          'activation-keys': {
            title: '启动按键',
            description: '开始检测前发送，并在进度条尚未出现时重试的按键。',
          },
          'activation-hold-duration': {
            title: '启动按键时长',
            description: '每次启动按键保持的毫秒数。',
          },
          'appearance-poll-duration': {
            title: '出现检查间隔',
            description: '等待进度条出现时两次抓取之间的毫秒数。',
          },
          'activation-retry-duration': {
            title: '启动重试间隔',
            description: '进度条尚未出现时再次发送启动按键的间隔。',
          },
          'appearance-timeout': {
            title: '出现超时',
            description: '等待进度条出现的最长毫秒数；0 表示不等待。',
          },
        },
        output: {
          frames: { title: '处理帧数', description: '本次调用处理的画面帧数。' },
          'left-actions': { title: '左按键次数', description: '本次调用发送左方向按键的次数。' },
          'right-actions': { title: '右按键次数', description: '本次调用发送右方向按键的次数。' },
          'neutral-actions': { title: '中性次数', description: '本次调用落在方向死区内的次数。' },
          'activation-actions': { title: '启动次数', description: '本次调用发送启动按键的次数。' },
        },
      },
      waitTemplate: {
        title: '等待模板出现',
        description: '在精确窗口中持续获取新画面，直到指定模板出现或超时。',
        input: {
          template: { title: '模板图片', description: '要等待的持久模板图片。' },
          region: { title: '搜索区域', description: '在窗口画面内搜索的区域。' },
          threshold: { title: '匹配阈值', description: '0 到 1；分数达到该值才算出现。' },
          timeout: { title: '等待超时', description: '最长等待时间（毫秒）；0 表示只检查一次。' },
          'poll-interval': { title: '检查间隔', description: '两次新画面检查之间的时间（毫秒）。' },
          'settle-duration': {
            title: '稳定等待',
            description: '首次命中后等待并重新定位的时间（毫秒）。',
          },
        },
        output: {
          matched: { title: '已匹配', description: '模板在最后一次画面中是否达到阈值。' },
          score: { title: '匹配分数', description: '最后一次模板匹配分数。' },
          center: { title: '中心点', description: '命中区域在捕获画面中的像素中心。' },
          bounds: { title: '命中区域', description: '命中区域在捕获画面中的像素边界。' },
        },
      },
      clickTemplate: {
        title: '点击模板',
        description: '复用给定源画面检查一次，或持续捕获精确窗口，再点击模板命中中心。',
        input: {
          image: {
            title: '源画面',
            description:
              '可选；连接“截取窗口”可跳过本节点截图。固定画面只检查一次，且稳定等待必须为 0。',
          },
          template: { title: '模板图片', description: '要等待并点击的持久模板图片。' },
          region: { title: '搜索区域', description: '在窗口画面内搜索的区域。' },
          threshold: { title: '匹配阈值', description: '0 到 1；分数达到该值才允许点击。' },
          timeout: {
            title: '等待超时',
            description: '未提供源画面时的最长等待时间（毫秒）；0 表示只检查一次。',
          },
          'poll-interval': { title: '检查间隔', description: '两次新画面检查之间的时间（毫秒）。' },
          'settle-duration': {
            title: '稳定等待',
            description: '实时截图首次命中后等待并重新定位的时间；提供源画面时必须为 0。',
          },
          button: { title: '指针按键', description: '点击时使用左键、右键或中键。' },
          'hold-duration': { title: '按住时长', description: '按下到释放之间的时间（毫秒）。' },
        },
        output: {
          matched: { title: '已匹配', description: '点击前的最终画面是否达到阈值。' },
          score: { title: '匹配分数', description: '点击前的最终模板匹配分数。' },
          center: { title: '点击中心', description: '用于换算窗口内点击位置的像素中心。' },
          bounds: { title: '命中区域', description: '点击前命中区域的像素边界。' },
        },
      },
      waitTemplateGone: {
        title: '等待模板消失',
        description: '在精确窗口中持续获取新画面，直到指定模板不再出现或超时。',
        input: {
          template: { title: '模板图片', description: '要等待其消失的持久模板图片。' },
          region: { title: '搜索区域', description: '在窗口画面内搜索的区域。' },
          threshold: { title: '匹配阈值', description: '低于该分数时认为模板已经消失。' },
          timeout: { title: '等待超时', description: '最长等待时间（毫秒）；0 表示只检查一次。' },
          'poll-interval': { title: '检查间隔', description: '两次新画面检查之间的时间（毫秒）。' },
        },
        output: {
          matched: { title: '仍然匹配', description: '最后一次画面中模板是否仍达到阈值。' },
          score: { title: '匹配分数', description: '最后一次模板匹配分数。' },
          center: { title: '最后中心点', description: '最后一次匹配得到的像素中心。' },
          bounds: { title: '最后命中区域', description: '最后一次匹配得到的像素边界。' },
        },
      },
      waitStable: {
        title: '等待画面稳定',
        description: '持续捕获精确目标，直到指定区域在稳定时长内保持低于变化阈值。',
        input: {
          region: { title: '观察区域', description: '精确目标画面内要比较的区域。' },
          threshold: { title: '变化阈值', description: '变化格比例不高于该值时认为稳定。' },
          timeout: { title: '等待超时', description: '等待稳定的最长毫秒数。' },
          'poll-interval': { title: '采样间隔', description: '两次画面捕获之间的毫秒数。' },
          'grid-size': { title: '采样网格', description: '每个方向的有界降采样格数。' },
          'cell-delta': { title: '单格差值', description: '把单个网格视为已变化的颜色差阈值。' },
          'stable-duration': { title: '稳定时长', description: '连续保持稳定所需的毫秒数。' },
        },
        output: {
          'changed-ratio': { title: '变化比例', description: '最后两帧发生明显变化的网格比例。' },
          'mean-difference': { title: '平均差值', description: '最后两帧网格颜色的平均绝对差。' },
        },
      },
      waitChange: {
        title: '等待画面变化',
        description: '持续捕获精确目标，直到指定区域相对基准画面的变化达到阈值。',
        input: {
          region: { title: '观察区域', description: '精确目标画面内要比较的区域。' },
          threshold: { title: '变化阈值', description: '变化格比例达到该值时报告变化。' },
          timeout: { title: '等待超时', description: '等待变化的最长毫秒数。' },
          'poll-interval': { title: '采样间隔', description: '两次画面捕获之间的毫秒数。' },
          'grid-size': { title: '采样网格', description: '每个方向的有界降采样格数。' },
          'cell-delta': { title: '单格差值', description: '把单个网格视为已变化的颜色差阈值。' },
        },
        output: {
          'changed-ratio': { title: '变化比例', description: '相对基准帧发生明显变化的网格比例。' },
          'mean-difference': { title: '平均差值', description: '相对基准帧网格颜色的平均绝对差。' },
        },
      },
      playInputClip: {
        title: '回放精准轨迹',
        description:
          '按原始时序回放 InputClip 中的按键、点击、连续移动、拖拽、滚轮与相对鼠标位移。',
      },
      playMacro: {
        title: '回放键鼠宏',
        description: '校验原子宏并在独占的精确目标会话中保持真实按下、松开与等待顺序。',
      },
    },
    observability: {
      log: {
        title: '写入日志',
        description: '写入一条有界、带 Run 归属的消息，action journal 只记录其摘要。',
        input: {
          message: {
            title: '消息',
            description: '可选的可观察值；连接后会覆盖检查器中配置的消息。',
          },
        },
        config: {
          message: {
            title: '消息',
            description: '在检查器中设置消息；连接输入后以输入值为准。',
          },
          level: {
            title: '日志级别',
            description: '选择调试、信息、警告或错误级别。',
          },
        },
      },
    },
    event: {
      runStarted: {
        title: 'Run 开始',
        description: '这个 Program Run 开始时，准确发出一次「已开始」。',
      },
    },
    control: {
      throw: {
        title: '令工作流失败',
        description: '以稳定的 control.thrown 错误码结束当前分支，不产生成功出口。',
      },
      branch: {
        title: '分支',
        description: '根据条件只从「真」或「假」中的一个路径继续。',
      },
      delay: {
        title: '延迟',
        description: '等待最长 24 小时且可随 Run 取消的时长，然后发出「完成」。',
      },
      endBranch: {
        title: '结束分支',
        description: '显式结束当前控制流分支，不再发出信号。',
      },
      repeat: {
        title: '重复',
        description:
          '运行 0 到 10000 次隔离 activation。本轮「循环体」排空后才进入下一轮，「中断」和「继续」只作用于这个 region。',
      },
      forEach: {
        title: '遍历',
        description: '对强类型列表最多 10000 个元素逐一运行隔离 activation，并输出当前索引和元素。',
      },
      retry: {
        title: '重试区域',
        description:
          '执行 1 到 100 次尝试，只重试显式路由回该 region 的失败；「完成」和「耗尽」是两个独立结果。',
      },
      switch: {
        title: '强类型 Switch',
        description: '按顺序比较可配置数量的同类型 case，并只触发首个匹配出口或默认出口。',
        config: {
          caseCount: {
            title: '分支数量',
            description:
              '设置 1 到 32 个稳定 case 输入与对应执行出口；减少数量会移除超出范围的连线。',
          },
        },
        input: {
          value: { title: '待匹配值', description: '决定控制流出口的强类型值。' },
          'case-1': { title: '分支 1', description: '第一个可选同类型匹配值。' },
          'case-2': { title: '分支 2', description: '第二个可选同类型匹配值。' },
          'case-3': { title: '分支 3', description: '第三个可选同类型匹配值。' },
          'case-4': { title: '分支 4', description: '第四个可选同类型匹配值。' },
          'case-5': { title: '分支 5', description: '第五个可选同类型匹配值。' },
          'case-6': { title: '分支 6', description: '第六个可选同类型匹配值。' },
          'case-7': { title: '分支 7', description: '第七个可选同类型匹配值。' },
          'case-8': { title: '分支 8', description: '第八个可选同类型匹配值。' },
        },
      },
    },

    structure: {
      breakPoint: { title: '拆分坐标', description: '把坐标拆成 X、Y 和单位三个强类型输出。' },
      breakRegion: { title: '拆分区域', description: '把区域拆成位置、宽高和单位强类型输出。' },
      breakTemplateMatch: {
        title: '拆分模板匹配',
        description: '取得模板匹配的分数、中心点和命中区域。',
      },
      breakQRCode: { title: '拆分二维码', description: '取得二维码文字和定位点列表。' },
      breakColorBlob: { title: '拆分颜色块', description: '取得颜色块面积、中心点和边界区域。' },
      breakFileMetadata: {
        title: '拆分文件信息',
        description: '取得路径、名称、扩展名、媒体类型、大小、修改时间和目录标记。',
      },
    },
    builtin: {
      'collection-append': {
        title: '追加元素',
        description: '追加一个同类型元素并返回新列表；输入列表本身不会被修改。',
      },
      'collection-contains': {
        title: '列表包含',
        description: '按规范值判断列表是否包含同类型、可比较的元素；不会转成文字后再比较。',
      },
      'collection-get': {
        title: '取列表元素',
        description:
          '按从 0 开始的序号取元素；负数或越界会明确失败，序号不确定时先接「列表长度」。',
      },
      'collection-join': {
        title: '拼接列表',
        description: '用分隔符拼接文字列表；不接受其它元素类型的列表。',
      },
      'collection-length': {
        title: '列表长度',
        description: '返回强类型列表中的元素数量。',
      },
      'collection-slice': {
        title: '截取列表',
        description:
          '从 Start 起取 Count 个元素并返回新列表。Start 为负时按 0；Count 为负时取到末尾；Count 为 0 或 Start 越界时返回空列表。',
      },
      'collection-split': {
        title: '拆分文本',
        description:
          '把文本按分隔符拆成列表。文本留空得空列表；分隔符留空则一个字一个字拆（中文安全）。',
      },
      'comparison-equal': {
        title: '等于',
        description: '按同一精确类型的规范值判断是否相等；不会把不同类型偷偷转成文字。',
      },
      'comparison-greater-or-equal': {
        title: '大于等于',
        description: '判断 A 是不是 ≥ B，给出真/假。',
      },
      'comparison-greater-than': { title: '大于', description: '判断 A 是不是 > B，给出真/假。' },
      'comparison-less-or-equal': {
        title: '小于等于',
        description: '判断 A 是不是 ≤ B，给出真/假。',
      },
      'comparison-less-than': { title: '小于', description: '判断 A 是不是 < B，给出真/假。' },
      'comparison-not-equal': {
        title: '不等于',
        description: '按同一精确类型的规范值判断是否不相等。',
      },
      'conversion-string-to-boolean': {
        title: '转布尔',
        description: '严格解析小写文字 true 或 false；其它内容会失败。',
      },
      'conversion-string-to-number': {
        title: '转数字',
        description: '严格解析有限十进制数字；空白、纯字母或超出范围会失败。',
      },
      'conversion-string-to-integer': {
        title: '文字转整数',
        description: '严格解析安全整数；小数、空白或超出安全范围会失败。',
      },
      'conversion-truncate-to-integer': {
        title: '截断为整数',
        description: '丢弃小数部分并检查安全整数范围，例如 -1.9 变成 -1。',
      },
      'conversion-floor-to-integer': {
        title: '向下取整',
        description: '向负无穷取整并检查安全整数范围，例如 -1.1 变成 -2。',
      },
      'conversion-ceiling-to-integer': {
        title: '向上取整',
        description: '向正无穷取整并检查安全整数范围，例如 1.1 变成 2。',
      },
      'conversion-round-to-integer': {
        title: '四舍五入为整数',
        description: '取最近整数并检查安全整数范围；半数远离零取整。',
      },
      'conversion-to-string': {
        title: '转字符串',
        description:
          '把可观察的 JSON 内联值转成文字；字符串保留原内容，其它值使用规范 JSON。Blob、图像和运行时流不支持直接转换。',
      },
      'geometry-make-point': {
        title: '组装坐标',
        description:
          '用 X、Y 和 ratio 或 px 单位组装强类型坐标；ratio 坐标使用 0 到 1 的画面空间。',
      },
      'geometry-offset-point': {
        title: '偏移坐标',
        description:
          '在一个坐标上加水平/垂直偏移。百分比坐标会限制在屏幕内；像素坐标保持像素单位不缩放。',
      },
      'geometry-point-distance': {
        title: '两点距离',
        description: '计算两个同单位坐标的直线距离，结果沿用该单位；ratio 与 px 混用会失败。',
      },
      'geometry-region-around-point': {
        title: '点周围区域',
        description: '按中心点的单位生成居中 ROI；ratio 区域会限制在画面内，px 区域保留像素宽高。',
      },
      'json-parse': {
        title: '解析 JSON',
        description:
          '把单个 JSON 文档解析并规范化成结构化值；不接受尾随值、负零或超出可互操作范围的数字。',
      },
      'json-path': {
        title: '取 JSON 路径',
        description:
          '从 JSON 值里取字段或数组项。支持 $、.字段、[序号] 和 [*]；未找到时得 null，通配列表会用 null 保留缺失项的位置。',
      },
      'json-stringify': {
        title: '转 JSON 文本',
        description:
          '把可观察的 JSON 内联值规范序列化成文本；Blob、图像和运行时流不支持直接序列化。',
      },
      'logic-and': {
        title: '逻辑与',
        description: '两个条件都为真时才给出真，否则给假。没接线的输入默认当真，不会干扰结果。',
      },
      'logic-not': { title: '逻辑非', description: '把真假反过来：真变假、假变真。' },
      'logic-or': {
        title: '逻辑或',
        description: '两个条件只要有一个为真就给出真，都为假才给假。',
      },
      'logic-select': {
        title: '三元选择',
        description:
          '条件为真时输出「为真时」的值，为假时输出「为假时」的值；两路必须是同一种可观察的 JSON 内联类型。',
      },
      'math-absolute': { title: '绝对值', description: '取 X 的绝对值（负变正）。' },
      'math-add': { title: '加', description: '把两个数字相加，给出和。' },
      'math-integer-add': {
        title: '整数加法',
        description: '把两个整数相加，结果仍为整数；超出安全整数范围会失败。',
      },
      'math-integer-subtract': {
        title: '整数减法',
        description: '用整数 A 减去整数 B，结果仍为整数；超出安全整数范围会失败。',
      },
      'math-integer-multiply': {
        title: '整数乘法',
        description: '把两个整数相乘，结果仍为整数；超出安全整数范围会失败。',
      },
      'math-integer-modulo': {
        title: '整数取模',
        description: '计算整数余数，结果仍为整数；除数为 0 会失败。',
      },
      'math-integer-negate': {
        title: '整数取负',
        description: '改变整数符号并保持整数类型。',
      },
      'math-integer-absolute': {
        title: '整数绝对值',
        description: '取得整数绝对值并保持整数类型。',
      },
      'math-integer-minimum': {
        title: '取较小整数',
        description: '从两个整数中给出较小值，结果仍为整数。',
      },
      'math-integer-maximum': {
        title: '取较大整数',
        description: '从两个整数中给出较大值，结果仍为整数。',
      },
      'math-integer-clamp': {
        title: '限制整数范围',
        description: '把整数限制在最小值和最大值之间，结果仍为整数。',
      },
      'math-ceiling': {
        title: '向上取整',
        description: '把 X 往大的方向取整：3.2 得 4，-3.7 得 -3。',
      },
      'math-clamp': {
        title: '限制范围',
        description:
          '把 X 限制在 Min~Max 里：小于 Min 出 Min，大于 Max 出 Max，否则原样出。Min 比 Max 大时自动交换。',
      },
      'math-divide': {
        title: '除',
        description: '用 A 除以 B，给出商。除数为 0 或结果超出有限数范围时会失败。',
      },
      'math-floor': {
        title: '向下取整',
        description: '把 X 往小的方向取整：3.7 得 3，-3.2 得 -4。',
      },
      'math-maximum': { title: '取较大', description: '两个数里取较大的那个。' },
      'math-minimum': { title: '取较小', description: '两个数里取较小的那个。' },
      'math-modulo': {
        title: '取模',
        description: '求 A 除以 B 的余数，支持小数；除数为 0 时会失败。',
      },
      'math-multiply': { title: '乘', description: '把两个数字相乘，给出积。' },
      'math-negate': { title: '取负', description: '把数字变号：正数变负、负数变正。' },
      'math-power': {
        title: '乘方',
        description:
          '算 Base 的 Exp 次方。负底数的分数次方会报定义域错误，非有限结果会失败；0 的 0 次方按 1 处理。',
      },
      'math-round': {
        title: '四舍五入',
        description:
          '把 X 四舍五入。位数=0 取到整数；位数=2 保留 2 位小数；位数=-2 取整到百位（12345 得 12300）。位数最多 ±15（再多超出小数精度，按 ±15 算）。',
      },
      'math-square-root': {
        title: '开平方',
        description: '算 X 的平方根。X 为负数时会报定义域错误（需要时先接「绝对值」节点）。',
      },
      'math-subtract': { title: '减', description: '用 A 减去 B，给出差。' },
      'text-contains': {
        title: '包含',
        description:
          '判断文本里是否出现指定子串，给出真/假。两个输入都必须是文字，并且区分大小写。',
      },
      'text-ends-with': {
        title: '结尾是',
        description: '判断文本是否以 Suffix 结尾。Suffix 留空恒为真。',
      },
      'text-index-of': {
        title: '查找位置',
        description:
          'Sub 在文本里第一次出现的位置（从 0 数，中文一个字算 1 个）。找不到得 -1。只想判断"包含吗"请用「包含」节点。',
      },
      'text-length': {
        title: '字符串长度',
        description:
          '数一段文字有多长，给出字符数。中文一个字算 1 个，与「截取文本」「查找位置」的位置口径一致。',
      },
      'text-lowercase': { title: '转小写', description: '把英文字母全部转成小写。' },
      'text-regex-extract': {
        title: '正则提取',
        description:
          '从文本里提取第一段匹配正则表达式的内容；表达式带括号分组时取第 1 组。没匹配到时得空串，表达式无效时运行失败。',
      },
      'text-regex-match': {
        title: '正则匹配',
        description:
          '判断文本里是否有匹配正则表达式的部分（是"包含"式：abc 用 b 也算中）。要整串完全匹配，给表达式首尾加 ^ 和 {\'$\'}。表达式无效时运行失败。',
      },
      'text-replace': {
        title: '替换文本',
        description:
          '把文本里的 Old 换成 New。「全部替换」开着换所有，关掉只换第一处。Old 留空时原样返回。',
      },
      'text-starts-with': {
        title: '开头是',
        description: '判断文本是否以 Prefix 开头。Prefix 留空恒为真。',
      },
      'text-substring': {
        title: '截取文本',
        description:
          '从第 Start 个字符开始截 Length 个字符（中文一个字算 1 个）。Length 填 -1（默认）截到末尾，0 得空串。Start 超出范围得空串。',
      },
      'text-trim': { title: '去首尾空白', description: '去掉文本开头和结尾的空格、换行、制表符。' },
      'text-uppercase': { title: '转大写', description: '把英文字母全部转成大写。' },
    },
  },
}
