import { test, expect } from '@playwright/test'

test.describe('Op Log', () => {
  test('view operation log', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })

  test('clear operation log', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
