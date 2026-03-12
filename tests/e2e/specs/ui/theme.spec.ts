import { test, expect } from '@playwright/test'

test.describe('Theme', () => {
  test('toggle theme', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
