import { type Page, type Locator } from '@playwright/test'

export class SidebarPage {
  readonly page: Page
  readonly newConnBtn: Locator
  readonly newGroupBtn: Locator
  readonly importBtn: Locator
  readonly exportBtn: Locator
  readonly connList: Locator
  readonly settingsBtn: Locator

  constructor(page: Page) {
    this.page = page
    this.newConnBtn = page.locator('.sidebar-toolbar .toolbar-btn').nth(0)
    this.newGroupBtn = page.locator('.sidebar-toolbar .toolbar-btn').nth(1)
    this.importBtn = page.locator('.sidebar-toolbar .toolbar-btn').nth(2)
    this.exportBtn = page.locator('.sidebar-toolbar .toolbar-btn').nth(3)
    this.connList = page.locator('.connection-list')
    this.settingsBtn = page.locator('.footer-settings-btn')
  }

  getConnection(name: string) {
    return this.connList.locator('.connection-item', { hasText: name })
  }

  async connect(name: string) {
    await this.getConnection(name).dblclick()
  }

  async rightClick(name: string) {
    await this.getConnection(name).click({ button: 'right' })
  }

  async selectContextItem(label: string) {
    await this.page.locator('.ctx-menu .ctx-menu-item', { hasText: label }).click()
  }

  async disconnect(name: string) {
    await this.rightClick(name)
    await this.selectContextItem('断开')
  }
}
