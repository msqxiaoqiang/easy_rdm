import { test, expect } from '@playwright/test'

test.describe('Memory Analysis', () => {
  test('open memory view', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
