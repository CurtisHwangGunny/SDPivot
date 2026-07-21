import { expect, test } from '@playwright/test'

test('root page mounts the main application without fatal errors', async ({ page }) => {
  const pageErrors: Error[] = []
  page.on('pageerror', (error) => pageErrors.push(error))

  const response = await page.goto('/')

  expect(response).not.toBeNull()
  expect(response?.ok()).toBe(true)
  await expect(page.locator('#app')).toBeVisible()
  await expect(page.locator('#app')).not.toBeEmpty()
  expect(pageErrors).toEqual([])
})
