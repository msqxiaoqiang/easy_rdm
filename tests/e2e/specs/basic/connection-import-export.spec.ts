import { test, expect } from '@playwright/test'
import { SidebarPage } from '../../pages/sidebar.page'

test.describe('Connection Import/Export', () => {
  test('export connections', async ({ page }) => {
    await page.goto('/')
    const sidebar = new SidebarPage(page)
    await expect(sidebar.exportBtn).toBeVisible({ timeout: 10000 })
    await sidebar.exportBtn.click()
    // 导出弹窗出现
    await expect(page.locator('.import-modal')).toBeVisible()
  })

  test('import connections', async ({ page }) => {
    await page.goto('/')
    const sidebar = new SidebarPage(page)
    await expect(sidebar.importBtn).toBeVisible({ timeout: 10000 })
    await sidebar.importBtn.click()
    // 导入弹窗出现，包含 textarea
    await expect(page.locator('.import-modal .import-textarea')).toBeVisible()
  })
})
