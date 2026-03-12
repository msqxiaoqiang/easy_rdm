# Easy RDM — 可视化 Redis 管理工具

可视化 Redis 管理工具，支持三种部署形态：GMSSH 平台插件、Web 独立部署、桌面客户端（Wails）。前端 Vue 3 + TypeScript，后端 Go。

## 功能概览

- 连接管理：多连接配置、分组、导入/导出、密码 AES 加密存储
- Key 浏览：SCAN 分页扫描、按模式过滤、批量删除、重命名、TTL 管理
- 全类型支持：
  - String — 文本/JSON/HEX 格式切换编辑
  - Hash — 字段级 CRUD、过滤、分页
  - List — 索引编辑、头尾插入、范围删除
  - Set — 成员管理、过滤、批量添加
  - ZSet — 分数排序、范围查询、成员编辑
  - Stream — 消息浏览、XADD/XDEL、消费者组管理
  - Geo — 地理坐标管理、距离计算、范围搜索
  - Bitmap — 位可视化、单位设置
  - Bitfield — 多字段定义、GET/SET/INCRBY
  - HyperLogLog — 基数统计、PFADD/PFMERGE
- 新建 Key：支持所有类型的结构化初始值输入（动态行表单）
- CLI 控制台：原生 Redis 命令执行，支持引号解析
- 服务器状态：实时监控 Redis 运行指标、自动刷新
- 操作日志：全操作审计记录、一键清空
- 多语言：中文简体/繁体、English、日本語、한국어、Русский、Français
- 主题：浅色/深色/自动（GMSSH 跟随平台，桌面端和 Web 端跟随系统）

## 项目结构

```
easy_rdm/
├── web/                          # 前端（Vue 3 + Vite + TypeScript）
│   ├── src/
│   │   ├── assets/styles/        # CSS 变量、全局样式
│   │   ├── components/
│   │   │   ├── layout/           # Sidebar, TopTabBar, MainLayout
│   │   │   ├── views/            # KeyListPanel, KeyDetailView, CliView, StatusView
│   │   │   │   └── collections/  # Hash/List/Set/ZSet/Stream/Geo/Bitmap/Bitfield/HLL 详情
│   │   │   ├── connection/       # ConnectionForm
│   │   │   ├── common/           # ContextMenu, OpLogDialog
│   │   │   └── settings/         # SettingsDialog
│   │   ├── stores/               # Pinia 状态管理（connection, app）
│   │   ├── i18n/locales/         # 7 种语言
│   │   ├── utils/                # platform.ts 平台适配层、request 封装
│   │   └── router/               # 路由
│   ├── vite.config.ts
│   ├── package.json
│   └── build.sh
├── backend/                      # 后端（Go 1.23+）
│   ├── main.go                   # 入口，双模服务（Socket/HTTP）
│   ├── app/
│   │   ├── services/
│   │   │   ├── handlers.go       # 连接管理、设置、会话
│   │   │   ├── key_handlers.go   # Key CRUD、SCAN、CLI、导入导出
│   │   │   ├── collection_handlers.go  # Hash/List/Set/ZSet 成员操作
│   │   │   ├── extended_handlers.go    # Stream/Geo/Bitmap/Bitfield/HLL
│   │   │   ├── op_log.go         # 操作日志
│   │   │   ├── redis_pool.go     # Redis 连接池管理
│   │   │   ├── storage.go        # JSON 文件持久化
│   │   │   └── polling.go        # 长轮询
│   │   └── utils/                # 加密、日志、响应类型
│   ├── config.yaml               # 应用配置
│   ├── install.sh
│   ├── uninstall.sh
│   ├── go.mod
│   └── go.sum
├── desktop/                      # 桌面客户端（Wails）
│   ├── main.go                   # Wails 入口，创建窗口 + 绑定
│   ├── app.go                    # 暴露给前端的方法（桥接 services.Handler）
│   ├── wails.json                # Wails 项目配置
│   └── go.mod
├── Makefile                      # 构建脚本
└── README.md
```

## 环境要求

| 依赖 | 版本      |
|------|---------|
| Go | `>= 1.24` |
| Node.js | `>= 18` |
| npm | `>= 9` |

## 本地开发

### 后端

```bash
cd backend

# 安装依赖
go mod download

# 启动（HTTP 调试模式，默认端口 8899）
go run .
```

`config.yaml` 默认 `server.mode: "http"`，`base_path: "."`，直接本地运行即可。日志输出到 `./logs/`，数据存储在 `./data/`。

### 前端

```bash
cd web

# 安装依赖
npm install

# 本地调试（Web 模式，端口 5173，代理 /dev-api → 127.0.0.1:8899/api）
npm run dev

# GM 联调模式（端口 5174，使用真实 $gm SDK，配合 GMSSH 平台调试）
npm run dev:gm
```

- `npm run dev` — 本地调试，Web 模式，请求通过 Vite proxy 转发到本地后端
- `npm run dev:gm` — GMSSH 平台联调，使用平台注入的真实 `$gm.request`，需在 GMSSH 平台 iframe 中访问

### 前后端联调

**本地调试：**

1. 先启动后端：`cd backend && go run .`（监听 8899）
2. 再启动前端：`cd web && npm run dev`（监听 5173，代理到 8899）
3. 浏览器访问 `http://localhost:5173`

**平台联调（GM SDK）：**

1. 启动前端：`cd web && npm run dev:gm`（监听 5174）
2. 在 GMSSH 平台配置开发地址指向 `http://<本机IP>:5174`
3. 请求通过平台 GA Agent 转发到插件后端

### 桌面客户端调试

桌面客户端基于 Wails v2，推荐使用 `wails dev` 进行开发调试，支持前后端热重载。

**环境准备：**

```bash
# 安装 Wails CLI（需要 Go 1.23+）
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 检查环境依赖是否满足
wails doctor
```

**启动调试：**

```bash
cd desktop
wails dev
```

`wails dev` 会自动：
- 安装前端依赖（`cd ../web && npm install`）
- 启动 Vite 开发服务器（`VITE_PLATFORM=desktop`，支持热重载）
- 编译并启动 Go 后端（监听代码变更自动重编译）
- 打开桌面窗口，指向 Vite 开发服务器

修改前端代码后窗口会自动刷新，修改 Go 代码后应用会自动重启。

**手动构建调试（不安装 Wails CLI）：**

```bash
# 1. 构建前端（desktop 模式）
cd web && VITE_PLATFORM=desktop npx vite build --outDir ../desktop/frontend/dist

# 2. 编译运行
cd desktop && CGO_ENABLED=1 go build -tags dev -o ./easy_rdm . && ./easy_rdm
```

> `dev` tag 会启用 Wails 开发者工具（右键菜单可打开 DevTools 检查元素）。生产构建使用 `production` tag。

## 构建打包

```bash
# GMSSH 插件（后端 + GMSSH 模式前端）
make all            # 前端 + 后端，双架构（x86_64 + arm64）
make build-x86      # 仅后端 x86_64（不构建前端）
make build-arm      # 仅后端 arm64（不构建前端）
make build-frontend # 仅前端（GMSSH 模式）
make build-mac      # macOS 本地调试版（含前端，保留调试配置）
make package        # 构建并打包为 tar.gz（用于平台上传）

# Web 独立部署前端（非 GMSSH 模式）
cd web && npm run build:web    # 构建产物在 web/dist/，配合后端 HTTP 模式使用

# 桌面客户端（Wails，自动构建前端并嵌入，需先 cd web && npm install）
make desktop-mac    # macOS 桌面客户端
make desktop-win    # Windows 桌面客户端（需要 mingw-w64 交叉编译工具链）

make clean          # 清理构建产物
```

> **Web 独立部署**：Go 后端 HTTP 模式本身就支持静态文件服务，只需用 `npm run build:web` 构建前端，将产物部署到后端可访问的目录即可。

构建产物输出到 `omc/` 目录：

```
omc/
├── x86/                        # GMSSH 插件 x86_64
│   ├── app/bin/main
│   ├── app/www/                # 前端静态资源
│   ├── config/config.yaml
│   └── ...
├── arm/                        # GMSSH 插件 arm64（结构同上）
├── desktop-mac/                # macOS 桌面客户端
│   ├── easy_rdm               # Wails 应用
│   └── config.yaml
├── desktop-win/                # Windows 桌面客户端
│   ├── easy_rdm.exe
│   └── config.yaml
├── easy_rdm_x86-64.tar.gz
└── easy_rdm_arm64.tar.gz
```

## 桌面客户端使用

### macOS

```bash
make desktop-mac
cd omc/desktop-mac && ./easy_rdm
```

直接双击运行 `easy_rdm` 即可启动，数据存储在同目录 `data/` 下。

### Windows

```bash
make desktop-win
```

将 `omc/desktop-win/` 目录拷贝到 Windows 机器，双击 `easy_rdm.exe` 运行。

> 桌面端为单文件应用（Wails），前端内嵌在可执行文件中，通过进程内 IPC 通信，无需网络服务。配置文件 `config.yaml` 可选，缺失时使用默认配置。

## 后台运行与进程管理（GMSSH 插件）

打包产物为 Socket 模式，通过 `tmp/app.sock` 通信。

### 后台启动

```bash
# 后台运行（以 x86 为例，arm 替换路径即可）
nohup ./omc/x86/app/bin/main &

# 或指定日志文件
nohup ./omc/x86/app/bin/main > ./omc/x86/logs/app.log 2>&1 &
```

### 查看进程

```bash
ps aux | grep bin/main
```

### 停止进程

```bash
# 按 PID 停止（优雅退出，自动清理 tmp/app.sock）
kill <PID>

# 按进程名批量停止
pkill -f "bin/main"
```

> 推荐使用 `kill`（SIGTERM）而非 `kill -9`，确保程序执行退出清理（删除 `tmp/app.sock`）。

## 生产部署

应用上架 GMSSH 后，用户安装时自动部署到：

```
/.__gmssh/plugin/{组织名}/easy_rdm/
```

生产环境需将 `config.yaml` 中的配置调整为：

```yaml
app:
  base_path: ""          # 留空，自动基于 exe 位置推算
server:
  mode: "socket"         # 切换为 Socket 模式
```

### 服务模式说明

| | Socket（GMSSH 生产） | HTTP（调试 + Web 独立部署） | Desktop（Wails 桌面端） |
|---|---|---|---|
| 协议 | JSON-RPC 2.0 over UDS | 标准 HTTP（Gin） | Wails binding（进程内 IPC） |
| 产物 | `tmp/app.sock` | `tmp/app.port` | 无（内嵌） |
| 健康检查 | JSON-RPC method `"ping"` | `GET /api/ping` | N/A |
| 切换方式 | `server.mode: "socket"` | `server.mode: "http"` | `utils.ServerMode = "desktop"` |

## 测试

### 后端

```bash
cd backend
go test ./...
```

### 前端

```bash
cd web
npm test
```

## 主要技术栈

- 前端：Vue 3、Pinia、Vue Router、Vue I18n、Vite、TypeScript
- 后端：Go、Gin、go-redis、Viper
- 桌面端：Wails v2（Go + 系统 WebView）
- 平台适配：`platform.ts` 三端统一适配层（GMSSH / Desktop / Web）
- 通信：GMSSH GM Web SDK（`gm-app-sdk`）/ Wails binding / HTTP fetch
- 构建：Make、交叉编译（linux/amd64、linux/arm64、darwin、windows）

## API 接口一览

### 连接管理
`connect` / `disconnect` / `test_connection` / `get_connections` / `save_connection` / `delete_connection` / `import_connections` / `export_connections`

### Key 操作
`scan_keys` / `get_key_info` / `get_key_value` / `set_key_value` / `delete_keys` / `rename_key` / `set_ttl` / `create_key` / `check_key_exists` / `execute_command`

### 集合成员操作
Hash: `hash_get_fields` / `hash_set_field` / `hash_delete_fields`
List: `list_get_range` / `list_set_index` / `list_push` / `list_rem` / `list_trim`
Set: `set_get_members` / `set_add_members` / `set_remove_members`
ZSet: `zset_get_range` / `zset_add_members` / `zset_remove_members` / `zset_score`

### 扩展类型
Stream: `xrange_messages` / `xadd_message` / `xdel_messages` / `xtrim_stream` / `xinfo_stream` / `xinfo_groups` / `xgroup_create` / `xgroup_destroy`
Geo: `geo_members` / `geo_add` / `geo_dist` / `geo_search`
Bitmap: `bitmap_get_range` / `bitmap_set_bit` / `bitmap_count`
Bitfield: `bitfield_get` / `bitfield_set` / `bitfield_incrby`
HyperLogLog: `pfcount` / `pfadd` / `pfmerge`

### 其他
`select_db` / `get_db_list` / `get_server_status` / `get_op_log` / `clear_op_log` / `get_settings` / `save_settings`

### 致谢
Tiny RDM 项目地址：​ https://github.com/tiny-craft/tiny-rdm

### 学习声明：
Easy RDM 在功能规划和界面布局上参考了​ Tiny RDM的优秀设计，但所有代码均由我本人在 AI 辅助下独立完成，底层架构完全独立设计。
项目采用 GPL-3.0 license​ 开源，既是学习成果，也希望为同样在探索中的开发者提供一个可参考的“AI 辅助学习案例”。

