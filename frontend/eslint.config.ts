// 抄自 Vue 官方脚手架最新推荐布局：
//   - oxlint 跑常见 correctness 规则（快）
//   - eslint + eslint-plugin-vue + @vue/eslint-config-typescript 跑 Vue/TS 专有 rules
//   - eslint-config-prettier 关掉所有跟格式化相关的 rule（格式化交给 oxfmt）
// 命令：
//   pnpm lint      → oxlint → eslint，依次做 check-only
//   pnpm lint:fix  → 明确请求时才自动修复
//   pnpm format    → oxfmt 改文件
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

  {
    name: 'app/typescript-baseline',
    rules: {
      // Counted by scripts/check-eslint.mjs so existing debt cannot grow unnoticed.
      '@typescript-eslint/no-explicit-any': 'warn',
    },
  },

  {
    name: 'app/commonjs-scripts',
    files: ['**/*.cjs'],
    rules: {
      '@typescript-eslint/no-require-imports': 'off',
    },
  },

  ...pluginOxlint.buildFromOxlintConfigFile('.oxlintrc.json'),

  skipFormatting,
)
