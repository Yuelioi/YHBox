<template>
  <div class="px-8 py-6 space-y-6">
    <!-- 快速上手 -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-3">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-rocket" class="size-4 text-primary" />
        <h2 class="text-sm font-medium text-highlighted">快速上手</h2>
      </div>
      <p class="text-xs text-muted">
        YHBox 提供两条路径：开箱即用的 5 个预设 Bot，以及完全自定义的容器编排系统。
      </p>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div class="rounded-lg bg-default/50 border border-default/60 p-4 space-y-2">
          <div class="flex items-center gap-2">
            <UIcon name="i-tabler-bolt" class="size-3.5 text-primary" />
            <span class="text-xs font-medium text-highlighted">A. 预设 Bot（开箱即用）</span>
          </div>
          <ol class="space-y-1 text-xs text-toned list-decimal pl-5 marker:text-dimmed">
            <li>启动 <span class="text-highlighted">异环</span> 游戏</li>
            <li>侧栏底部应显示 <span class="text-primary">已检测</span></li>
            <li>切到对应标签（钓鱼 / 店长 / 弹琴 / 战斗 / 音游）</li>
            <li>调整配置 → 点 <span class="text-primary">开始</span></li>
          </ol>
        </div>
        <div class="rounded-lg bg-default/50 border border-default/60 p-4 space-y-2">
          <div class="flex items-center gap-2">
            <UIcon name="i-tabler-schema" class="size-3.5 text-fuchsia-300" />
            <span class="text-xs font-medium text-highlighted">B. 容器编排（自定义）</span>
          </div>
          <ol class="space-y-1 text-xs text-toned list-decimal pl-5 marker:text-dimmed">
            <li>侧栏点 <span class="text-highlighted">容器</span> → 新建</li>
            <li>编辑器拖节点 / 连线 / 配置参数</li>
            <li>"试运行" 立即跑；或在 <span class="text-highlighted">计划任务</span> 安排热键 / 定时</li>
            <li>常用片段可"折叠为子图"或"发布到库"复用</li>
          </ol>
        </div>
      </div>
    </section>

    <!-- v2 核心概念 -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-bulb" class="size-4 text-amber-300" />
        <h2 class="text-sm font-medium text-highlighted">容器系统核心概念</h2>
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div
          v-for="c in concepts"
          :key="c.name"
          class="rounded-lg bg-default/50 border border-default/60 p-4"
        >
          <div class="flex items-center gap-2 mb-2">
            <UIcon :name="c.icon" class="size-4" :class="c.iconClass" />
            <span class="text-sm font-medium text-highlighted">{{ c.name }}</span>
          </div>
          <p class="text-xs text-muted leading-relaxed">{{ c.desc }}</p>
        </div>
      </div>
    </section>

    <!-- 节点类别 -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-3">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-layout-grid" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">节点类别速查</h2>
      </div>
      <div class="space-y-2 text-xs">
        <div v-for="g in nodeGroups" :key="g.label" class="flex gap-3">
          <span class="text-toned shrink-0 w-12 font-medium">{{ g.label }}</span>
          <span class="text-muted leading-relaxed">{{ g.desc }}</span>
        </div>
      </div>
    </section>

    <!-- 预设 Bot 模块 -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-list-check" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">预设 Bot 模块</h2>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div
          v-for="m in modules"
          :key="m.name"
          class="rounded-lg bg-default/50 border border-default/60 p-4"
        >
          <div class="flex items-center gap-2 mb-2">
            <UIcon :name="m.icon" class="size-4 text-muted" />
            <span class="text-sm font-medium text-highlighted">{{ m.name }}</span>
          </div>
          <p class="text-xs text-muted leading-relaxed">{{ m.desc }}</p>
        </div>
      </div>
    </section>

    <!-- FAQ -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-help-circle" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">常见问题</h2>
      </div>

      <div
        v-for="(item, idx) in faq"
        :key="idx"
        class="border-b border-default/60 last:border-0 pb-3 last:pb-0"
      >
        <h3 class="text-sm font-medium text-default mb-1.5 flex items-start gap-2">
          <UIcon name="i-tabler-chevron-right" class="size-3.5 mt-0.5 text-dimmed shrink-0" />
          <span>{{ item.q }}</span>
        </h3>
        <p class="text-xs text-muted leading-relaxed pl-5">{{ item.a }}</p>
      </div>
    </section>

    <!-- 故障排查 -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-3">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-tool" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">故障排查</h2>
      </div>

      <div class="space-y-3 text-sm">
        <div
          v-for="(item, idx) in troubleshoot"
          :key="idx"
          class="flex items-start gap-3 rounded-lg bg-default/40 border border-default/60 p-3"
        >
          <UIcon name="i-tabler-alert-triangle" class="size-4 text-warning shrink-0 mt-0.5" />
          <div>
            <div class="text-default mb-0.5">{{ item.symptom }}</div>
            <div class="text-xs text-muted leading-relaxed">
              {{ item.fix }}
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
const concepts = [
  {
    name: '容器 (Container)',
    icon: 'i-tabler-package',
    iconClass: 'text-primary',
    desc: '一个自包含的可执行编排单元。含一张主图（节点 + 边）、所属子图集合、变量、热键。导出后可整体分享给其他机器用户。',
  },
  {
    name: '子图 (Subgraph)',
    icon: 'i-tabler-schema',
    iconClass: 'text-fuchsia-300',
    desc: '容器内可复用的执行函数：有一个入口 (SubgraphInput)、一个或多个命名出口 (SubgraphOutput)。父图通过 Subgraph 调用节点执行子图，按出口名分流。',
  },
  {
    name: '库 (Library)',
    icon: 'i-tabler-books',
    iconClass: 'text-emerald-300',
    desc: '全机器共享的子图 + 模板仓库。从库拖入容器走 copy-on-use（复制独立副本，互不影响）。可把容器子图反向"发布到库"分享。',
  },
  {
    name: '计划任务 (Schedule)',
    iconClass: 'text-amber-300',
    icon: 'i-tabler-calendar-clock',
    desc: '安排容器自动执行：热键触发 / 定时（cron / 间隔）/ 一次性 / 手动。多个容器可挂同一计划，按顺序串行执行。',
  },
]

const nodeGroups = [
  { label: '控制流', desc: 'Start / Sleep / Loop / If / Parallel / Race / Stop / Break / Continue' },
  { label: '变量', desc: 'SetVar / IncVar — 容器作用域 + 子图局部作用域（runtime 自动隔离）' },
  { label: '图像', desc: 'WaitTemplate / CheckTemplate / ClickTemplate / DetectColor — 模板匹配与颜色检测' },
  { label: '输入', desc: 'ClickAt / KeyPress / MouseMoveRel / Scroll — 注入输入到游戏窗口' },
  { label: '事件', desc: 'OnEvent — listener 形态入口，匹配触发即 spawn 子图执行' },
  { label: '子图', desc: 'Subgraph 调用 / SubgraphInput 入口 / SubgraphOutput 出口' },
  { label: '系统', desc: 'BringGameForeground — 把游戏窗口拉到前台（前台模式必备）' },
  { label: '配置', desc: 'MouseCalibration — 标定本机 360° HID counts，给 MouseMoveRel 缩放用' },
  { label: '调试', desc: 'Log / Toast' },
]

const modules = [
  {
    name: '钓鱼',
    icon: 'i-tabler-fish',
    desc: '全自动钓鱼：识别上钩，自动按键收鱼。可开启鱼仓满后自动出售普通鱼。',
  },
  {
    name: '店长',
    icon: 'i-tabler-tools-kitchen-2',
    desc: '锤子连点：识别画面里的锤子图标并自动点击。点击间隔可调（100–10000 ms）。',
  },
  {
    name: '弹琴',
    icon: 'i-tabler-music',
    desc: '导入 MIDI 文件自动按键演奏。支持 36 键 / 21 键、智能选轨、自动八度对齐。',
  },
  {
    name: '战斗',
    icon: 'i-tabler-swords',
    desc: '全局热键一键切换上阵队伍 1–6。可配置修饰键组合避开占用。',
  },
  {
    name: '音游',
    icon: 'i-tabler-keyboard',
    desc: '节奏游戏辅助。',
  },
]

const faq = [
  {
    q: '什么时候用预设 Bot，什么时候用容器？',
    a: '5 个预设 Bot 是开箱即用、针对游戏内固定玩法的成熟方案（识别 + 调参即可）。容器是自定义编排：组合任意节点实现新流程（如自定义副本路径、签到流、特殊事件响应）。预设达不到你需要的 → 用容器。',
  },
  {
    q: '为什么按下"开始"没反应？',
    a: '请先确认侧栏底部"游戏窗口"显示绿色"已检测"。若未检测，进入设置页点"重新检测"。游戏需以前台主界面运行。',
  },
  {
    q: '容器 试运行 报 EMPTY_SUBGRAPH_OUTPUT 怎么办？',
    a: '子图至少要有 1 个 SubgraphOutput 节点。新建子图后端会自动预填一个，但旧子图（v2 早期建的）可能缺。重新打开容器编辑器即可触发自动 self-heal 补上；或手动从 palette 拖入 SubgraphOutput。',
  },
  {
    q: '从库拖入子图后弹"来自另一台机器"是什么意思？',
    a: '子图在源机器录制时会写入 360° HID counts（源 mouseCounts360）。本机 counts 与源不同时，MouseMoveRel 的相对位移会按 target/source 缩放。弹窗让你选：把本机值同步到所有容器 / 仅改本容器 / 不改（按源值跑，可能动作幅度不对）。',
  },
  {
    q: 'MouseCalibration 节点旁边有黄色 FOREIGN 警告？',
    a: '节点 config.counts360 与本机设置不同。点警告里的"同步到所有容器"按钮一键覆盖；或进入设置 → 输入校正重新校准本机值。',
  },
  {
    q: '容器编辑器是独立窗口，关掉怎么不回 dashboard？',
    a: '设计如此：每个容器一个独立 Frameless 窗口，主 dashboard 在另一窗口里继续显示。关闭编辑器窗口 = "返回"。同一容器二次"编辑"会 focus 已有窗口而非新建。',
  },
  {
    q: '工具运行中游戏窗口失去焦点会怎样？',
    a: '钓鱼、音游、战斗：用 PostMessage + 模拟激活，YHBox 抢前台时游戏仍能收到按键；店长：依赖前台截屏，失焦会停。容器内可用 BringGameForeground 节点强制拉前台。',
  },
  {
    q: '战斗热键没生效？',
    a: '热键被其它应用（如截图工具、输入法）占用是常见原因。改用其它修饰键组合（如 Ctrl+Alt）或关掉冲突应用。',
  },
  {
    q: '日志文件存在哪里？',
    a: '在 YHBox.exe 同目录的 logs/ 下，按功能名和启动时间命名。JSON Lines 格式，可用 jq 解析。',
  },
]

const troubleshoot = [
  {
    symptom: 'YHBox 启动时没弹出 UAC 提权窗',
    fix: '右键 YHBox.exe → "以管理员身份运行"。游戏注入输入需要管理员权限，否则 YHBox 无法工作。',
  },
  {
    symptom: '日志疯狂报 MISS / 抓帧失败',
    fix: '分辨率与模板不匹配。预设 Bot 均已校准 1920×1080 与 1280×720；其它分辨率识别率低，UI 缩放需 100%。容器系统模板可自己重录。',
  },
  {
    symptom: '钓鱼运行一会儿就卡住不动',
    fix: '可能因为游戏弹了对话框、广告或事件提示遮住 ROI。手动关掉弹窗，工具会自动恢复。',
  },
  {
    symptom: '容器 "未保存" 标记一直亮，怎么也保存不掉',
    fix: '某个子图后端保存失败（toast 应该会列出失败 sg id）。可能是磁盘权限或文件被占用。检查 bin/data/containers/<容器 id>/subgraphs/ 目录是否可写。',
  },
  {
    symptom: '容器编辑器框选 / 节点连线不工作',
    fix: 'vue-flow 版本相关。当前已通过 selection-key-code=true 和 activeGraph 统一写入点修复。若复现请检查浏览器 console 是否有 vue-flow 报错。',
  },
  {
    symptom: '弹琴音域明显错乱',
    fix: '关掉"自动对齐音域"，手动调"八度微调"。某些复杂 MIDI 智能选轨会挑到伴奏轨，可改"选轨"为具体的 track 编号。',
  },
]
</script>
