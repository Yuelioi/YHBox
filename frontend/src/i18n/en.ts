// English locale composition root. See README.md for feature-module ownership.
import aboutMessages from './locales/en/about'
import appMessages from './locales/en/app'
import nodeMessages from './locales/en/node'
import recordingMessages from './locales/en/recording'
import resourcesMessages from './locales/en/resources'
import scheduleMessages from './locales/en/schedule'
import settingsMessages from './locales/en/settings'
import toolsMessages from './locales/en/tools'
import workflowMessages from './locales/en/workflow'

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
