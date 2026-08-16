import { expect, test } from '@playwright/test'

test('root page mounts the main application without fatal errors', async ({ page }) => {
  const pageErrors: Error[] = []
  page.on('pageerror', (error) => pageErrors.push(error))

  const response = await page.goto('/')

  expect(response).not.toBeNull()
  expect(response?.ok()).toBe(true)
  await expect(page.locator('#app')).toBeVisible()
  await expect(page.getByRole('heading', { name: '登录你的工作台' })).toBeVisible()
  await page.waitForTimeout(300)
  expect(pageErrors).toEqual([])
})

test('successful login persists the session and opens the spaces dashboard', async ({ page }) => {
  await page.route('**/api/v1/sdp/auth/login', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        access_token: 'access-123',
        refresh_token: 'refresh-123',
        expires_in: 3600,
        user: {
          id: 'user-1',
          username: 'operator',
          email: 'operator@example.com',
          avatar: '',
          tenant_id: 1,
          is_active: true,
        },
      }),
    })
  })
  await page.route('**/api/v1/sdp/spaces', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
  })

  await page.goto('/login')
  await page.getByPlaceholder('请输入手机号').fill('13800138000')
  await page.getByPlaceholder('请输入密码').fill('Password1!')
  await page.getByRole('button', { name: '登录' }).click()

  await expect(page).toHaveURL(/\/spaces$/)
  const session = await page.evaluate(() => ({
    accessToken: localStorage.getItem('sdp_access_token'),
    refreshToken: localStorage.getItem('sdp_refresh_token'),
    user: JSON.parse(localStorage.getItem('sdp_user') || 'null'),
  }))
  expect(session).toEqual({
    accessToken: 'access-123',
    refreshToken: 'refresh-123',
    user: expect.objectContaining({ id: 'user-1', email: 'operator@example.com' }),
  })
})

test('authenticated users cannot remain on the login route', async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem('sdp_access_token', 'access-123')
    localStorage.setItem('sdp_refresh_token', 'refresh-123')
    localStorage.setItem('sdp_user', JSON.stringify({ id: 'user-1', username: 'operator' }))
  })
  await page.route('**/api/v1/sdp/spaces', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
  })

  await page.goto('/login')

  await expect(page).toHaveURL(/\/spaces$/)
})

test('failed login stays on the form and shows the backend error', async ({ page }) => {
  await page.route('**/api/v1/sdp/auth/login', async (route) => {
    await route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({ error: '账号或密码错误' }),
    })
  })

  await page.goto('/login')
  await page.getByPlaceholder('请输入手机号').fill('13800138000')
  await page.getByPlaceholder('请输入密码').fill('wrong-password')
  await page.getByRole('button', { name: '登录' }).click()

  await expect(page).toHaveURL(/\/login$/)
  await expect(page.getByText('账号或密码错误')).toBeVisible()
})

test('document import starts an upload request and completes parsing', async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem('sdp_access_token', 'access-123')
    localStorage.setItem('sdp_refresh_token', 'refresh-123')
    localStorage.setItem('sdp_user', JSON.stringify({ id: 'user-1', username: 'operator' }))
  })
  await page.route('**/api/v1/sdp/spaces/space-1', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ id: 'space-1', name: '测试空间', description: '', visibility: 'private', owner_id: 'user-1', created_at: '2026-08-15T00:00:00Z', updated_at: '2026-08-15T00:00:00Z' }),
  }))
  await page.route('**/api/v1/sdp/spaces/space-1/members', route => route.fulfill({ status: 200, contentType: 'application/json', body: '{"members":[]}' }))
  await page.route('**/api/v1/sdp/documents?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: '{"documents":[],"total":0,"page":1,"page_size":100}' }))
  await page.route('**/api/v1/sdp/qa/sessions', route => route.fulfill({ status: 200, contentType: 'application/json', body: '{"sessions":[]}' }))
  await page.route('**/api/v1/sdp/documents/doc-1/parse-status', route => route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"completed","progress":100}' }))

  let uploadedBody = ''
  await page.route('**/api/v1/sdp/documents/upload', async route => {
    uploadedBody = (await route.request().postDataBuffer())?.toString('utf8') || ''
    await route.fulfill({
      status: 201,
      contentType: 'application/json',
      body: JSON.stringify({ document: { id: 'doc-1', space_id: 'space-1' }, message: 'created' }),
    })
  })

  await page.goto('/spaces/space-1')
  await page.getByRole('button', { name: '导入文档' }).first().click()
  await page.locator('input[type="file"]').setInputFiles({ name: 'contract.txt', mimeType: 'text/plain', buffer: Buffer.from('test document') })
  await expect(page.getByText('等待中')).toBeVisible()
  await page.getByRole('button', { name: '开始导入' }).click()

  await expect.poll(() => uploadedBody).toContain('name="space_id"\r\n\r\nspace-1')
  expect(uploadedBody).toContain('filename="contract.txt"')
  await expect(page.getByText('已完成')).toBeVisible()
})

test('document import drawer remains usable in short viewports', async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem('sdp_access_token', 'access-123')
    localStorage.setItem('sdp_refresh_token', 'refresh-123')
    localStorage.setItem('sdp_user', JSON.stringify({ id: 'user-1', username: 'operator' }))
  })
  await page.route('**/api/v1/sdp/spaces/space-1', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ id: 'space-1', name: '测试空间', description: '', visibility: 'private', owner_id: 'user-1', created_at: '2026-08-15T00:00:00Z', updated_at: '2026-08-15T00:00:00Z' }),
  }))
  await page.route('**/api/v1/sdp/spaces/space-1/members', route => route.fulfill({ status: 200, contentType: 'application/json', body: '{"members":[]}' }))
  await page.route('**/api/v1/sdp/documents?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: '{"documents":[],"total":0,"page":1,"page_size":100}' }))
  await page.route('**/api/v1/sdp/qa/sessions', route => route.fulfill({ status: 200, contentType: 'application/json', body: '{"sessions":[]}' }))

  for (const viewport of [{ width: 1440, height: 650 }, { width: 1024, height: 600 }]) {
    await page.setViewportSize(viewport)
    await page.goto('/spaces/space-1')
    await page.getByRole('button', { name: '导入文档' }).first().click()

    const drawer = page.locator('.drawer')
    const body = page.locator('.drawer-body')
    const footer = page.locator('.drawer-foot')
    const selectFile = page.getByRole('button', { name: '选择文件', exact: true })
    await expect(drawer).toBeVisible()
    await expect(footer).toBeVisible()
    await expect(selectFile).toBeVisible()

    const layout = await page.evaluate(() => {
      const drawerElement = document.querySelector<HTMLElement>('.drawer')!
      const bodyElement = document.querySelector<HTMLElement>('.drawer-body')!
      const footerElement = document.querySelector<HTMLElement>('.drawer-foot')!
      const selectElement = document.querySelector<HTMLElement>('.select-file')!
      return {
        drawerBottom: drawerElement.getBoundingClientRect().bottom,
        footerTop: footerElement.getBoundingClientRect().top,
        footerBottom: footerElement.getBoundingClientRect().bottom,
        bodyBottom: bodyElement.getBoundingClientRect().bottom,
        selectBottom: selectElement.getBoundingClientRect().bottom,
        bodyOverflowY: getComputedStyle(bodyElement).overflowY,
      }
    })
    expect(layout.drawerBottom).toBeLessThanOrEqual(viewport.height)
    expect(layout.footerBottom).toBeLessThanOrEqual(viewport.height)
    expect(layout.bodyBottom).toBeLessThanOrEqual(layout.footerTop)
    expect(layout.selectBottom).toBeLessThanOrEqual(layout.footerTop)
    expect(layout.bodyOverflowY).toBe('auto')

    const fileChooserPromise = page.waitForEvent('filechooser')
    await selectFile.click()
    await fileChooserPromise
  }
})
