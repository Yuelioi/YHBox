import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { pathToFileURL } from 'node:url'

const sourceDirectories = ['src', 'public', 'scripts', 'bindings']
const sourceFiles = [
  'index.html',
  'package.json',
  'pnpm-lock.yaml',
  'pnpm-workspace.yaml',
  'tsconfig.json',
  'vite.config.ts',
  'auto-imports.d.ts',
  'components.d.ts',
  'bundle-budgets.json',
]

export function isFrontendBuildCurrent(root, mode) {
  const marker = join(root, 'dist', '.yotta-build-mode')
  let markerStat
  try {
    if (readFileSync(marker, 'utf8') !== mode) return false
    markerStat = statSync(marker)
  } catch {
    return false
  }

  for (const relative of sourceFiles) {
    try {
      if (statSync(join(root, relative)).mtimeMs > markerStat.mtimeMs) return false
    } catch {
      // Optional generated declarations may be absent before first install.
    }
  }
  for (const relative of sourceDirectories) {
    if (directoryHasNewerFile(join(root, relative), markerStat.mtimeMs)) return false
  }
  return true
}

function directoryHasNewerFile(directory, markerTime) {
  let entries
  try {
    entries = readdirSync(directory, { withFileTypes: true })
  } catch {
    return false
  }
  for (const entry of entries) {
    const path = join(directory, entry.name)
    if (entry.isDirectory()) {
      if (directoryHasNewerFile(path, markerTime)) return true
    } else if (entry.isFile() && statSync(path).mtimeMs > markerTime) {
      return true
    }
  }
  return false
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  process.exit(isFrontendBuildCurrent(process.cwd(), process.argv[2] ?? '') ? 0 : 1)
}
