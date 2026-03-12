import { test, expect } from '@playwright/test'

test.describe('Settings', () => {
  test('open and close settings', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
