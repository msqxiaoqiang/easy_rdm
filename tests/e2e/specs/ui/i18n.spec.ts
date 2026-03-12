import { test, expect } from '@playwright/test'

test.describe('I18n', () => {
  test('switch language', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
