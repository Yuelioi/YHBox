// 抄自 Vue 官方脚手架最新推荐布局：
//   - oxlint 跑常见 correctness 规则（快）
//   - eslint + eslint-plugin-vue + @vue/eslint-config-typescript 跑 Vue/TS 专有 rules
//   - eslint-config-prettier 关掉所有跟格式化相关的 rule（格式化交给 oxfmt）
// 命令：
//   npm run lint       → oxlint → eslint，依次跑（都 --fix）
//   npm run format     → oxfmt 改文件
//   npm run type-check → vue-tsc --noEmit
import { globalIgnores } from 'eslint/config'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'
import pluginVue from 'eslint-plugin-vue'
import pluginOxlint from 'eslint-plugin-oxlint'
import skipFormatting from 'eslint-config-prettier/flat'

export default defineConfigWithVueTs(
  {
    name: 'app/files-to-lint',
    files: ['**/*.{vue,ts,mts,tsx}'],
  },

  globalIgnores([
    '**/dist/**',
    '**/dist-ssr/**',
    '**/bindings/**', // wails3 生成
    '**/node_modules/**',
  ]),

  ...pluginVue.configs['flat/essential'],
  vueTsConfigs.recommended,

  ...pluginOxlint.buildFromOxlintConfigFile('.oxlintrc.json'),

  skipFormatting,
)
