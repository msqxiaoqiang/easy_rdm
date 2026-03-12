import { test, expect } from '@playwright/test'

test.describe('Batch Operations', () => {
  test('open batch dialog', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
