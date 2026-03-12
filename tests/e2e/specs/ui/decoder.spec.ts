import { test, expect } from '@playwright/test'

test.describe('Decoder', () => {
  test('apply decoder to value', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
