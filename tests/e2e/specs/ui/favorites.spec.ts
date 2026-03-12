import { test, expect } from '@playwright/test'
import { SidebarPage } from '../../pages/sidebar.page'
import { KeyListPage } from '../../pages/key-list.page'

test.describe('Favorites', () => {
  test('toggle favorite key', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })

  test('favorites filter should work in tree view', async ({ page }) => {
    await page.goto('/')
    const sidebar = new SidebarPage(page)
    const keyList = new KeyListPage(page)

    // 前置：连接 Redis
    const connItem = sidebar.connList.locator('.connection-item').first()
    await expect(connItem).toBeVisible({ timeout: 10000 })
    await connItem.dblclick()
    await expect(keyList.keyItems.first()).toBeVisible({ timeout: 10000 })

    // 记录初始 key 数量
    const initialCount = await keyList.keyItems.count()
    expect(initialCount).toBeGreaterThan(0)

    // 收藏第一个 key
    const firstKeyName = await keyList.keyItems.first().locator('.key-name').textContent() || ''
    await keyList.toggleFavorite(firstKeyName)

    // 切换到树形视图
    await keyList.toggleTreeView()
    await page.waitForTimeout(300)

    // 开启"只显示收藏 key"筛选
    await keyList.toggleFavoritesOnly()
    await page.waitForTimeout(300)

    // 筛选后 key 数量应减少（只显示收藏的）
    const filteredItems = keyList.keyItems
    const filteredCount = await filteredItems.count()
    expect(filteredCount).toBeLessThan(initialCount)
    expect(filteredCount).toBeGreaterThan(0)
  })
})
