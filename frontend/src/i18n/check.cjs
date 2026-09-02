// node frontend/src/i18n/check.cjs  或  npm run i18n:check
//
// 五件事:
//   1. zh.ts / en.ts 键集合对齐 — 缺则报 missing
//   2. [compile] 每条 message 真过一遍 vue-i18n message compiler — 抓
//      `||` / 裸 `{` `}` `@` 这类没 escape 的特殊字符 (parity 只对 key 不 compile,
//      跑起来才炸整个组件; 见 incident vue-i18n-message-compiler-traps)
//   3. scan frontend/src/**/*.{vue,ts} 残留中文字面值 (排除 .i18n-ignore + i18n/ 自身)
//   4. scan static t('key') / te('key') references and fail on missing keys
//   5. 任一项失败 → exit 1
//
// 用 jiti 加载 ESM/TS module (zh.ts/en.ts 是 `export default {...}`).

const fs = require('fs')
const path = require('path')
const { createJiti } = require('jiti')

const ROOT_I18N = __dirname
const ROOT_SRC = path.join(__dirname, '..')
const ROOT_FE = path.join(__dirname, '..', '..')
const NODE_PROJECTION = path.join(
  ROOT_FE,
  '..',
  'contracts',
  'node',
  'current',
  'builtin-authoring.json',
)

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

function loadMessages(file) {
  const mod = jiti(path.join(ROOT_I18N, file))
  return mod && typeof mod === 'object' && mod.default ? mod.default : mod
}

function checkKeyParity() {
  const zh = flatten(loadMessages('zh.ts'))
  const en = flatten(loadMessages('en.ts'))
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

// 每条 message 真过一遍 vue-i18n compiler。vue-i18n full build 编译失败时是
// console.error 出 "Message compilation error: ..." 再 fallback (不 throw), 所以逐 key
// t() 时捕获 console 判定。只需 vue-i18n (frontend 直接依赖, pnpm 下可解析)。
function checkMessageCompile() {
  let createI18n
  try {
    const vi = jiti('vue-i18n')
    createI18n = vi.createI18n || (vi.default && vi.default.createI18n)
  } catch (e) {
    console.error('[compile] FAIL 无法加载 vue-i18n: ' + e.message)
    return false
  }
  if (typeof createI18n !== 'function') {
    console.error('[compile] FAIL vue-i18n.createI18n 不可用')
    return false
  }

  const fails = []
  for (const [loc, file] of [
    ['zh', 'zh.ts'],
    ['en', 'en.ts'],
  ]) {
    const msgs = loadMessages(file)
    const flat = flatten(msgs)
    const i18n = createI18n({
      legacy: false,
      locale: loc,
      fallbackLocale: false,
      messages: { [loc]: msgs },
      missingWarn: false,
      fallbackWarn: false,
      warnHtmlMessage: false,
    })
    const t = i18n.global.t
    const origErr = console.error
    const origWarn = console.warn
    for (const key of Object.keys(flat)) {
      if (typeof flat[key] !== 'string') continue
      let err = null
      const cap = (...a) => {
        const s = a.map(String).join(' ')
        if (/compil/i.test(s)) err = s
      }
      console.error = cap
      console.warn = cap
      try {
        t(key)
      } catch (e) {
        err = e.message
      } finally {
        console.error = origErr
        console.warn = origWarn
      }
      if (err) fails.push(`[${loc}] ${key} :: ${String(err).split('\n')[0]}`)
    }
  }
  if (fails.length === 0) {
    console.log('[compile] OK 全部 message 编译通过')
    return true
  }
  console.error(
    `[compile] FAIL ${fails.length} 处 (vue-i18n message compiler — 含特殊字符要 {'literal'} escape):`,
  )
  fails.forEach((f) => console.error('  ' + f))
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

function sourceFilesForChecks() {
  const ignores = loadIgnore()
  const i18nDir = path.normalize(ROOT_I18N) + path.sep
  return walkSrc(ROOT_SRC, [])
    .filter((f) => !f.startsWith(i18nDir))
    .filter((f) => {
      const rel = path.relative(ROOT_FE, f).replace(/\\/g, '/')
      return !ignores.some((re) => re.test(rel))
    })
}

function checkStaticKeyReferences() {
  const zh = flatten(loadMessages('zh.ts'))
  const en = flatten(loadMessages('en.ts'))
  const zhKeys = new Set(Object.keys(zh))
  const enKeys = new Set(Object.keys(en))
  const hits = []
  const callRe = /(?<![\w$])(?:t|te)\(\s*(['"])([a-zA-Z0-9_.-]+)\1/g

  for (const f of sourceFilesForChecks()) {
    const rel = path.relative(ROOT_FE, f).replace(/\\/g, '/')
    const text = fs.readFileSync(f, 'utf8')
    const lineStarts = [0]
    for (let i = 0; i < text.length; i++) {
      if (text.charCodeAt(i) === 10) lineStarts.push(i + 1)
    }
    for (const match of text.matchAll(callRe)) {
      const key = match[2]
      if (zhKeys.has(key) && enKeys.has(key)) continue
      const line = lineStarts.findLastIndex((start) => start <= match.index) + 1
      const missing = [zhKeys.has(key) ? null : 'zh', enKeys.has(key) ? null : 'en']
        .filter(Boolean)
        .join('/')
      hits.push({ file: rel, line, key, missing })
    }
  }

  if (hits.length === 0) {
    console.log('[refs] OK static i18n key references resolve')
    return true
  }
  console.error(`[refs] FAIL ${hits.length} static i18n references missing keys:`)
  hits.forEach((h) => console.error(`  ${h.file}:${h.line} ${h.key} missing ${h.missing}`))
  return false
}

function checkProjectedNodeErrorMessages() {
  let projection
  try {
    projection = JSON.parse(fs.readFileSync(NODE_PROJECTION, 'utf8'))
  } catch (error) {
    console.error('[node-errors] FAIL 无法读取当前节点投影: ' + error.message)
    return false
  }
  const nodes = projection?.body?.nodes
  if (!Array.isArray(nodes)) {
    console.error('[node-errors] FAIL 当前节点投影缺少 body.nodes')
    return false
  }
  const zh = flatten(loadMessages('zh.ts'))
  const en = flatten(loadMessages('en.ts'))
  const codes = [
    ...new Set(
      nodes.flatMap((node) =>
        Array.isArray(node.errors) ? node.errors.map((item) => item.code) : [],
      ),
    ),
  ].sort()
  const missing = []
  for (const code of codes) {
    const key = `error.${code}`
    const locales = [
      typeof zh[key] === 'string' ? null : 'zh',
      typeof en[key] === 'string' ? null : 'en',
    ]
      .filter(Boolean)
      .join('/')
    if (locales) missing.push({ key, locales })
  }
  if (missing.length === 0) {
    console.log(`[node-errors] OK ${codes.length} projected error codes have zh/en messages`)
    return true
  }
  console.error(`[node-errors] FAIL ${missing.length} projected error codes are missing messages:`)
  missing.forEach((item) => console.error(`  - ${item.key} missing ${item.locales}`))
  return false
}

function checkProjectedNodeOutcomes() {
  let projection
  try {
    projection = JSON.parse(fs.readFileSync(NODE_PROJECTION, 'utf8'))
  } catch (error) {
    console.error('[node-outcomes] FAIL 无法读取当前节点投影: ' + error.message)
    return false
  }
  const failures = []
  for (const node of projection?.body?.nodes ?? []) {
    const outcomes = (node.signals ?? [])
      .filter(
        (signal) =>
          signal.channel === 'exec' &&
          signal.direction === 'output' &&
          ['timeout', 'exhausted'].includes(signal.id),
      )
      .map((signal) => signal.id)
    const statuses = node.statusEvents ?? []
    for (const outcome of outcomes) {
      if (!statuses.some((status) => status.code.endsWith(`.${outcome}`))) {
        failures.push(`${node.nodeRef.nodeTypeId} output ${outcome} lacks terminal status evidence`)
      }
      if (outcome === 'timeout' && !statuses.some((status) => status.category === 'waiting')) {
        failures.push(`${node.nodeRef.nodeTypeId} timeout lacks waiting status evidence`)
      }
    }
  }
  if (failures.length === 0) {
    console.log('[node-outcomes] OK timeout/exhausted outputs declare status evidence')
    return true
  }
  console.error(`[node-outcomes] FAIL ${failures.length} outcome contract gaps:`)
  failures.forEach((failure) => console.error('  - ' + failure))
  return false
}

const parityOK = checkKeyParity()
const compileOK = checkMessageCompile()
const residueOK = checkResidueLiterals()
const refsOK = checkStaticKeyReferences()
const nodeErrorsOK = checkProjectedNodeErrorMessages()
const nodeOutcomesOK = checkProjectedNodeOutcomes()
process.exit(parityOK && compileOK && residueOK && refsOK && nodeErrorsOK && nodeOutcomesOK ? 0 : 1)
