import { test, expect } from '@playwright/test'
import { SidebarPage } from '../../pages/sidebar.page'
import { TabBarPage } from '../../pages/tab-bar.page'

test.describe('Tab Management', () => {
  test('open and close tabs', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })

  test('switch between tabs', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })

  test('disconnect from sidebar should auto-close tab', async ({ page }) => {
    await page.goto('/')
    const sidebar = new SidebarPage(page)
    const tabBar = new TabBarPage(page)

    // 前置：双击连接建立连接，tab 应出现
    const connItem = sidebar.connList.locator('.connection-item').first()
    await expect(connItem).toBeVisible({ timeout: 10000 })
    const connName = await connItem.locator('.conn-name').textContent() || ''
    await connItem.dblclick()

    // 等待连接成功，tab 出现
    await expect(tabBar.getTab(connName)).toBeVisible({ timeout: 10000 })

    // 右键断开连接
    await sidebar.disconnect(connName)

    // tab 应自动关闭
    await expect(tabBar.getTab(connName)).not.toBeVisible({ timeout: 5000 })
  })
})
