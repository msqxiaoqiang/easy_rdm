import { test, expect } from '@playwright/test'

test.describe('Server Persistence', () => {
  test('view persistence info', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
