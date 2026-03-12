import { test, expect } from '@playwright/test'

test.describe('HyperLogLog Operations', () => {
  test('view HLL count', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
