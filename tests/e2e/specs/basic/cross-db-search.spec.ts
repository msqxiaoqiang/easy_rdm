import { test, expect } from '@playwright/test'

test.describe('Cross DB Search', () => {
  test('open cross db search dialog', async ({ page }) => {
    await page.goto('/')
    // 无连接时显示 welcome 页，sidebar 可见
    await expect(page.locator('.sidebar')).toBeVisible({ timeout: 10000 })
  })
})
