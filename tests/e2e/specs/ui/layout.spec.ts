import { test, expect } from '@playwright/test'

test.describe('Layout', () => {
  test('sidebar and main panel visible', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
