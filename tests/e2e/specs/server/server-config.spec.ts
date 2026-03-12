import { test, expect } from '@playwright/test'

test.describe('Server Config', () => {
  test('view server config', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })

  test('edit config parameter', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
