import { test, expect } from '@playwright/test'
import { SidebarPage } from '../../pages/sidebar.page'
import { KeyListPage } from '../../pages/key-list.page'
import { CliPage } from '../../pages/cli.page'

test.describe('CLI', () => {
  test('execute command', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })

  test('SELECT command should update db and persist across reconnect', async ({ page }) => {
    await page.goto('/')
    const sidebar = new SidebarPage(page)
    const keyList = new KeyListPage(page)

    // 前置：连接 Redis
    const connItem = sidebar.connList.locator('.connection-item').first()
    await expect(connItem).toBeVisible({ timeout: 10000 })
    const connName = await connItem.locator('.conn-name').textContent() || ''
    await connItem.dblclick()
    await expect(keyList.dbSelect).toBeVisible({ timeout: 10000 })

    // 通过 db 选择器切换到 db3
    await keyList.switchDB(3)
    await page.waitForTimeout(500)

    // 验证 db 选择器当前值为 3
    await expect(keyList.dbSelect).toHaveValue('3')

    // 断开连接
    await sidebar.disconnect(connName)
    await page.waitForTimeout(500)

    // 重新连接
    await sidebar.connect(connName)
    await expect(keyList.dbSelect).toBeVisible({ timeout: 10000 })

    // 验证重连后 db 应恢复为 3（持久化生效）
    await expect(keyList.dbSelect).toHaveValue('3')
  })

  test('CLI SELECT should only affect CLI prompt, not key list db', async ({ page }) => {
    await page.goto('/')
    const sidebar = new SidebarPage(page)
    const keyList = new KeyListPage(page)
    const cli = new CliPage(page)

    // 前置：连接 Redis
    const connItem = sidebar.connList.locator('.connection-item').first()
    await expect(connItem).toBeVisible({ timeout: 10000 })
    await connItem.dblclick()
    await expect(keyList.dbSelect).toBeVisible({ timeout: 10000 })

    // 确认键列表 db 为 0
    await expect(keyList.dbSelect).toHaveValue('0')

    // 切换到 CLI tab
    await cli.switchToCliTab()
    await expect(cli.input).toBeVisible({ timeout: 5000 })

    // 执行 SELECT 5
    await cli.execute('SELECT 5')
    await page.waitForTimeout(500)

    // CLI prompt 应显示 db5
    await expect(cli.promptLocator).toContainText('db5')

    // 切回键详情
    await cli.switchToKeyDetailTab()
    await expect(keyList.dbSelect).toBeVisible({ timeout: 5000 })

    // 键列表 db 选择器应仍为 0（不受 CLI SELECT 影响）
    await expect(keyList.dbSelect).toHaveValue('0')
  })

  test('CLI history should persist across tab switches', async ({ page }) => {
    await page.goto('/')
    const sidebar = new SidebarPage(page)
    const keyList = new KeyListPage(page)
    const cli = new CliPage(page)

    // 前置：连接 Redis
    const connItem = sidebar.connList.locator('.connection-item').first()
    await expect(connItem).toBeVisible({ timeout: 10000 })
    await connItem.dblclick()
    await expect(keyList.dbSelect).toBeVisible({ timeout: 10000 })

    // 切换到 CLI tab，执行命令
    await cli.switchToCliTab()
    await expect(cli.input).toBeVisible({ timeout: 5000 })
    await cli.execute('PING')
    await page.waitForTimeout(500)

    // 记录输出行数
    const lineCount = await cli.getOutputLineCount()
    expect(lineCount).toBeGreaterThan(0)

    // 切到键详情再切回 CLI
    await cli.switchToKeyDetailTab()
    await page.waitForTimeout(300)
    await cli.switchToCliTab()
    await expect(cli.input).toBeVisible({ timeout: 5000 })

    // 输出行数应与之前一致（历史保留）
    await expect(cli.outputLines).toHaveCount(lineCount)
  })

  test('CLI history should clear after disconnect and reconnect', async ({ page }) => {
    await page.goto('/')
    const sidebar = new SidebarPage(page)
    const keyList = new KeyListPage(page)
    const cli = new CliPage(page)

    // 前置：连接 Redis
    const connItem = sidebar.connList.locator('.connection-item').first()
    await expect(connItem).toBeVisible({ timeout: 10000 })
    const connName = await connItem.locator('.conn-name').textContent() || ''
    await connItem.dblclick()
    await expect(keyList.dbSelect).toBeVisible({ timeout: 10000 })

    // 切换到 CLI tab，执行命令
    await cli.switchToCliTab()
    await expect(cli.input).toBeVisible({ timeout: 5000 })
    await cli.execute('PING')
    await page.waitForTimeout(500)
    const lineCount = await cli.getOutputLineCount()
    expect(lineCount).toBeGreaterThan(0)

    // 断开连接
    await sidebar.disconnect(connName)
    await page.waitForTimeout(500)

    // 重新连接
    await sidebar.connect(connName)
    await expect(keyList.dbSelect).toBeVisible({ timeout: 10000 })

    // 切换到 CLI tab
    await cli.switchToCliTab()
    await expect(cli.input).toBeVisible({ timeout: 5000 })

    // 输出应为空（重连后清空）
    await expect(cli.outputLines).toHaveCount(0)
  })
})
