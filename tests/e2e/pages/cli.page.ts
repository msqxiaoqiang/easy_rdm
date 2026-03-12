import { type Page, type Locator } from '@playwright/test'

export class CliPage {
  readonly page: Page
  readonly input: Locator
  readonly output: Locator
  readonly promptLocator: Locator
  readonly outputLines: Locator

  constructor(page: Page) {
    this.page = page
    this.input = page.locator('.cli-input input')
    this.output = page.locator('.cli-output')
    this.promptLocator = page.locator('.cli-input-row .cli-prompt')
    this.outputLines = page.locator('.cli-output .cli-line')
  }

  async execute(command: string) {
    await this.input.fill(command)
    await this.input.press('Enter')
  }

  async getPromptText() {
    return this.promptLocator.textContent()
  }

  async switchToCliTab() {
    await this.page.locator('.sub-tab', { hasText: /CLI/i }).click()
  }

  async switchToKeyDetailTab() {
    await this.page.locator('.sub-tab', { hasText: /键详情|Key/i }).click()
  }

  async getOutputLineCount() {
    return this.outputLines.count()
  }
}
