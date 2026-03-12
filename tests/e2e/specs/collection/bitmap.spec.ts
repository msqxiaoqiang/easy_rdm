import { test, expect } from '@playwright/test'

test.describe('Bitmap Operations', () => {
  test('view bitmap range', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
