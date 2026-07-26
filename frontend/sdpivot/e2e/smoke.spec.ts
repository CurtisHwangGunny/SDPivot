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
