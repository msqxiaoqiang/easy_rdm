import { test, expect } from '@playwright/test'

test.describe('Bitfield Operations', () => {
  test('view bitfield values', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
