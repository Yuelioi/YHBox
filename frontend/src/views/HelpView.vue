<template>
  <div class="px-8 py-6 space-y-6">
    <!-- 快速上手 -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-3">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-rocket" class="size-4 text-primary" />
        <h2 class="text-sm font-medium text-highlighted">快速上手</h2>
      </div>
      <ol class="space-y-2 text-sm text-toned list-decimal pl-5 marker:text-dimmed">
        <li>
          启动
          <span class="text-highlighted font-medium">异环</span>
          游戏，等待进入主界面
        </li>
        <li>
          YHBox 侧栏底部应该显示
          <span class="text-primary font-medium">已检测</span>。未检测可点"重新检测"
        </li>
        <li>切到对应功能标签（钓鱼 / 店长 / 弹琴 / 战斗 / 音游）</li>
        <li>
          调整配置（如钓鱼的自动出售、店长的点击间隔），然后点
          <span class="text-primary font-medium">开始</span>
        </li>
        <li>底部日志面板会实时显示运行状态</li>
      </ol>
    </section>

    <!-- 功能简述 -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-list-check" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">功能模块</h2>
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
]

const faq = [
  {
    q: '为什么按下"开始"没反应？',
    a: '请先确认侧栏底部"游戏窗口"显示绿色"已检测"。若未检测，进入设置页点"重新检测"。游戏需以前台主界面运行。',
  },
  {
    q: '工具运行中游戏窗口失去焦点会怎样？',
    a: '钓鱼、音游、战斗：用 PostMessage + 模拟激活，YHBox 抢前台时游戏仍能收到按键；店长：依赖前台截屏，失焦会停。如果异环放副屏或始终置顶最稳。',
  },
  {
    q: '点了"停止"为什么还要几秒才真停？',
    a: '已优化为接近即时（< 50 ms）。停止时会清理键盘状态、关闭日志文件、释放热键，期间按钮显示"停止中..."。',
  },
  {
    q: '战斗热键没生效？',
    a: '热键被其它应用（如截图工具、输入法）占用是常见原因。改用其它修饰键组合（如 Ctrl+Alt）或关掉冲突应用。',
  },
  {
    q: '日志文件存在哪里？',
    a: '在 YHBox.exe 同目录的 logs/ 下，按功能名和启动时间命名（如 yh_fish_20260513_180312.log）。JSON Lines 格式，可用 jq 解析。',
  },
]

const troubleshoot = [
  {
    symptom: 'YHBox 启动时没弹出 UAC 提权窗',
    fix: '右键 YHBox.exe → "以管理员身份运行"。游戏注入输入需要管理员权限，否则 YHBox 无法工作。',
  },
  {
    symptom: '日志疯狂报 MISS / 抓帧失败',
    fix: '分辨率与模板不匹配。当前所有功能均已校准 1920×1080 与 1280×720；其它分辨率可能识别率低，UI 缩放需 100%。',
  },
  {
    symptom: '钓鱼运行一会儿就卡住不动',
    fix: '可能因为游戏弹了对话框、广告或事件提示遮住 ROI。手动关掉弹窗，工具会自动恢复。',
  },
  {
    symptom: '弹琴音域明显错乱',
    fix: '关掉"自动对齐音域"，手动调"八度微调"。某些复杂 MIDI 智能选轨会挑到伴奏轨，可改"选轨"为具体的 track 编号。',
  },
]
</script>
