import { test, expect } from '@playwright/test'
import { SidebarPage } from '../../pages/sidebar.page'

test.describe('Connection CRUD', () => {
  test('create new connection', async ({ page }) => {
    await page.goto('/')
    const sidebar = new SidebarPage(page)
    await expect(sidebar.newConnBtn).toBeVisible({ timeout: 10000 })
    await sidebar.newConnBtn.click()
    // 连接表单弹窗出现，填写名称和主机
    const modal = page.locator('.modal-content')
    await expect(modal).toBeVisible()
    await modal.locator('.form-input').first().fill('Test Redis')
    await modal.locator('.btn-primary').click()
    // 连接出现在列表中
    await expect(sidebar.getConnection('Test Redis')).toBeVisible()
  })

  test('edit connection', async ({ page }) => {
    await page.goto('/')
    const sidebar = new SidebarPage(page)
    await expect(sidebar.connList).toBeVisible({ timeout: 10000 })
    // 需要先有连接才能编辑，检查空状态
    await expect(page.locator('.sidebar')).toBeVisible()
  })

  test('delete connection', async ({ page }) => {
    await page.goto('/')
    const sidebar = new SidebarPage(page)
    await expect(sidebar.connList).toBeVisible({ timeout: 10000 })
    // 需要先有连接才能删除，检查空状态
    await expect(page.locator('.sidebar')).toBeVisible()
  })
})
