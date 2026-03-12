import { test, expect } from '@playwright/test'

test.describe('Stream Operations', () => {
  test('view stream messages', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
