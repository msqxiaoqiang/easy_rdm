/**
 * 比较两个语义化版本号
 * @returns -1 (a < b), 0 (a == b), 1 (a > b)
 */
export function compareVersion(a: string, b: string): number {
  const pa = a.split('.').map(Number)
  const pb = b.split('.').map(Number)
  const len = Math.max(pa.length, pb.length)
  for (let i = 0; i < len; i++) {
    const na = pa[i] || 0
    const nb = pb[i] || 0
    if (na > nb) return 1
    if (na < nb) return -1
  }
  return 0
}

/** 版本 >= 目标版本 */
export function versionGte(version: string | undefined, target: string): boolean {
  if (!version) return false
  return compareVersion(version, target) >= 0
}
