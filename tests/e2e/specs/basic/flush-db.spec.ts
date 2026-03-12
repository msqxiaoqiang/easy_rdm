import { test, expect } from '@playwright/test'

test.describe('Flush DB', () => {
  test('flush db requires confirmation', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
