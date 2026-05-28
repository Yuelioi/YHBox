// node frontend/src/i18n/check.cjs  或  npm run i18n:check
//
// 三件事:
//   1. zh.ts / en.ts 键集合对齐 — 缺则报 missing
//   2. scan frontend/src/**/*.{vue,ts} 残留中文字面值 (排除 .i18n-ignore + i18n/ 自身)
//   3. 任一项失败 → exit 1
//
// 用 jiti 加载 ESM/TS module (zh.ts/en.ts 是 `export default {...}`).

const fs = require('fs')
const path = require('path')
const { createJiti } = require('jiti')

const ROOT_I18N = __dirname
const ROOT_SRC = path.join(__dirname, '..')
const ROOT_FE = path.join(__dirname, '..', '..')

const jiti = createJiti(__filename, { interopDefault: true })

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

function checkKeyParity() {
  const zh = flatten(jiti(path.join(ROOT_I18N, 'zh.ts')))
  const en = flatten(jiti(path.join(ROOT_I18N, 'en.ts')))
  const zhKeys = new Set(Object.keys(zh))
  const enKeys = new Set(Object.keys(en))
  const missingInEn = [...zhKeys].filter((k) => !enKeys.has(k))
  const missingInZh = [...enKeys].filter((k) => !zhKeys.has(k))
  if (missingInEn.length === 0 && missingInZh.length === 0) {
    console.log(`[parity] OK ${zhKeys.size} keys`)
    return true
  }
  if (missingInEn.length) {
    console.error(`[parity] FAIL en.ts 缺 ${missingInEn.length} 键:`)
    missingInEn.forEach((k) => console.error('  - ' + k))
  }
  if (missingInZh.length) {
    console.error(`[parity] FAIL zh.ts 缺 ${missingInZh.length} 键:`)
    missingInZh.forEach((k) => console.error('  - ' + k))
  }
  return false
}

// minimatch-style glob → regex (只支持本脚本用的 `**`, `*`, 精确路径)
function globToRegex(glob) {
  const re = glob
    .replace(/[.+^${}()|[\]\\]/g, '\\$&')
    .replace(/\*\*/g, '__GLOBSTAR__')
    .replace(/\*/g, '[^/]*')
    .replace(/__GLOBSTAR__/g, '.*')
  return new RegExp('^' + re + '$')
}

function loadIgnore() {
  const p = path.join(ROOT_I18N, '.i18n-ignore')
  if (!fs.existsSync(p)) return []
  return fs
    .readFileSync(p, 'utf8')
    .split('\n')
    .map((l) => l.trim())
    .filter((l) => l && !l.startsWith('#'))
    .map(globToRegex)
}

function walkSrc(dir, out) {
  for (const ent of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, ent.name)
    if (ent.isDirectory()) {
      walkSrc(full, out)
    } else if (ent.isFile() && /\.(vue|ts)$/.test(ent.name)) {
      out.push(full)
    }
  }
  return out
}

function checkResidueLiterals() {
  const ignores = loadIgnore()
  const i18nDir = path.normalize(ROOT_I18N) + path.sep
  const files = walkSrc(ROOT_SRC, []).filter((f) => !f.startsWith(i18nDir))
  const hits = []
  for (const f of files) {
    const rel = path.relative(ROOT_FE, f).replace(/\\/g, '/')
    if (ignores.some((re) => re.test(rel))) continue
    const text = fs.readFileSync(f, 'utf8')
    // 剥 // 行注释 + /* */ 块注释 + <!-- --> HTML/Vue 注释
    // (粗略, 不处理字符串内 // 的边角; 注释跨行用空格 padding 保行号)
    const stripped = text
      .replace(/<!--[\s\S]*?-->/g, (m) => m.replace(/[^\n]/g, ' '))
      .replace(/\/\*[\s\S]*?\*\//g, (m) => m.replace(/[^\n]/g, ' '))
      .replace(/\/\/[^\n]*/g, (m) => m.replace(/[^\n]/g, ' '))
    const lines = stripped.split('\n')
    lines.forEach((line, i) => {
      const m = line.match(/[一-鿿][^\n]*/)
      if (m) hits.push({ file: rel, line: i + 1, snippet: m[0].slice(0, 80) })
    })
  }
  if (hits.length === 0) {
    console.log(`[residue] OK 0 residue 中文字面值`)
    return true
  }
  console.error(`[residue] FAIL ${hits.length} 处:`)
  hits.forEach((h) => console.error(`  ${h.file}:${h.line}  ${h.snippet}`))
  return false
}

const parityOK = checkKeyParity()
const residueOK = checkResidueLiterals()
process.exit(parityOK && residueOK ? 0 : 1)
