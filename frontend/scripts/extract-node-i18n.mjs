// 从 frontend/src/i18n/zh.ts 的 node.* 块抽出节点展示文案, 生成 Go 端 go:embed 的
// internal/catalog/node-i18n.json。catalog.BuildWithI18n() 按 kind JOIN 进节点目录,
// 供 yotta-mcp 的 list_nodes 给 LLM 提供 label/description/pin hint。
//
// 用法:  pnpm gen:node-i18n   (npm script 先用 esbuild 把 zh.ts 打成 ESM, 再跑本脚本)
// zh.ts 是 export default 纯对象字面量, 走 esbuild AST 打包而非正则 (zh.ts 改结构这里要重跑;
// Go 端 TestBuildWithI18n_AllKindsLabeled 会在漏 kind 时 fail, 逼着同步)。
import { readFileSync, writeFileSync } from 'node:fs'
import { pathToFileURL } from 'node:url'
import { resolve } from 'node:path'

const COMPILED = resolve('node_modules/.cache/zh-i18n.mjs')
const OUT = resolve('../internal/catalog/node-i18n.json')

const mod = await import(pathToFileURL(COMPILED).href)
const node = mod.default?.node
if (!node || typeof node !== 'object') {
  console.error('FATAL: zh.ts 顶层 node 块缺失或非对象')
  process.exit(1)
}

function pinMap(obj, withHint) {
  if (!obj) return undefined
  const out = {}
  for (const [pin, v] of Object.entries(obj)) {
    if (!v || typeof v !== 'object') continue
    const e = {}
    if (typeof v.label === 'string') e.label = v.label
    if (withHint && typeof v.hint === 'string') e.hint = v.hint
    if (withHint && v.option && typeof v.option === 'object' && !Array.isArray(v.option)) {
      const option = {}
      for (const [key, label] of Object.entries(v.option)) {
        if (typeof label === 'string') option[key] = label
      }
      if (Object.keys(option).length) e.option = option
    }
    if (Object.keys(e).length) out[pin] = e
  }
  return Object.keys(out).length ? out : undefined
}

const result = {}
for (const [kind, v] of Object.entries(node)) {
  if (!v || typeof v !== 'object') continue
  const e = {}
  if (typeof v.label === 'string') e.label = v.label
  if (typeof v.description === 'string') e.description = v.description
  if (typeof v.example === 'string') e.example = v.example
  const ins = pinMap(v.input, true)   // input pin: label + hint + dropdown option labels
  const outs = pinMap(v.output, false) // output pin: label only
  if (ins) e.input = ins
  if (outs) e.output = outs
  result[kind] = e
}

// 稳定排序 (按 kind), 让生成物 diff 干净
const sorted = {}
for (const k of Object.keys(result).sort()) sorted[k] = result[k]

writeFileSync(OUT, JSON.stringify(sorted, null, 2) + '\n')
console.log(`node-i18n.json 已生成: ${Object.keys(sorted).length} 个节点 → ${OUT}`)
