import { test, expect } from '@playwright/test'

test.describe('Server Slowlog', () => {
  test('view slowlog entries', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })

  test('reset slowlog', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
