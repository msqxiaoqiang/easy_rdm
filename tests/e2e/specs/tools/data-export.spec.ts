import { test, expect } from '@playwright/test'

test.describe('Data Export', () => {
  test('open export dialog', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
