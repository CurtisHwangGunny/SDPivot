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
