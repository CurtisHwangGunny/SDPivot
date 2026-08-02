import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const routerSource = await readFile(new URL('../src/router/index.ts', import.meta.url), 'utf8')
const redesignRouterSource = await readFile(new URL('../src/router/redesign.ts', import.meta.url), 'utf8')
const layoutSource = await readFile(new URL('../src/layouts/MainLayout.vue', import.meta.url), 'utf8')

test('route imports are gated directly by the Vite compile-time mode in their module', () => {
  assert.match(routerSource, /const isOpBuild = import\.meta\.env\.MODE === 'op'/)
  assert.doesNotMatch(routerSource, /createMainRoutes/)
  assert.match(routerSource, /\.\.\.\(!isOpBuild[\s\S]*?import\('@\/views\/ops\/OpsPage\.vue'\)[\s\S]*?: \[\]\)/)
  assert.match(routerSource, /\.\.\.\(!isOpBuild[\s\S]*?import\('@\/views\/ops\/OpsLoginPage\.vue'\)[\s\S]*?: \[\]\)/)
  assert.match(routerSource, /\.\.\.\(!isOpBuild[\s\S]*?import\('@\/views\/org\/OrgPage\.vue'\)[\s\S]*?: \[\]\)/)
  assert.match(redesignRouterSource, /const isOpBuild = import\.meta\.env\.MODE === 'op'/)
  assert.match(redesignRouterSource, /\.\.\.\(!isOpBuild[\s\S]*?import\('@\/views\/design\/OrgManagementPage\.vue'\)[\s\S]*?: \[\]\)/)
})

test('standard source contract retains operations routes and UI', () => {
  for (const marker of [
    "path: '/ops-login'",
    "name: 'opsLogin'",
    "path: 'ops'",
    "name: 'ops'",
    "label: '运营管理'",
    "loginPath: '/ops-login'",
    "rootPath: '/ops'",
  ]) {
    assert.equal(`${routerSource}\n${layoutSource}`.includes(marker), true, `missing standard marker: ${marker}`)
  }
})

test('layout operations UI is represented only in a compile-time excluded config branch', () => {
  assert.match(layoutSource, /const isOpBuild = import\.meta\.env\.MODE === 'op'/)
  assert.doesNotMatch(layoutSource, /<t-menu-item[^>]+value="ops"/)
  assert.match(layoutSource, /const operationsUi = isOpBuild\s*\? null\s*:\s*\{[\s\S]*?label: '运营管理'/)
})

test('layout organization UI is represented only in a compile-time excluded config branch', () => {
  assert.doesNotMatch(layoutSource, /<t-menu-item[^>]+value="org"/)
  assert.match(layoutSource, /const organizationUi = isOpBuild \? null : \{ menuValue: 'org', label: '企业管理' \}/)
})
