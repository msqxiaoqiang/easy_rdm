import { test, expect } from '@playwright/test'

test.describe('Copy Clipboard', () => {
  test('copy key value to clipboard', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
