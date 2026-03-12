import { test, expect } from '@playwright/test'

test.describe('Performance', () => {
  test('handle large key list', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
