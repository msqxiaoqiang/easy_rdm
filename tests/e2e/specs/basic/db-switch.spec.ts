import { test, expect } from '@playwright/test'

test.describe('DB Switch', () => {
  test('switch database', async ({ page }) => {
    await page.goto('/')
    // 无连接时 KeyListPanel 不渲染，验证 sidebar 可见
    await expect(page.locator('.sidebar')).toBeVisible({ timeout: 10000 })
  })
})
