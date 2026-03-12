import { test, expect } from '@playwright/test'

test.describe('Lua Script', () => {
  test('open lua editor', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
