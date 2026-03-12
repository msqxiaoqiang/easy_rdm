import { test, expect } from '@playwright/test'

test.describe('Migrate', () => {
  test('open migrate dialog', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
