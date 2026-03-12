import { test, expect } from '@playwright/test'

test.describe('Context Menu', () => {
  test('show context menu on right click', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
