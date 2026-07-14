import { readdirSync, readFileSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

interface IconifyCollection {
  icons: Record<string, unknown>
  aliases?: Record<string, unknown>
}

function sourceFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name)
    if (entry.isDirectory()) return sourceFiles(path)
    return /\.(?:ts|vue)$/.test(entry.name) ? [path] : []
  })
}

describe('literal Tabler icon references', () => {
  it('only uses icons included in the installed collection', () => {
    const collection = JSON.parse(
      readFileSync(join(process.cwd(), 'node_modules/@iconify-json/tabler/icons.json'), 'utf8'),
    ) as IconifyCollection
    const available = new Set([
      ...Object.keys(collection.icons),
      ...Object.keys(collection.aliases ?? {}),
    ])
    const missing: string[] = []

    for (const file of sourceFiles(join(process.cwd(), 'src'))) {
      const source = readFileSync(file, 'utf8')
      for (const match of source.matchAll(/i-tabler-([a-z0-9-]+)/g)) {
        if (!available.has(match[1])) {
          missing.push(`${relative(process.cwd(), file)}: i-tabler-${match[1]}`)
        }
      }
    }

    expect(missing).toEqual([])
  })
})
