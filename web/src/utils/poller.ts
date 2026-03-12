import { request } from './request'

export interface PollEvent {
  seq: number
  time: number
  data: any
}

export interface PollerOptions {
  /** 连接 ID */
  connId: string
  /** 场景标识：pubsub / monitor / status / sentinel */
  scene: string
  /** 长轮询超时（秒），后端阻塞等待时间，默认 25 */
  timeout?: number
  /** 短轮询间隔（毫秒），仅 timeout=0 时生效，默认 5000 */
  interval?: number
  /** 收到增量数据时的回调 */
  onData: (events: PollEvent[]) => void
  /** 错误回调 */
  onError?: (err: any) => void
}

/**
 * 通用轮询客户端
 *
 * 支持两种模式：
 * - 长轮询（默认）：收到响应后立即发下一次请求，后端阻塞等待数据
 * - 短轮询：定时调用，适合状态图表等低频场景
 */
export class Poller {
  private connId: string
  private scene: string
  private timeout: number
  private interval: number
  private onData: (events: PollEvent[]) => void
  private onError?: (err: any) => void

  private _active: boolean = false
  private lastSeq: number = 0

  constructor(options: PollerOptions) {
    this.connId = options.connId
    this.scene = options.scene
    this.timeout = options.timeout ?? 25
    this.interval = options.interval ?? 5000
    this.onData = options.onData
    this.onError = options.onError
  }

  /** 启动轮询 */
  start() {
    if (this._active) return
    this._active = true
    this.loop()
  }

  /** 停止轮询 */
  stop() {
    this._active = false
  }

  /** 重置 offset，下次拉取从头开始 */
  reset() {
    this.lastSeq = 0
  }

  /** 更新回调（组件挂载/卸载时切换） */
  setCallbacks(onData: (events: PollEvent[]) => void, onError?: (err: any) => void) {
    this.onData = onData
    this.onError = onError
  }

  /** 是否正在运行 */
  get running(): boolean {
    return this._active
  }

  private async loop() {
    while (this._active) {
      try {
        const res = await request<PollEvent[]>('poll', {
          params: {
            conn_id: this.connId,
            scene: this.scene,
            after: this.lastSeq,
            timeout: this.timeout,
          },
        })
        if (!this._active) break
        const events = res.data
        if (events && events.length > 0) {
          this.lastSeq = events[events.length - 1].seq
          this.onData(events)
        }
      } catch (err) {
        if (!this._active) break
        this.onError?.(err)
        // 出错后短暂等待再重试
        await new Promise(r => setTimeout(r, 2000))
      }
      // 短轮询模式下加间隔
      if (this._active && this.timeout === 0) {
        await new Promise(r => setTimeout(r, this.interval))
      }
    }
  }
}
