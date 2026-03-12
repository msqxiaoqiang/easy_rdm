# Easy RDM 构建系统
# 用法:
#   make all           — 构建 x86 + arm 双架构（GMSSH 插件，生产）
#   make build-x86     — 仅构建 Linux x86_64
#   make build-arm     — 仅构建 Linux arm64
#   make build-mac     — 构建 macOS 本地调试版（保留调试配置）
#   make package       — 构建并打包为 tar.gz（用于平台上传）
#   make desktop-mac   — 构建 macOS 桌面客户端（Wails）
#   make desktop-win   — 构建 Windows 桌面客户端（Wails）
#   make clean         — 清理构建产物

APP_NAME    := easy_rdm
OUTPUT_DIR  := omc

.PHONY: all clean init-dirs build-frontend build-backend-x86 build-backend-arm \
        build-x86 build-arm build-mac package desktop-mac desktop-win \
        build-frontend-desktop

# ========== 核心目标 ==========

# 双架构构建（含前端）
all: build-frontend build-x86 copy-frontend-x86 build-arm copy-frontend-arm

clean:
	rm -rf $(OUTPUT_DIR)

# ========== 单架构构建（GMSSH 插件） ==========

build-x86: clean-x86 init-dirs-x86 build-backend-x86
	@echo ">>> x86_64 构建完成: $(OUTPUT_DIR)/x86/"

build-arm: clean-arm init-dirs-arm build-backend-arm
	@echo ">>> arm64 构建完成: $(OUTPUT_DIR)/arm/"

# macOS 本地调试构建（不替换生产配置）
build-mac: clean-mac init-dirs-mac build-frontend
	cd backend && CGO_ENABLED=0 GOOS=darwin \
		GOARCH=$(shell uname -m | sed 's/x86_64/amd64/' | sed 's/arm64/arm64/') \
		go build -ldflags="-s -w" -trimpath -o ../$(OUTPUT_DIR)/mac/app/bin/main .
	cp backend/config.yaml $(OUTPUT_DIR)/mac/config/config.yaml
	cp backend/install.sh $(OUTPUT_DIR)/mac/install.sh
	cp backend/uninstall.sh $(OUTPUT_DIR)/mac/uninstall.sh
	cp -r web/dist/* $(OUTPUT_DIR)/mac/app/www/
	@echo ">>> macOS 构建完成: $(OUTPUT_DIR)/mac/"

# ========== 桌面客户端（Wails） ==========

# macOS 桌面客户端（生成 .app 包）
desktop-mac: build-frontend-desktop
	@echo ">>> 构建 macOS 桌面客户端 (.app)..."
	rm -rf $(OUTPUT_DIR)/desktop-mac
	mkdir -p $(OUTPUT_DIR)/desktop-mac
	# 拷贝前端产物到 desktop/frontend/dist
	rm -rf desktop/frontend/dist
	mkdir -p desktop/frontend/dist
	cp -r web/dist/* desktop/frontend/dist/
	# 使用 wails build 生成 .app 包
	cd desktop && wails build -clean -trimpath -ldflags="-s -w"
	# 将 .app 移到产物目录，并清除隔离属性
	mv desktop/build/bin/*.app $(OUTPUT_DIR)/desktop-mac/
	xattr -cr $(OUTPUT_DIR)/desktop-mac/*.app 2>/dev/null || true
	@echo ">>> macOS 桌面客户端构建完成: $(OUTPUT_DIR)/desktop-mac/"
	@echo ">>> 可直接双击 .app 运行，或拖入 /Applications 安装"
	@echo ">>> 注意：未签名应用，其他 Mac 首次打开需执行: xattr -cr /Applications/Easy\\ RDM.app"

# macOS DMG 安装镜像（可选，依赖 create-dmg）
desktop-dmg: desktop-mac
	@echo ">>> 生成 DMG 安装镜像..."
	@command -v create-dmg >/dev/null 2>&1 || { echo "请先安装: brew install create-dmg"; exit 1; }
	rm -f $(OUTPUT_DIR)/desktop-mac/EasyRDM.dmg
	# 准备 DMG 内容目录（.app + 安装说明）
	rm -rf $(OUTPUT_DIR)/desktop-mac/dmg-tmp
	mkdir -p $(OUTPUT_DIR)/desktop-mac/dmg-tmp
	cp -r $(OUTPUT_DIR)/desktop-mac/*.app $(OUTPUT_DIR)/desktop-mac/dmg-tmp/
	cp desktop/build/安装说明.txt $(OUTPUT_DIR)/desktop-mac/dmg-tmp/
	create-dmg \
		--volname "Easy RDM" \
		--volicon "desktop/build/appicon.icns" \
		--window-pos 200 120 \
		--window-size 660 400 \
		--icon-size 80 \
		--icon "Easy RDM.app" 180 180 \
		--app-drop-link 480 180 \
		--hide-extension "Easy RDM.app" \
		$(OUTPUT_DIR)/desktop-mac/EasyRDM.dmg \
		$(OUTPUT_DIR)/desktop-mac/dmg-tmp
	rm -rf $(OUTPUT_DIR)/desktop-mac/dmg-tmp
	@echo ">>> DMG 安装镜像: $(OUTPUT_DIR)/desktop-mac/EasyRDM.dmg"

# Windows 桌面客户端（需要 mingw-w64 交叉编译）
desktop-win: build-frontend-desktop
	@echo ">>> 构建 Windows 桌面客户端..."
	rm -rf $(OUTPUT_DIR)/desktop-win
	mkdir -p $(OUTPUT_DIR)/desktop-win
	# 拷贝前端产物
	rm -rf desktop/frontend/dist
	mkdir -p desktop/frontend/dist
	cp -r web/dist/* desktop/frontend/dist/
	# 交叉编译 Windows
	cd desktop && CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
		CC=x86_64-w64-mingw32-gcc \
		go build -tags production -ldflags="-s -w -H windowsgui" -trimpath -o ../$(OUTPUT_DIR)/desktop-win/$(APP_NAME).exe .
	mkdir -p $(OUTPUT_DIR)/desktop-win/data
	@echo ">>> Windows 桌面客户端构建完成: $(OUTPUT_DIR)/desktop-win/$(APP_NAME).exe"

# Desktop 模式前端构建（VITE_PLATFORM=desktop）
build-frontend-desktop:
	cd web && VITE_PLATFORM=desktop npx vite build

# ========== 打包（平台上传用） ==========

package: all
	@echo ">>> 打包中..."
	cd $(OUTPUT_DIR)/x86 && COPYFILE_DISABLE=1 tar -czf ../$(APP_NAME)_x86-64.tar.gz *
	@echo ">>> 已创建: $(OUTPUT_DIR)/$(APP_NAME)_x86-64.tar.gz"
	cd $(OUTPUT_DIR)/arm && COPYFILE_DISABLE=1 tar -czf ../$(APP_NAME)_arm64.tar.gz *
	@echo ">>> 已创建: $(OUTPUT_DIR)/$(APP_NAME)_arm64.tar.gz"

# ========== 内部目标 ==========

# 目录初始化（按架构分离）
init-dirs-x86:
	mkdir -p $(OUTPUT_DIR)/x86/app/bin $(OUTPUT_DIR)/x86/app/www \
		$(OUTPUT_DIR)/x86/config $(OUTPUT_DIR)/x86/data \
		$(OUTPUT_DIR)/x86/logs $(OUTPUT_DIR)/x86/tmp

init-dirs-arm:
	mkdir -p $(OUTPUT_DIR)/arm/app/bin $(OUTPUT_DIR)/arm/app/www \
		$(OUTPUT_DIR)/arm/config $(OUTPUT_DIR)/arm/data \
		$(OUTPUT_DIR)/arm/logs $(OUTPUT_DIR)/arm/tmp

init-dirs-mac:
	mkdir -p $(OUTPUT_DIR)/mac/app/bin $(OUTPUT_DIR)/mac/app/www \
		$(OUTPUT_DIR)/mac/config $(OUTPUT_DIR)/mac/data \
		$(OUTPUT_DIR)/mac/logs $(OUTPUT_DIR)/mac/tmp

clean-x86:
	rm -rf $(OUTPUT_DIR)/x86

clean-arm:
	rm -rf $(OUTPUT_DIR)/arm

clean-mac:
	rm -rf $(OUTPUT_DIR)/mac

# 前端构建（GMSSH 模式，只构建一次，由各架构拷贝）
build-frontend:
	@if [ ! -d web/dist ] || [ -z "$$(ls -A web/dist 2>/dev/null)" ]; then \
		cd web && bash build.sh; \
	fi

# 后端构建 + 配置拷贝 + 生产配置替换
build-backend-x86:
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -ldflags="-s -w" -trimpath -o ../$(OUTPUT_DIR)/x86/app/bin/main .
	$(call copy-backend-assets,x86)
	$(call patch-prod-config,x86)

build-backend-arm:
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
		go build -ldflags="-s -w" -trimpath -o ../$(OUTPUT_DIR)/arm/app/bin/main .
	$(call copy-backend-assets,arm)
	$(call patch-prod-config,arm)

# 前端产物拷贝到构建目录
copy-frontend-x86:
	cp -r web/dist/* $(OUTPUT_DIR)/x86/app/www/

copy-frontend-arm:
	cp -r web/dist/* $(OUTPUT_DIR)/arm/app/www/

# ========== 函数 ==========

# 拷贝后端资产（不含前端）
define copy-backend-assets
	cp backend/config.yaml $(OUTPUT_DIR)/$(1)/config/config.yaml
	cp backend/install.sh $(OUTPUT_DIR)/$(1)/install.sh
	cp backend/uninstall.sh $(OUTPUT_DIR)/$(1)/uninstall.sh
endef

# 替换生产配置（socket 模式 + 空 base_path）
define patch-prod-config
	sed -i.bak 's/mode: "http"/mode: "socket"/' $(OUTPUT_DIR)/$(1)/config/config.yaml
	sed -i.bak 's/base_path: "."/base_path: ""/' $(OUTPUT_DIR)/$(1)/config/config.yaml
	rm -f $(OUTPUT_DIR)/$(1)/config/config.yaml.bak
endef
