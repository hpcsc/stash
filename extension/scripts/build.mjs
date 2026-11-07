#!/usr/bin/env node
import { spawnSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import { cp, mkdir, readFile, rm, writeFile } from 'node:fs/promises'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { buildManifest, chromeExtensionId, firefoxId } from './manifest.mjs'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const dist = join(root, 'dist')
const runtimeDirs = ['src', 'popup', 'options', 'icons', 'picker']
const zipEntries = ['manifest.json', ...runtimeDirs]

function parseArgs(argv) {
  const args = {}
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i]
    if (arg === '--version') {
      args.version = argv[i + 1]
      i += 1
    } else if (arg.startsWith('--version=')) {
      args.version = arg.slice('--version='.length)
    } else {
      throw new Error(`unknown argument: ${arg}`)
    }
  }
  return args
}

function validateVersion(raw) {
  const version = String(raw).trim().replace(/^v/, '')
  if (!/^\d+(\.\d+){0,3}$/.test(version)) {
    throw new Error(`extension version must be one to four dot-separated numbers, got "${raw}"`)
  }
  return version
}

async function resolveVersion(explicit) {
  if (explicit) return validateVersion(explicit)
  if (process.env.EXTENSION_VERSION) return validateVersion(process.env.EXTENSION_VERSION)
  const pkg = JSON.parse(await readFile(join(root, 'package.json'), 'utf8'))
  return validateVersion(pkg.version)
}

async function stage(target, manifest) {
  const out = join(dist, target)
  await mkdir(out, { recursive: true })
  for (const dir of runtimeDirs) {
    await cp(join(root, dir), join(out, dir), { recursive: true })
  }
  await writeFile(join(out, 'manifest.json'), `${JSON.stringify(manifest, null, 2)}\n`)
  return out
}

function zip(sourceDir, outFile) {
  const result = spawnSync('zip', ['-r', '-X', '-q', outFile, ...zipEntries], {
    cwd: sourceDir,
    stdio: 'inherit',
  })
  if (result.error) throw result.error
  if (result.status !== 0) throw new Error(`zip exited with status ${result.status}`)
}

async function main() {
  const args = parseArgs(process.argv.slice(2))
  const version = await resolveVersion(args.version)
  const chromeKey = (await readFile(join(root, 'chrome-key.pub'), 'utf8')).trim()
  if (!chromeKey) throw new Error('chrome-key.pub is empty')

  const base = JSON.parse(await readFile(join(root, 'manifest.json'), 'utf8'))

  await rm(dist, { recursive: true, force: true })
  await mkdir(dist, { recursive: true })

  const chromeDir = await stage('chrome', buildManifest(base, 'chrome', { version, chromeKey }))
  const firefoxDir = await stage('firefox', buildManifest(base, 'firefox', { version }))

  const chromeZip = join(dist, `stash-chrome-${version}.zip`)
  const firefoxZip = join(dist, `stash-firefox-${version}.zip`)
  zip(chromeDir, chromeZip)
  zip(firefoxDir, firefoxZip)

  console.log(`chrome  ${chromeZip}`)
  console.log(`        extension id ${chromeExtensionId(chromeKey)}`)
  console.log(`firefox ${firefoxZip}`)
  console.log(`        add-on id ${firefoxId}`)
}

main().catch((error) => {
  console.error(error.message)
  process.exit(1)
})
