package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"easy_rdm/app/services"
	"easy_rdm/app/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/spf13/viper"
)

//go:embed all:www
var embeddedFrontend embed.FS

// ========== RPC 响应（与 simplejrpc-go Response 格式一致） ==========

type RPCMeta struct {
	Endpoint string `json:"endpoint"`
	Close    int    `json:"close"`
}

type RPCResult struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
	Meta *RPCMeta    `json:"meta"`
}

// ========== Socket 模式（JSON-RPC 2.0 + VSCode Codec） ==========

var rpcHandlers = map[string]services.RPCHandlerFunc{}

func registerRPC(method string, handler services.RPCHandlerFunc) {
	rpcHandlers[method] = handler
}

// rpcBridge 桥接 sourcegraph/jsonrpc2 与业务 handler
type rpcBridge struct{}

func (h *rpcBridge) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	defer func() {
		if r := recover(); r != nil {
			result := RPCResult{
				Code: http.StatusInternalServerError,
				Msg:  fmt.Sprintf("internal error: %v", r),
				Meta: &RPCMeta{Endpoint: req.Method},
			}
			conn.Reply(ctx, req.ID, result)
		}
	}()

	handler, ok := rpcHandlers[req.Method]
	if !ok {
		result := RPCResult{
			Code: http.StatusNotFound,
			Msg:  "method not found",
			Meta: &RPCMeta{Endpoint: req.Method},
		}
		conn.Reply(ctx, req.ID, result)
		return
	}

	var params json.RawMessage
	if req.Params != nil {
		params = *req.Params
	}

	data, err := handler(params)
	result := RPCResult{
		Code: http.StatusOK,
		Data: data,
		Msg:  http.StatusText(http.StatusOK),
		Meta: &RPCMeta{Endpoint: req.Method},
	}
	if err != nil {
		result.Code = http.StatusBadRequest
		result.Msg = err.Error()
		result.Data = nil
	}
	conn.Reply(ctx, req.ID, result)
}

func listenSocket(sockPath string) {
	os.Remove(sockPath)
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		panic(fmt.Sprintf("创建 socket 失败: %v", err))
	}
	os.Chmod(sockPath, 0660)
	fmt.Printf("Socket(JSON-RPC + VSCodeCodec) 服务启动: %s\n", sockPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go func(c net.Conn) {
			<-jsonrpc2.NewConn(
				context.Background(),
				jsonrpc2.NewBufferedStream(c, jsonrpc2.VSCodeObjectCodec{}),
				&rpcBridge{},
			).DisconnectNotify()
		}(conn)
	}
}

// ========== HTTP 响应包装 ==========

type BizResponse struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

func wrapHTTP(handler services.RPCHandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, _ := c.GetRawData()
		data, err := handler(body)
		if err != nil {
			c.JSON(http.StatusOK, BizResponse{Code: 400, Msg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, BizResponse{Code: 200, Data: data, Msg: "OK"})
	}
}

// ========== 信号处理 ==========

func setupSignalHandler(cleanup func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		services.FlushOpLog()
		cleanup()
		os.Exit(0)
	}()
}

// ========== 主入口 ==========

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(utils.ExeRelative("../../config"))
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("读取配置失败: %v", err))
	}

	basePath := utils.ResolveBasePath(viper.GetString("app.base_path"))
	mode := viper.GetString("server.mode")
	utils.ServerMode = mode

	services.InitStorage(filepath.Join(basePath, "data"))
	utils.InitCrypto(viper.GetString("security.key_seed"))
	services.InitOpLog()
	services.RegisterHandlers(registerRPC)

	switch mode {
	case "socket":
		sockPath := filepath.Join(basePath, "tmp", viper.GetString("server.socket.path"))
		os.MkdirAll(filepath.Dir(sockPath), 0755)
		setupSignalHandler(func() { utils.RemoveFile(sockPath) })
		listenSocket(sockPath)

	case "http":
		engine := gin.Default()
		engine.Use(cors.New(cors.Config{
			AllowOriginFunc: func(origin string) bool {
				return strings.HasPrefix(origin, "http://localhost:") ||
					strings.HasPrefix(origin, "http://127.0.0.1:") ||
					origin == "http://localhost" ||
					origin == "http://127.0.0.1"
			},
			AllowMethods: []string{"GET", "POST", "OPTIONS"},
			AllowHeaders: []string{"Content-Type"},
		}))

		// === 注册 /api/ 路由 + 静态文件服务 ===
		apiGroup := engine.Group("/api")
		apiGroup.GET("/ping", func(c *gin.Context) {
			data, err := rpcHandlers["ping"](nil)
			if err != nil {
				c.JSON(http.StatusOK, BizResponse{Code: 400, Msg: err.Error()})
				return
			}
			c.JSON(http.StatusOK, BizResponse{Code: 200, Data: data, Msg: "OK"})
		})
		for method, handler := range rpcHandlers {
			apiGroup.POST("/"+method, wrapHTTP(handler))
		}
		apiGroup.GET("/download_export", func(c *gin.Context) {
			includePasswords := c.Query("include_passwords") == "true"
			tmpDir := os.TempDir()
			params, _ := json.Marshal(map[string]interface{}{
				"include_passwords": includePasswords,
				"export_path":       tmpDir,
			})
			data, err := rpcHandlers["export_connections_zip"](params)
			if err != nil {
				c.JSON(http.StatusOK, BizResponse{Code: 400, Msg: err.Error()})
				return
			}
			result, _ := data.(map[string]interface{})
			zipPath, _ := result["path"].(string)
			if zipPath == "" {
				c.JSON(http.StatusOK, BizResponse{Code: 400, Msg: "导出路径为空"})
				return
			}
			defer os.Remove(zipPath)
			c.FileAttachment(zipPath, filepath.Base(zipPath))
		})
		apiGroup.POST("/upload_import", func(c *gin.Context) {
			file, err := c.FormFile("file")
			if err != nil {
				c.JSON(http.StatusOK, BizResponse{Code: 400, Msg: "未收到文件"})
				return
			}
			src, err := file.Open()
			if err != nil {
				c.JSON(http.StatusOK, BizResponse{Code: 400, Msg: "打开文件失败"})
				return
			}
			defer src.Close()
			tmpFile, err := os.CreateTemp("", "easy_rdm_import_*.zip")
			if err != nil {
				c.JSON(http.StatusOK, BizResponse{Code: 400, Msg: "创建临时文件失败"})
				return
			}
			tmpPath := tmpFile.Name()
			defer os.Remove(tmpPath)
			if _, err := io.Copy(tmpFile, src); err != nil {
				tmpFile.Close()
				c.JSON(http.StatusOK, BizResponse{Code: 400, Msg: "保存文件失败"})
				return
			}
			tmpFile.Close()
			params, _ := json.Marshal(map[string]interface{}{"file_path": tmpPath})
			data, err := rpcHandlers["import_connections_zip"](params)
			if err != nil {
				c.JSON(http.StatusOK, BizResponse{Code: 400, Msg: err.Error()})
				return
			}
			c.JSON(http.StatusOK, BizResponse{Code: 200, Data: data, Msg: "OK"})
		})
		apiGroup.POST("/download_keys_export", func(c *gin.Context) {
			var req struct {
				ConnID  string   `json:"conn_id"`
				Format  string   `json:"format"`
				Scope   string   `json:"scope"`
				Keys    []string `json:"keys"`
				Pattern string   `json:"pattern"`
				Limit   int      `json:"limit"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusOK, BizResponse{Code: 400, Msg: "参数错误"})
				return
			}
			if req.Format == "" {
				req.Format = "json"
			}
			ext := services.ExportFileExt(req.Format)
			contentType := "application/json"
			switch req.Format {
			case "csv":
				contentType = "text/csv"
			case "redis_cmd":
				contentType = "text/plain"
			}
			now := time.Now()
			fileName := fmt.Sprintf("easy_rdm_keys_%s%s", now.Format("20060102_150405_000"), ext)
			c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
			c.Header("Content-Type", contentType)
			c.Status(http.StatusOK)
			_, err := services.StreamExportToWriter(c.Writer, req.ConnID, req.Format, req.Scope, req.Keys, req.Pattern, req.Limit)
			if err != nil {
				fmt.Fprintf(c.Writer, "\n\n# ERROR: %s", err.Error())
			}
		})
		apiGroup.POST("/upload_keys_import", func(c *gin.Context) {
			file, err := c.FormFile("file")
			if err != nil {
				c.JSON(http.StatusOK, BizResponse{Code: 400, Msg: "未收到文件"})
				return
			}
			const maxImportSize = 50 * 1024 * 1024
			if file.Size > maxImportSize {
				c.JSON(http.StatusOK, BizResponse{Code: 400, Msg: fmt.Sprintf("文件过大（%.1fMB），最大支持 50MB", float64(file.Size)/1024/1024)})
				return
			}
			src, err := file.Open()
			if err != nil {
				c.JSON(http.StatusOK, BizResponse{Code: 400, Msg: "打开文件失败"})
				return
			}
			defer src.Close()
			fileData, err := io.ReadAll(src)
			if err != nil {
				c.JSON(http.StatusOK, BizResponse{Code: 400, Msg: "读取文件失败"})
				return
			}
			connID := c.PostForm("conn_id")
			format := c.PostForm("format")
			conflictMode := c.PostForm("conflict_mode")
			if format == "" {
				format = "json"
			}
			if conflictMode == "" {
				conflictMode = "skip"
			}
			params, _ := json.Marshal(map[string]interface{}{
				"conn_id":       connID,
				"format":        format,
				"content":       string(fileData),
				"conflict_mode": conflictMode,
			})
			data, err := rpcHandlers["import_keys"](params)
			if err != nil {
				c.JSON(http.StatusOK, BizResponse{Code: 400, Msg: err.Error()})
				return
			}
			c.JSON(http.StatusOK, BizResponse{Code: 200, Data: data, Msg: "OK"})
		})

		// === 静态文件服务：嵌入式前端或外部目录 ===
		staticDir := viper.GetString("server.static_dir")
		if staticDir != "" {
			// 优先使用配置的外部静态文件目录
			absStaticDir, _ := filepath.Abs(staticDir)
			engine.NoRoute(func(c *gin.Context) {
				path := c.Request.URL.Path
				// 路径遍历防护：确保解析后的路径在 staticDir 范围内
				absFilePath := filepath.Clean(filepath.Join(absStaticDir, path))
				if !strings.HasPrefix(absFilePath, absStaticDir+string(os.PathSeparator)) && absFilePath != absStaticDir {
					c.JSON(http.StatusForbidden, BizResponse{Code: 403, Msg: "forbidden"})
					return
				}
				if _, err := os.Stat(absFilePath); err == nil {
					c.File(absFilePath)
					return
				}
				// SPA fallback: 非 API 路由返回 index.html
				if !strings.HasPrefix(path, "/api/") {
					c.File(filepath.Join(absStaticDir, "index.html"))
					return
				}
				c.JSON(http.StatusNotFound, BizResponse{Code: 404, Msg: "not found"})
			})
		} else {
			// 使用 embed 的前端产物
			frontendFS, err := fs.Sub(embeddedFrontend, "www")
			if err == nil {
				httpFS := http.FS(frontendFS)
				engine.NoRoute(func(c *gin.Context) {
					path := c.Request.URL.Path
					// 尝试直接匹配文件
					if f, err := frontendFS.Open(strings.TrimPrefix(path, "/")); err == nil {
						f.Close()
						c.FileFromFS(path, httpFS)
						return
					}
					// SPA fallback
					if !strings.HasPrefix(path, "/api/") {
						c.FileFromFS("/index.html", httpFS)
						return
					}
					c.JSON(http.StatusNotFound, BizResponse{Code: 404, Msg: "not found"})
				})
			}
		}

		port := viper.GetInt("server.http.port")
		portFile := filepath.Join(basePath, "tmp", "app.port")
		os.MkdirAll(filepath.Dir(portFile), 0755)
		setupSignalHandler(func() { utils.RemoveFile(portFile) })
		if err := utils.WritePortToFile(portFile, port); err != nil {
			panic(fmt.Sprintf("写入端口文件失败: %v", err))
		}
		fmt.Printf("HTTP 服务启动: :%d\n", port)
		host := viper.GetString("server.http.host")
		if err := engine.Run(fmt.Sprintf("%s:%d", host, port)); err != nil {
			utils.RemoveFile(portFile)
			panic(fmt.Sprintf("服务器启动失败: %v", err))
		}
	}
}
