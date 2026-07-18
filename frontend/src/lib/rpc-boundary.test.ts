import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const sourceRoot = join(process.cwd(), 'src')

function sourceFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name)
    if (entry.isDirectory()) return sourceFiles(path)
    return /\.(ts|vue)$/.test(entry.name) ? [path] : []
  })
}

describe('typed RPC seam', () => {
  it('keeps transport decoding free of presentation side effects and failure sentinels', () => {
    const invoke = readFileSync(join(sourceRoot, 'lib/invoke.ts'), 'utf8')
    const main = readFileSync(join(sourceRoot, 'main.ts'), 'utf8')

    expect(invoke).toContain('throw toRPCError(error, operation)')
    expect(invoke).not.toContain('useToast')
    expect(invoke).not.toContain('toast.add')
    expect(invoke).not.toContain('invokeVoid')
    expect(main).not.toContain('setupInvoker')
  })

  it('routes bound service calls through the two approved transport modules', () => {
    const offenders = sourceFiles(sourceRoot)
      .filter((path) => !path.endsWith(join('lib', 'backend.ts')))
      .filter((path) => !path.endsWith(join('app', 'transport', 'workflow.ts')))
      .filter((path) =>
        /import\s+\*\s+as\s+\w+\s+from\s+['"]@bindings/.test(readFileSync(path, 'utf8')),
      )

    expect(offenders).toEqual([])
  })

  it('does not use browser-native blocking dialogs', () => {
    const offenders = sourceFiles(sourceRoot).filter((path) =>
      /\b(?:window|globalThis)\.(?:alert|confirm|prompt)\s*\(|\balert\s*\(/.test(
        readFileSync(path, 'utf8'),
      ),
    )

    expect(offenders).toEqual([])
  })

  it('does not type wrapped RPC failure as an optional success value', () => {
    const backend = readFileSync(join(sourceRoot, 'lib/backend.ts'), 'utf8')
    expect(backend).not.toMatch(/as Promise<[^\n>]*\| undefined>/)
  })
})
