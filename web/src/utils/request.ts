/**
 * 请求封装 — 从 platform 层重新导出
 *
 * 保持原有 import { request } from '@/utils/request' 路径兼容。
 * 所有实现已迁移到 platform.ts。
 */
export { request, BizError } from './platform'
export type { } from './platform'
