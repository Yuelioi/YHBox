// 中文 locale 组合入口；功能模块归属见 README.md（en.ts 是平行翻译）。
import aboutMessages from './locales/zh/about'
import appMessages from './locales/zh/app'
import nodeMessages from './locales/zh/node'
import recordingMessages from './locales/zh/recording'
import resourcesMessages from './locales/zh/resources'
import scheduleMessages from './locales/zh/schedule'
import settingsMessages from './locales/zh/settings'
import toolsMessages from './locales/zh/tools'
import workflowMessages from './locales/zh/workflow'

export default {
  ...appMessages,
  ...nodeMessages,
  ...workflowMessages,
  ...scheduleMessages,
  ...resourcesMessages,
  ...recordingMessages,
  ...settingsMessages,
  ...toolsMessages,
  ...aboutMessages,
}
