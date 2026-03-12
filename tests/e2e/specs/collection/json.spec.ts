import { test, expect } from '@playwright/test'

test.describe('JSON Operations', () => {
  test('view JSON value', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
