import assert from 'node:assert/strict'
import { lstat, mkdtemp, mkdir, readdir, readFile, rm, stat, symlink } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const dist = fileURLToPath(new URL('../dist/', import.meta.url))

async function files(directory, prefix = '') {
  if ((await lstat(directory)).isSymbolicLink()) {
    throw new Error(`symbolic link is not allowed: ${prefix || path.basename(directory)}`)
  }
  const entries = await readdir(directory, { withFileTypes: true })
  const nested = await Promise.all(entries.map(async (entry) => {
    const relative = path.posix.join(prefix, entry.name)
    if (entry.isSymbolicLink()) {
      throw new Error(`symbolic link is not allowed: ${relative}`)
    }
    return entry.isDirectory() ? files(path.join(directory, entry.name), relative) : [relative]
  }))
  return nested.flat()
}

test('OP artifact scanner rejects symbolic links', async () => {
  const fixture = await mkdtemp(path.join(tmpdir(), 'op-artifact-'))
  try {
    const target = path.join(fixture, 'target')
    const scanRoot = path.join(fixture, 'scan-root')
    const linkedRoot = path.join(fixture, 'linked-root')
    await mkdir(target)
    await mkdir(scanRoot)
    await symlink(target, path.join(scanRoot, 'linked-directory'))
    await symlink(target, linkedRoot)

    await assert.rejects(
      files(scanRoot),
      /symbolic link is not allowed: linked-directory/,
    )
    await assert.rejects(
      files(linkedRoot),
      /symbolic link is not allowed: linked-root/,
    )
  } finally {
    await rm(fixture, { recursive: true, force: true })
  }
})

test('OP artifact contains only the main HTML entry and no operations code', async () => {
  const entries = await files(dist)
  assert.deepEqual(entries.filter((entry) => entry.endsWith('.html')), ['index.html'])
  assert.equal((await stat(path.join(dist, 'index.html'))).isFile(), true)

  const forbiddenNames = /(?:^|\/)(?:ops(?:-[A-Za-z0-9_-]+)?|OpsLoginPage-[A-Za-z0-9_-]+|OpsPage-[A-Za-z0-9_-]+)\.(?:js|css)$/
  assert.deepEqual(entries.filter((entry) => forbiddenNames.test(entry)), [])

  const assets = entries.filter((entry) => /\.(?:html|js|css)$/.test(entry))
  const text = (await Promise.all(assets.map((entry) => readFile(path.join(dist, entry), 'utf8')))).join('\n')
  for (const marker of ['/ops-login', 'OpsLoginPage', 'OpsPage', '运营管理']) {
    assert.equal(text.includes(marker), false, `unexpected operations marker: ${marker}`)
  }
  assert.equal(/(?:^|["'`])\/ops(?:[\/"'`?#]|$)/m.test(text), false, 'unexpected /ops route literal')
  for (const marker of ['/organizations', 'OrgManagementPage', 'OrgPage', '创建组织', '加入企业', '企业管理']) {
    assert.equal(text.includes(marker), false, `unexpected organization marker: ${marker}`)
  }
  assert.equal(/(?:^|["'`])\/org(?:[\/"'`?#]|$)/m.test(text), false, 'unexpected /org route literal')
})
