import assert from 'node:assert/strict'
import { readdir, readFile, stat } from 'node:fs/promises'
import path from 'node:path'
import test from 'node:test'

const dist = new URL('../dist/', import.meta.url)

async function files(directory, prefix = '') {
  const entries = await readdir(directory, { withFileTypes: true })
  const nested = await Promise.all(entries.map(async (entry) => {
    const relative = path.posix.join(prefix, entry.name)
    return entry.isDirectory() ? files(new URL(`${relative}/`, dist), relative) : [relative]
  }))
  return nested.flat()
}

test('OP artifact contains only the main HTML entry and no operations code', async () => {
  const entries = await files(dist)
  assert.deepEqual(entries.filter((entry) => entry.endsWith('.html')), ['index.html'])
  assert.equal((await stat(new URL('index.html', dist))).isFile(), true)

  const forbiddenNames = /(?:^|\/)(?:ops(?:-[A-Za-z0-9_-]+)?|OpsLoginPage-[A-Za-z0-9_-]+|OpsPage-[A-Za-z0-9_-]+)\.(?:js|css)$/
  assert.deepEqual(entries.filter((entry) => forbiddenNames.test(entry)), [])

  const assets = entries.filter((entry) => /\.(?:html|js|css)$/.test(entry))
  const text = (await Promise.all(assets.map((entry) => readFile(new URL(entry, dist), 'utf8')))).join('\n')
  for (const marker of ['/ops-login', 'OpsLoginPage', 'OpsPage', '运营管理']) {
    assert.equal(text.includes(marker), false, `unexpected operations marker: ${marker}`)
  }
  assert.equal(/(?:^|["'`])\/ops(?:[\/"'`?#]|$)/m.test(text), false, 'unexpected /ops route literal')
})
