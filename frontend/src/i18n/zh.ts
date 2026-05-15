// 中文文案。键命名约定：
//   sidebar.<bot>       侧栏每个项的 label
//   controls.<state>    BotControls 按钮 / 状态
//   view.<bot>.<key>    各 view 内部文案
//   settings.<key>      设置面板
//   common.<key>        共用通用文案
//
// 用 ts 而不是 yaml 是因为 unplugin-vue-i18n 的 yaml 输出格式跟 vue-i18n 9
// runtime 不兼容会让 t() 触发 SyntaxError. 直接 ts 给 createI18n 喂 plain
// string，runtime 自己 compile，所有 unicode 字符都 OK.

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
  },
  common: {
    game_not_detected: '未检测到异环窗口',
  },
}
