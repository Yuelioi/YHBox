import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { resolve } from 'node:path'

const CHECK_ORDER = [
  'check:changed:self',
  'check:supply-chain:actions',
  'check:supply-chain:toolchains',
  'check:supply-chain:artifacts',
  'check:supply-chain:go',
  'check:supply-chain:frontend',
  'check:rust',
  'contracts:check',
  'plugins:check',
  'check:ai-eval',
  'version:verify',
  'wails:verify',
  'check:bindings',
  'check:go:changed',
  'check:frontend:quick',
]

const normalize = (path) => path.trim().replaceAll('\\', '/')
const matches = (path, pattern) => pattern.test(path)

export function planChecks(paths) {
  const files = [...new Set(paths.map(normalize).filter(Boolean))]
  const selected = new Set()
  const add = (check) => selected.add(check)

  for (const path of files) {
    if (matches(path, /^(Taskfile\.yml|scripts\/check-(changed|go-changed)(\.test)?\.mjs)$/)) {
      add('check:changed:self')
    }
    if (matches(path, /^\.github\/workflows\//)) add('check:supply-chain:actions')
    if (matches(path, /^(\.tool-versions|scripts\/verify-toolchains\.ps1)$/)) {
      add('check:supply-chain:toolchains')
    }
    if (matches(path, /^(third_party\/|scripts\/verify-third-party-artifacts\.ps1)/)) {
      add('check:supply-chain:artifacts')
    }
    if (matches(path, /^(go\.mod|go\.sum)$/)) {
      add('check:supply-chain:go')
      add('wails:verify')
      add('check:go:changed')
    }
    if (matches(path, /^frontend\/(pnpm-lock\.yaml|package\.json)$/)) {
      add('check:supply-chain:frontend')
    }
    if (matches(path, /^(native\/capture_wgc\/|Cargo\.(toml|lock)$|deny\.toml$)/)) add('check:rust')

    if (matches(path, /^(internal\/(nodes|nodecontract|nodeauthoring|datatype|nodeinstance)\/|internal\/workflow\/schema\/|contracts\/(node|workflow)\/|scripts\/generate-workflow-contracts\.mjs$)/)) {
      add('contracts:check')
    }
    if (matches(path, /^(internal\/(nodepackage|pluginconformance|pluginhost|pluginprotocol|processsandbox|wasmrunner)\/|sdk\/plugin\/|contracts\/plugin\/|cmd\/plugin-sdk\/)/)) {
      add('plugins:check')
    }
    if (matches(path, /^(internal\/(ai|aiauthoring)\/|cmd\/ai-eval\/|contracts\/ai\/)/)) add('check:ai-eval')
    if (matches(path, /^(VERSION|frontend\/package\.json|pkg\/version\/|scripts\/verify-version\.ps1)/)) add('version:verify')
    if (matches(path, /^(wails\.json|build\/.*wails|scripts\/verify-wails-version\.ps1)/)) add('wails:verify')
    if (matches(path, /^(main\.go|internal\/services\/.*\.go$|contracts\/wails-rpc\.json$)/)) add('check:bindings')

    if (path.endsWith('.go')) add('check:go:changed')
    if (matches(path, /^frontend\/(?!dist\/|bindings\/)/) || matches(path, /^contracts\/(node|workflow)\/.*\.ts$/)) {
      add('check:frontend:quick')
    }
  }

  return CHECK_ORDER.filter((check) => selected.has(check))
}

function gitLines(args) {
  const result = spawnSync('git', args, { encoding: 'utf8' })
  if (result.status !== 0) {
    throw new Error(`git ${args.join(' ')} failed: ${(result.stderr || result.stdout).trim()}`)
  }
  return result.stdout.split(/\r?\n/).map(normalize).filter(Boolean)
}

export function changedFiles(environment = process.env) {
  const files = new Set()
  const base = environment.CHECK_BASE?.trim()
  if (base) {
    for (const path of gitLines(['diff', '--name-only', '--diff-filter=ACMR', `${base}...HEAD`])) files.add(path)
  }
  for (const path of gitLines(['diff', '--name-only', '--diff-filter=ACMR', 'HEAD'])) files.add(path)
  for (const path of gitLines(['ls-files', '--others', '--exclude-standard'])) files.add(path)
  return [...files].sort()
}

function run() {
  const files = changedFiles()
  const checks = planChecks(files)
  console.log(`Changed files: ${files.length}`)
  if (checks.length === 0) {
    console.log('No code checks are required for the current change set.')
    return
  }
  console.log(`Planned checks: ${checks.join(', ')}`)
  if (process.argv.includes('--plan')) return
  for (const check of checks) {
    const result = spawnSync('task', [check], { stdio: 'inherit', env: process.env })
    if (result.error) throw result.error
    if (result.status !== 0) process.exit(result.status ?? 1)
  }
}

if (process.argv[1] && fileURLToPath(import.meta.url) === resolve(process.argv[1])) run()
