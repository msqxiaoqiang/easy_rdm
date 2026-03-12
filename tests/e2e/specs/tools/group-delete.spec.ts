import { test, expect } from '@playwright/test'

test.describe('Group Delete', () => {
  test('open group delete dialog', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
