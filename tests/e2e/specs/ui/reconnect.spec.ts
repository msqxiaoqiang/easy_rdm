import { test, expect } from '@playwright/test'

test.describe('Reconnect', () => {
  test('reconnect after disconnect', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
