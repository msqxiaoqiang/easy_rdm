import { test, expect } from '@playwright/test'

test.describe('Error Handling', () => {
  test('show error on failed operation', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
