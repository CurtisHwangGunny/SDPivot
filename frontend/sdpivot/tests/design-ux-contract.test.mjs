import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const readSource = relativePath => readFile(new URL(relativePath, import.meta.url), 'utf8')

const [
  loginSource,
  sidebarSource,
  routerSource,
  spaceDetailSource,
  peopleSource,
  adminSource,
  tagsSource,
  spacesSource,
  qaSource,
  settingsSource,
] = await Promise.all([
  readSource('../src/views/design/LoginPage.vue'),
  readSource('../src/layouts/design/SdpSidebarLayout.vue'),
  readSource('../src/router/index.ts'),
  readSource('../src/views/design/SpaceDetailPage.vue'),
  readSource('../src/views/design/PeopleListPage.vue'),
  readSource('../src/views/design/AdminDashboardPage.vue'),
  readSource('../src/views/design/TagDictionaryPage.vue'),
  readSource('../src/views/design/SpaceHomePage.vue'),
  readSource('../src/views/design/QAWorkspacePage.vue'),
  readSource('../src/views/design/PersonalSettingsPage.vue'),
])

test('login methods use equal native tab targets with a 40px minimum height', () => {
  assert.match(loginSource, /class="sdp-login__tabs" role="tablist"/)
  assert.match(loginSource, /type="button"\s+role="tab"/)
  assert.match(loginSource, /grid-template-columns: repeat\(3, minmax\(var\(--space-0\), 1fr\)\)/)
  assert.match(loginSource, /\.sdp-login__tabs button \{[\s\S]*?min-height: var\(--space-10\)/)
  assert.doesNotMatch(loginSource, /<t-tabs/)
})

test('redesign sidebar exposes a local-first logout action', () => {
  for (const marker of ['aria-label="退出登录"', 'await logout()', 'authStore.clearAuth()', "await router.replace('/login')"]) {
    assert.equal(sidebarSource.includes(marker), true, `missing logout marker: ${marker}`)
  }
  assert.match(sidebarSource, /\.sdp-sidebar-layout__user-card button \{[\s\S]*?min-height: var\(--space-8\)/)
})

test('personal settings uses the redesign sidebar logout layout', () => {
  assert.match(settingsSource, /<SdpSidebarLayout>/)
  assert.match(settingsSource, /import SdpSidebarLayout from '@\/layouts\/design\/SdpSidebarLayout\.vue'/)
})

test('tag dictionary uses the canonical SDPivot admin API', () => {
  for (const marker of [
    "client.get<{ dimensions: TagDimension[]; tags: TagEntry[] }>('/admin/tags')",
    "client.post('/admin/tags', payload)",
    'client.put(`/admin/tags/${encodeURIComponent(editingTag.value.id)}`, payload)',
    'client.delete(`/admin/tags/${encodeURIComponent(tag.id)}`)',
  ]) {
    assert.equal(tagsSource.includes(marker), true, `missing tag API marker: ${marker}`)
  }
  assert.doesNotMatch(tagsSource, /\/api\/v1\/system\/.*tag/)
})

test('login fields and agreement controls meet minimum target sizes', () => {
  assert.match(loginSource, /\.sdp-login__field :deep\(\.t-input__inner\) \{[\s\S]*?min-height: var\(--space-10\)/)
  assert.match(loginSource, /class="sdp-login__agreement-check"/)
  assert.match(loginSource, /\.sdp-login__agreement-check input \{[\s\S]*?width: var\(--space-8\);[\s\S]*?height: var\(--space-8\)/)
  assert.match(loginSource, /\.sdp-login__agreement :deep\(\.t-link\) \{[\s\S]*?min-height: var\(--space-8\)/)
  assert.doesNotMatch(loginSource, /<t-checkbox/)
})

test('space detail route resolves only to the redesign page and exposes required sections', () => {
  assert.doesNotMatch(routerSource, /views\/spaces\/SpaceDetailPage\.vue/)
  assert.doesNotMatch(routerSource, /path: 'spaces\/:id'/)
  for (const marker of ['Workspace Detail', '空间成员', '空间设置摘要', '存储占用', '问答调用', '导入文档', '成员管理']) {
    assert.equal(spaceDetailSource.includes(marker), true, `missing space detail marker: ${marker}`)
  }
  for (const apiMarker of ['getSpace(', 'listSpaceMembers(', 'listDocuments(', 'listSessions(']) {
    assert.equal(spaceDetailSource.includes(apiMarker), true, `missing real data call: ${apiMarker}`)
  }
})

test('reported interactive targets have at least 32px on their accessible element', () => {
  assert.match(spacesSource, /\.sdp-space-home__search input \{[\s\S]*?min-height: var\(--space-8\)/)
  assert.match(qaSource, /\.sdp-qa-workspace__search input \{[^}]*min-height: var\(--space-8\)/)
  assert.match(adminSource, /\.sdp-admin-dashboard__status button,[\s\S]*?min-height: var\(--space-8\)/)
  assert.match(tagsSource, /\.sdp-tag-dictionary__card li button,[\s\S]*?min-height: var\(--space-8\)/)
  assert.match(peopleSource, /<label class="sdp-people-list__checkbox"><input type="checkbox"/)
  assert.match(peopleSource, /\.sdp-people-list input\[type='checkbox'\] \{ width: var\(--space-8\); height: var\(--space-8\)/)
})
