// 用法：node frontend/src/i18n/check.cjs  或  npm run i18n:check
// 功能：diff zh.yaml 和 en.yaml 的键集，列出缺失键。

const fs = require('fs')
const path = require('path')
const yaml = require('js-yaml')

function flatten(obj, prefix = '') {
  const out = {}
  for (const k of Object.keys(obj)) {
    const key = prefix ? `${prefix}.${k}` : k
    if (obj[k] && typeof obj[k] === 'object' && !Array.isArray(obj[k])) {
      Object.assign(out, flatten(obj[k], key))
    } else {
      out[key] = obj[k]
    }
  }
  return out
}

const zhPath = path.join(__dirname, 'zh.yaml')
const enPath = path.join(__dirname, 'en.yaml')

const zh = flatten(yaml.load(fs.readFileSync(zhPath, 'utf8')))
const en = flatten(yaml.load(fs.readFileSync(enPath, 'utf8')))

const zhKeys = new Set(Object.keys(zh))
const enKeys = new Set(Object.keys(en))

const missingInEn = [...zhKeys].filter((k) => !enKeys.has(k))
const missingInZh = [...enKeys].filter((k) => !zhKeys.has(k))

if (missingInEn.length === 0 && missingInZh.length === 0) {
  console.log('OK zh.yaml 和 en.yaml 键完全一致 (' + zhKeys.size + ' keys)')
  process.exit(0)
}
if (missingInEn.length > 0) {
  console.error('FAIL en.yaml 缺少的键 (' + missingInEn.length + ')：')
  missingInEn.forEach((k) => console.error('  - ' + k))
}
if (missingInZh.length > 0) {
  console.error('FAIL zh.yaml 缺少的键 (' + missingInZh.length + ')：')
  missingInZh.forEach((k) => console.error('  - ' + k))
}
process.exit(1)
