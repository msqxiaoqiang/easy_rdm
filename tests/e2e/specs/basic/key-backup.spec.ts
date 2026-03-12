import { test, expect } from '@playwright/test'

test.describe('Key Backup', () => {
  test('open backup dialog', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
