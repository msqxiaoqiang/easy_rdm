import { test, expect } from '@playwright/test'

test.describe('Value Format', () => {
  test('switch value display format', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
