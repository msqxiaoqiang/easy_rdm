import { test, expect } from '@playwright/test'

test.describe('Server Status', () => {
  test('view server status', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
