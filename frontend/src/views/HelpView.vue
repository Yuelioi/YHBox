<template>
  <div class="px-8 py-6 space-y-6">
    <!-- 快速上手 -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-3">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-rocket" class="size-4 text-primary" />
        <h2 class="text-sm font-medium text-highlighted">快速上手</h2>
      </div>
      <p class="text-xs text-muted">
        YHBox 是通用游戏自动化脚本框架. 工作流: 建容器 → 编排节点图 → 触发执行.
      </p>
      <ol class="space-y-1 text-xs text-toned list-decimal pl-5 marker:text-dimmed">
        <li>侧栏点 <span class="text-highlighted">容器</span> → 新建一个</li>
        <li>编辑器拖节点 / 连线 / 配置参数 (左侧 palette, 右侧 inspector)</li>
        <li>点 <span class="text-primary">试运行</span> 立即跑一次, 或在 <span class="text-highlighted">计划</span> 绑热键 / 定时</li>
        <li>可复用片段 → 折叠为子图 → 发布到库, 跨容器共享</li>
      </ol>
    </section>

    <!-- 核心概念 -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-bulb" class="size-4 text-amber-300" />
        <h2 class="text-sm font-medium text-highlighted">核心概念</h2>
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
    desc: '自包含的可执行编排单元. 含一张主图 (节点 + 边)、所属子图集合、变量、热键. 导出后可整体分享给其他机器用户.',
  },
  {
    name: '子图 (Subgraph)',
    icon: 'i-tabler-schema',
    iconClass: 'text-fuchsia-300',
    desc: '容器内可复用的执行函数: 有一个入口 (Entry)、一个或多个命名出口. 父图通过 Subgraph 调用节点执行子图, 按出口名分流.',
  },
  {
    name: '库 (Library)',
    icon: 'i-tabler-books',
    iconClass: 'text-emerald-300',
    desc: '全机器共享的子图 + 模板仓库. 从库拖入容器走 copy-on-use (复制独立副本, 互不影响). 可把容器子图反向"发布到库"分享.',
  },
  {
    name: '计划任务 (Schedule)',
    iconClass: 'text-amber-300',
    icon: 'i-tabler-calendar-clock',
    desc: '安排容器自动执行: 热键触发 / 定时 (cron / 间隔) / 一次性 / 手动. 多个容器可挂同一计划, 按顺序串行执行.',
  },
]

const nodeGroups = [
  { label: '控制流', desc: 'Start / Sleep / Loop / If / Stop / Break / Continue' },
  { label: '变量', desc: 'SetVar / IncVar — 容器作用域 + 子图局部作用域 (runtime 自动隔离)' },
  { label: '图像', desc: 'WaitTemplate / CheckTemplate / ClickTemplate / DetectColor — 模板匹配与颜色检测' },
  { label: '输入', desc: 'ClickAt / KeyPress / MouseMoveRel / Scroll — 注入输入到目标窗口' },
  { label: '事件', desc: 'OnEvent — listener 形态入口, 匹配触发即 spawn 子图执行' },
  { label: '子图', desc: 'Subgraph 调用 / SubgraphInput 入口 / SubgraphOutput 出口' },
  { label: '系统', desc: 'WindowTarget — 选定目标窗口 (title / class / processName 任意组合匹配)' },
  { label: '配置', desc: 'MouseCalibration — 标定本机 360° HID counts, 给 MouseMoveRel 缩放用' },
  { label: '调试', desc: 'Log / Toast' },
]

const faq = [
  {
    q: '容器跑哪个目标窗口?',
    a: '在容器图最上游加一个 WindowTarget 节点, 用 title / class / processName 匹配目标. 后续 ClickAt / KeyPress 等输入节点会注入到该窗口. 不限游戏 — 任意 Windows 进程都行.',
  },
  {
    q: '为什么按下"试运行"没反应?',
    a: '看日志面板有没有报错. 常见: (a) WindowTarget 匹配不到 (窗口标题/进程名拼错); (b) 模板节点 ROI 不匹配 (目标窗口分辨率变了); (c) 容器有未保存修改 → 先点"保存".',
  },
  {
    q: '容器 试运行 报 EMPTY_SUBGRAPH_OUTPUT 怎么办?',
    a: '子图至少要有 1 个 SubgraphOutput 节点. 新建子图后端会自动预填一个, 但旧子图 (v2 早期建的) 可能缺. 重新打开容器编辑器即可触发自动 self-heal 补上; 或手动从 palette 拖入 SubgraphOutput.',
  },
  {
    q: '从库拖入子图后弹"来自另一台机器"是什么意思?',
    a: '子图在源机器录制时会写入 360° HID counts (源 mouseCounts360). 本机 counts 与源不同时, MouseMoveRel 的相对位移会按 target/source 缩放. 弹窗让你选: 把本机值同步到所有容器 / 仅改本容器 / 不改 (按源值跑, 可能动作幅度不对).',
  },
  {
    q: 'MouseCalibration 节点旁边有黄色 FOREIGN 警告?',
    a: '节点 config.counts360 与本机设置不同. 点警告里的"同步到所有容器"按钮一键覆盖; 或进入设置 → 输入校正重新校准本机值.',
  },
  {
    q: '切到设置/帮助后再切回"容器" 怎么回到正在编辑的容器?',
    a: '侧栏"容器" tab 会自动跳回上次编辑的容器 (keep-alive 保留 draft + canvas 状态). 想回容器列表 → 编辑器工具栏左上"←" 按钮.',
  },
  {
    q: '日志文件存在哪里?',
    a: '在 YHBox.exe 同目录的 logs/ 下. JSON Lines 格式, 可用 jq 解析. 设置 → 通用 → 日志 可关闭文件写入.',
  },
]

const troubleshoot = [
  {
    symptom: 'YHBox 启动时没弹出 UAC 提权窗',
    fix: '右键 YHBox.exe → "以管理员身份运行". 注入输入到游戏需要管理员权限, 否则无法工作.',
  },
  {
    symptom: '模板节点疯狂 MISS',
    fix: '分辨率与模板录制时不匹配. 重新录制模板 (容器编辑器 → 模板节点 inspector → 重录), 或在录制时把目标窗口拉到模板原始分辨率.',
  },
  {
    symptom: '容器跑一会儿就卡住不动',
    fix: '目标窗口弹了对话框或事件提示遮住 ROI. 手动关掉, 容器会自动恢复. 想让脚本绕过 → 加 OnEvent listener 检测弹窗特征 + 自动关闭子图.',
  },
  {
    symptom: '容器"未保存"标记一直亮, 怎么也保存不掉',
    fix: '某个子图后端保存失败 (toast 应该会列出失败 sg id). 可能是磁盘权限或文件被占用. 检查 bin/data/containers/<容器 id>/subgraphs/ 目录是否可写.',
  },
  {
    symptom: '容器编辑器框选 / 节点连线不工作',
    fix: 'vue-flow 版本相关. 复现请检查浏览器 console 是否有 vue-flow 报错并附 console 截图.',
  },
]
</script>
