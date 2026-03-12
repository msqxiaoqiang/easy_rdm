package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"easy_rdm/app/services"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 暴露给前端的方法（通过 Wails binding）
type App struct {
	ctx      context.Context
	handlers map[string]services.RPCHandlerFunc
}

// NewApp 创建 App 实例并注册所有业务 handler
func NewApp() *App {
	app := &App{
		handlers: make(map[string]services.RPCHandlerFunc),
	}
	services.RegisterHandlers(func(method string, handler services.RPCHandlerFunc) {
		app.handlers[method] = handler
	})
	return app
}

// startup Wails 生命周期回调
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Call 通用 RPC 桥接：前端通过 Wails binding 调用 → 直接执行 Handler
// 参数和返回值都是 JSON 字符串
func (a *App) Call(method string, paramsJSON string) (result string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC] Call(%s): %v", method, r)
			result = toJSON(bizResponse{Code: 500, Msg: "内部错误"})
		}
	}()

	handler, ok := a.handlers[method]
	if !ok {
		return toJSON(bizResponse{Code: 404, Msg: "method not found"})
	}

	var params json.RawMessage
	if paramsJSON != "" && paramsJSON != "{}" {
		params = json.RawMessage(paramsJSON)
	}

	data, err := handler(params)
	if err != nil {
		return toJSON(bizResponse{Code: 400, Msg: err.Error()})
	}
	return toJSON(bizResponse{Code: 200, Data: data, Msg: "OK"})
}

// ChooseFile 打开系统文件选择对话框
func (a *App) ChooseFile(title string, filters string) (result string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC] ChooseFile: %v", r)
			result = ""
		}
	}()

	if title == "" {
		title = "选择文件"
	}
	opts := runtime.OpenDialogOptions{Title: title}
	if filters != "" {
		// 规范化 filter pattern，macOS 要求 glob 格式
		// 前端可能传 ".zip" 或 ".json,.csv,.txt"
		// Wails FileFilter.Pattern 用分号分隔：如 "*.json;*.csv;*.txt"
		pattern := normalizeFilePattern(filters)
		opts.Filters = []runtime.FileFilter{
			{DisplayName: "Files", Pattern: pattern},
		}
	}
	path, err := runtime.OpenFileDialog(a.ctx, opts)
	if err != nil {
		log.Printf("[ERROR] ChooseFile: %v", err)
		return ""
	}
	return path
}

// ChooseFolder 打开系统目录选择对话框
func (a *App) ChooseFolder(title string) (result string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC] ChooseFolder: %v", r)
			result = ""
		}
	}()

	if title == "" {
		title = "选择目录"
	}
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
	if err != nil {
		log.Printf("[ERROR] ChooseFolder: %v", err)
		return ""
	}
	return path
}

// === 内部工具 ===

type bizResponse struct {
	Code int         `json:"code"`
	Data interface{} `json:"data,omitempty"`
	Msg  string      `json:"msg"`
}

func toJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("[ERROR] toJSON marshal failed: %v", err)
		return `{"code":500,"msg":"序列化错误"}`
	}
	return string(b)
}

// normalizeFilePattern 将前端传入的文件过滤器规范化为 Wails/macOS 要求的 glob 格式
// 输入: ".zip" 或 ".json,.csv,.txt" 或 "*.zip" 或 "zip"
// 输出: "*.zip" 或 "*.json;*.csv;*.txt"
func normalizeFilePattern(filters string) string {
	parts := strings.Split(filters, ",")
	normalized := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, ".") && !strings.HasPrefix(p, "*") {
			// ".zip" → "*.zip"
			p = "*" + p
		} else if !strings.Contains(p, ".") && !strings.Contains(p, "*") {
			// "zip" → "*.zip"
			p = "*." + p
		}
		normalized = append(normalized, p)
	}
	return strings.Join(normalized, ";")
}
