import { type Page, type Locator } from '@playwright/test'

export class KeyListPage {
  readonly page: Page
  readonly dbSelect: Locator
  readonly searchInput: Locator
  readonly keyItems: Locator

  constructor(page: Page) {
    this.page = page
    this.dbSelect = page.locator('.db-select')
    this.searchInput = page.locator('.key-search input')
    this.keyItems = page.locator('.key-item')
  }

  async switchDB(db: number) {
    await this.dbSelect.selectOption({ value: String(db) })
  }

  async searchKey(pattern: string) {
    await this.searchInput.fill(pattern)
    await this.searchInput.press('Enter')
  }

  async clickKey(name: string) {
    await this.page.locator('.key-item', { hasText: name }).click()
  }

  async createKey() {
    await this.page.locator('.key-toolbar .tool-btn').nth(2).click()
  }

  async toggleTreeView() {
    await this.page.locator('.key-toolbar .tool-btn').nth(0).click()
  }

  async openMoreMenu() {
    await this.page.locator('.more-dropdown .tool-btn').click()
  }

  async toggleFavoritesOnly() {
    await this.openMoreMenu()
    await this.page.locator('.more-menu-item', { hasText: /收藏|favorites/i }).click()
  }

  async toggleFavorite(keyName: string) {
    await this.page.locator('.key-item', { hasText: keyName }).locator('.fav-star').click()
  }

  get treeGroups() {
    return this.page.locator('.tree-group')
  }
}
