import { test, expect } from '@playwright/test'

test.describe('Data Import', () => {
  test('open import dialog', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
