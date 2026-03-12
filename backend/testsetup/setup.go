package testsetup

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"easy_rdm/app/services"
	"easy_rdm/app/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// BizResponse HTTP 模式的业务响应格式
type BizResponse struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

// SetupTestRouter 创建测试用 Gin 引擎，复用生产路由注册逻辑
func SetupTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	services.InitStorage(dir)
	utils.InitCrypto("test-key-seed")

	handlers := map[string]services.RPCHandlerFunc{}
	register := func(method string, handler services.RPCHandlerFunc) {
		handlers[method] = handler
	}
	services.RegisterHandlers(register)

	engine := gin.New()
	engine.Use(cors.Default())

	apiGroup := engine.Group("/api")
	apiGroup.GET("/ping", func(c *gin.Context) {
		data, err := handlers["ping"](nil)
		if err != nil {
			c.JSON(http.StatusOK, BizResponse{Code: 400, Msg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, BizResponse{Code: 200, Data: data, Msg: "OK"})
	})
	for method, handler := range handlers {
		h := handler
		apiGroup.POST("/"+method, func(c *gin.Context) {
			body, _ := c.GetRawData()
			data, err := h(body)
			if err != nil {
				c.JSON(http.StatusOK, BizResponse{Code: 400, Msg: err.Error()})
				return
			}
			c.JSON(http.StatusOK, BizResponse{Code: 200, Data: data, Msg: "OK"})
		})
	}
	return engine
}

// PostJSON 发送 POST JSON 请求
func PostJSON(router *gin.Engine, path string, body interface{}) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", path, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// PostRaw 发送原始 POST 请求体
func PostRaw(router *gin.Engine, path string, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("POST", path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// GetJSON 发送 GET 请求
func GetJSON(router *gin.Engine, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ParseResponse 解析响应
func ParseResponse(t *testing.T, w *httptest.ResponseRecorder) BizResponse {
	t.Helper()
	var resp BizResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response error: %v, body: %s", err, w.Body.String())
	}
	return resp
}
