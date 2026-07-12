import { mkdir, readFile, readdir, writeFile } from 'node:fs/promises'
import { dirname, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import ts from 'typescript'

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const bindingsRoot = resolve(frontendRoot, 'bindings/github.com/yottaapp/yotta')
const contractPath = resolve(frontendRoot, '../contracts/wails-rpc.json')
const writeMode = process.argv.includes('--write')

const contract = await buildContract(bindingsRoot)
const serialized = `${JSON.stringify(contract, null, 2)}\n`

if (writeMode) {
  await mkdir(dirname(contractPath), { recursive: true })
  await writeFile(contractPath, serialized)
  console.log(
    `wrote ${contract.summary.services} services, ${contract.summary.methods} methods and ${contract.summary.models} models to ${contractPath}`,
  )
  process.exit(0)
}

let expected
try {
  expected = await readFile(contractPath, 'utf8')
} catch (error) {
  fail(
    `cannot read ${contractPath}: ${error.message}\nRun pnpm bindings:update after reviewing the generated API.`,
  )
}

if (expected !== serialized) {
  fail(
    `generated Wails contract differs from ${contractPath}. Review the RPC/model change, then run pnpm bindings:update.`,
  )
}

console.log(
  `Wails contract OK: ${contract.summary.services} services, ${contract.summary.methods} methods, ${contract.summary.models} models`,
)

async function buildContract(root) {
  const paths = (await listTypeScriptFiles(root)).sort()
  if (paths.length === 0) fail(`no generated TypeScript bindings found under ${root}`)

  const services = []
  const models = []
  for (const path of paths) {
    const text = await readFile(path, 'utf8')
    const source = ts.createSourceFile(path, text, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS)
    const modulePath = relative(root, path).replaceAll('\\', '/').replace(/\.ts$/, '')
    const methods = []
    const declarations = []

    for (const statement of source.statements) {
      if (!isExported(statement)) continue
      if (ts.isFunctionDeclaration(statement) && statement.name) {
        methods.push(readMethod(statement, source))
        continue
      }
      const declaration = readModel(statement, source)
      if (declaration) declarations.push(declaration)
    }

    if (methods.length > 0) {
      services.push({ module: modulePath, methods: methods.sort(byName) })
    }
    if (declarations.length > 0) {
      models.push({ module: modulePath, declarations: declarations.sort(byName) })
    }
  }

  services.sort(byModule)
  models.sort(byModule)
  return {
    schemaVersion: 1,
    generator: 'wails3',
    summary: {
      services: services.length,
      methods: services.reduce((total, service) => total + service.methods.length, 0),
      models: models.reduce((total, module) => total + module.declarations.length, 0),
    },
    services,
    models,
  }
}

function readMethod(node, source) {
  let bindingId
  visit(node.body)
  if (!bindingId) fail(`missing $Call.ByID in ${node.name.text}`)

  return {
    name: node.name.text,
    bindingId,
    parameters: node.parameters.map((parameter) => ({
      name: parameter.name.getText(source),
      optional: Boolean(parameter.questionToken || parameter.initializer),
      type: parameter.type?.getText(source) ?? 'unknown',
    })),
    returns: node.type?.getText(source) ?? 'unknown',
  }

  function visit(current) {
    if (!current || bindingId) return
    if (
      ts.isCallExpression(current) &&
      current.expression.getText(source) === '$Call.ByID' &&
      current.arguments.length > 0
    ) {
      bindingId = current.arguments[0].getText(source)
      return
    }
    ts.forEachChild(current, visit)
  }
}

function readModel(node, source) {
  if (ts.isClassDeclaration(node) && node.name) {
    return {
      kind: 'class',
      name: node.name.text,
      fields: node.members.filter(ts.isPropertyDeclaration).map((field) => ({
        name: field.name.getText(source),
        optional: Boolean(field.questionToken),
        type: field.type?.getText(source) ?? 'unknown',
      })),
    }
  }
  if (ts.isEnumDeclaration(node)) {
    return {
      kind: 'enum',
      name: node.name.text,
      members: node.members.map((member) => ({
        name: member.name.getText(source),
        value: member.initializer?.getText(source) ?? null,
      })),
    }
  }
  if (ts.isInterfaceDeclaration(node)) {
    return {
      kind: 'interface',
      name: node.name.text,
      members: node.members.map((member) => member.getText(source)),
    }
  }
  if (ts.isTypeAliasDeclaration(node)) {
    return {
      kind: 'type',
      name: node.name.text,
      definition: node.type.getText(source),
    }
  }
  return null
}

function isExported(node) {
  return node.modifiers?.some((modifier) => modifier.kind === ts.SyntaxKind.ExportKeyword) ?? false
}

async function listTypeScriptFiles(root) {
  const result = []
  for (const entry of await readdir(root, { withFileTypes: true })) {
    const path = resolve(root, entry.name)
    if (entry.isDirectory()) result.push(...(await listTypeScriptFiles(path)))
    else if (entry.isFile() && entry.name.endsWith('.ts') && entry.name !== 'index.ts')
      result.push(path)
  }
  return result
}

function byName(left, right) {
  return left.name.localeCompare(right.name)
}

function byModule(left, right) {
  return left.module.localeCompare(right.module)
}

function fail(message) {
  console.error(message)
  process.exit(1)
}
