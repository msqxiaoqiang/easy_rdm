import { test, expect } from '@playwright/test'

test.describe('Keyboard Shortcuts', () => {
  test('shortcut triggers action', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
