package tests

import (
	"testing"

	"easy_rdm/testsetup"
)

// 验证 GET /api/ping 可用
func TestAPI_PingViaApiPrefix(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.GetJSON(router, "/api/ping")
	if w.Code != 200 {
		t.Fatalf("expected HTTP 200, got %d", w.Code)
	}
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("expected biz code 200, got %d", resp.Code)
	}
}

// 验证裸路由已不存在
func TestAPI_BareRouteReturns404(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.GetJSON(router, "/ping")
	if w.Code != 404 {
		t.Fatalf("expected 404 for bare /ping, got %d", w.Code)
	}
}

// 验证 POST /api/{method} 可用
func TestAPI_PostMethodViaApiPrefix(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/get_connections", map[string]interface{}{})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d, msg: %s", resp.Code, resp.Msg)
	}
}
