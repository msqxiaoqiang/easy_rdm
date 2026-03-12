package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"easy_rdm/app/services"
	"easy_rdm/app/utils"

	"github.com/spf13/viper"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 设置 ServerMode 为 desktop（禁用 GA RPC）
	utils.ServerMode = "desktop"

	// 数据目录：跟随应用安装位置，删应用即删数据
	// macOS .app: Easy RDM.app/Contents/data/
	// Windows:    easy_rdm.exe 同目录/data/
	appDataDir := appRelativeDir()
	if err := os.MkdirAll(appDataDir, 0700); err != nil {
		log.Fatalf("创建数据目录失败: %v", err)
	}
	fmt.Printf("数据目录: %s\n", appDataDir)

	// 加载配置（从数据目录或默认值）
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(appDataDir)
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		// 首次启动：生成随机加密种子并保存配置
		seed := generateRandomSeed(32)
		viper.Set("security.key_seed", seed)
		configPath := filepath.Join(appDataDir, "config.yaml")
		if err := viper.SafeWriteConfigAs(configPath); err != nil {
			log.Printf("保存默认配置失败: %v", err)
		}
	}

	services.InitStorage(filepath.Join(appDataDir, "data"))
	utils.InitCrypto(viper.GetString("security.key_seed"))

	// 创建 App 实例（注册所有 handler）
	app := NewApp()

	// 启动 Wails 应用
	err := wails.Run(&options.App{
		Title:         "Easy RDM",
		Width:         1280,
		Height:        800,
		MinWidth:      800,
		MinHeight:     600,
		DisableResize: false,
		Fullscreen:    false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.startup,
		OnShutdown: func(ctx context.Context) {
			services.DisconnectAll()
		},
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarDefault(),
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})
	if err != nil {
		fmt.Printf("启动失败: %v\n", err)
		os.Exit(1)
	}
}

// generateRandomSeed 使用 crypto/rand 生成 n 字节的随机种子（hex 编码）
func generateRandomSeed(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("生成随机种子失败: %v", err)
	}
	return hex.EncodeToString(b)
}

// appRelativeDir 返回应用安装目录（数据存放在此）
// macOS .app 包: 可执行文件在 Contents/MacOS/easy-rdm → 返回 Contents/
// Windows/其他:  可执行文件同级目录
func appRelativeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	exe, _ = filepath.EvalSymlinks(exe)
	exeDir := filepath.Dir(exe) // Contents/MacOS/

	// macOS .app 包结构：Contents/MacOS/easy-rdm
	// 数据放到 Contents/ 下，这样删除 .app 就一起删除
	if filepath.Base(exeDir) == "MacOS" {
		parent := filepath.Dir(exeDir) // Contents/
		if filepath.Base(parent) == "Contents" {
			return parent
		}
	}

	// Windows / 其他：exe 同级目录
	return exeDir
}
