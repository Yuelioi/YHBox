import { spawnSync } from 'node:child_process'
import { existsSync, lstatSync, readdirSync, readFileSync } from 'node:fs'
import { dirname, extname, relative, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const skippedDirectories = new Set([
  '.git',
  '.task',
  'artifacts',
  'bin',
  'dist',
  'node_modules',
  'third_party',
])

const normalize = (value) => value.split(sep).join('/')

function shouldSkipDirectory(path) {
  const name = path.split(sep).at(-1)
  if (skippedDirectories.has(name)) return true
  const local = normalize(relative(repositoryRoot, path))
  return local === 'flightdeck/work' || local === 'contracts/node'
}

export function markdownFiles(root = repositoryRoot) {
  const result = []
  const visit = (directory) => {
    for (const entry of readdirSync(directory, { withFileTypes: true })) {
      const path = resolve(directory, entry.name)
      if (entry.isDirectory()) {
        if (!shouldSkipDirectory(path)) visit(path)
      } else if (entry.isFile() && extname(entry.name).toLowerCase() === '.md') {
        result.push(path)
      }
    }
  }
  visit(root)
  return result.sort()
}

export function localLinkTargets(markdown) {
  const result = []
  const pattern = /!?\[[^\]]*\]\(\s*(?:<([^>]+)>|([^\s)]+))(?:\s+[^)]*)?\)/g
  for (const match of markdown.matchAll(pattern)) {
    const target = (match[1] ?? match[2] ?? '').trim()
    if (target) result.push(target)
  }
  return result
}

export function taskNames(markdown) {
  return [...markdown.matchAll(/(?:^|`)task\s+([a-z0-9][a-z0-9:_-]*)/gm)].map(
    (match) => match[1],
  )
}

export function isExternalTarget(target) {
  return target.startsWith('//') || /^[a-z][a-z\d+.-]*:/i.test(target)
}

function taskRegistry() {
  const result = spawnSync('task', ['--list-all', '--json'], {
    cwd: repositoryRoot,
    encoding: 'utf8',
  })
  if (result.error) throw result.error
  if (result.status !== 0) {
    throw new Error(`task --list-all --json failed: ${(result.stderr || result.stdout).trim()}`)
  }
  const parsed = JSON.parse(result.stdout)
  return new Set(parsed.tasks.map((entry) => entry.task))
}

function resolveLocalLink(markdownPath, target) {
  const withoutFragment = target.split('#', 1)[0].split('?', 1)[0]
  if (!withoutFragment || isExternalTarget(withoutFragment)) return null
  let decoded
  try {
    decoded = decodeURIComponent(withoutFragment)
  } catch {
    return { error: `invalid URL encoding in link ${JSON.stringify(target)}` }
  }
  const path = decoded.startsWith('/')
    ? resolve(repositoryRoot, `.${decoded}`)
    : resolve(dirname(markdownPath), decoded)
  const local = relative(repositoryRoot, path)
  if (local === '..' || local.startsWith(`..${sep}`)) {
    return { error: `local link escapes the repository: ${JSON.stringify(target)}` }
  }
  if (!existsSync(path)) return { error: `missing local link target ${JSON.stringify(target)}` }
  if (lstatSync(path).isSymbolicLink()) {
    return { error: `local documentation link must not rely on a symlink: ${JSON.stringify(target)}` }
  }
  return null
}

export function checkMarkdown(markdownPath, markdown, knownTasks) {
  const errors = []
  for (const target of localLinkTargets(markdown)) {
    const issue = resolveLocalLink(markdownPath, target)
    if (issue) errors.push(issue.error)
  }
  for (const name of taskNames(markdown)) {
    if (!knownTasks.has(name)) errors.push(`unknown Task command ${JSON.stringify(name)}`)
  }
  for (const [label, pattern] of [
    ['YHBox', /\byhbox\b/i],
    ['yhfish', /\byhfish\b/i],
    ['deleted fish preview', /preview[\\/]fish\.png/i],
    ['deleted piano preview', /preview[\\/]piano\.png/i],
  ]) {
    if (pattern.test(markdown)) errors.push(`obsolete public reference: ${label}`)
  }
  return errors
}

function run() {
  const files = markdownFiles()
  const knownTasks = taskRegistry()
  const failures = []
  for (const path of files) {
    const markdown = readFileSync(path, 'utf8')
    for (const error of checkMarkdown(path, markdown, knownTasks)) {
      failures.push(`${normalize(relative(repositoryRoot, path))}: ${error}`)
    }
  }
  if (failures.length > 0) {
    throw new Error(`documentation checks failed:\n${failures.map((item) => `- ${item}`).join('\n')}`)
  }
  console.log(`Documentation OK: ${files.length} Markdown files, local links and Task references verified.`)
}

if (process.argv[1] && fileURLToPath(import.meta.url) === resolve(process.argv[1])) run()
