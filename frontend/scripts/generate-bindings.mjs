import { spawn } from 'node:child_process'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../..')
const command = process.platform === 'win32' ? 'wails3.exe' : 'wails3'
const child = spawn(command, ['generate', 'bindings', '-ts', './...'], {
  cwd: repoRoot,
  env: process.env,
  stdio: ['ignore', 'pipe', 'pipe'],
})

let output = ''
child.stdout.on('data', forward(process.stdout))
child.stderr.on('data', forward(process.stderr))

child.on('error', (error) => {
  console.error(`cannot start ${command}: ${error.message}`)
  process.exit(1)
})

child.on('close', (code) => {
  if (code !== 0) process.exit(code ?? 1)
  if (/[1-9][0-9]* warnings? emitted/i.test(output)) {
    console.error('Wails binding generation emitted warnings')
    process.exit(1)
  }
})

function forward(stream) {
  return (chunk) => {
    const text = chunk.toString()
    output += text
    stream.write(chunk)
  }
}
