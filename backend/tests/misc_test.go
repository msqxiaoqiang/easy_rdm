package tests

import (
	"testing"

	"easy_rdm/app/services"
	"easy_rdm/testsetup"
)

// ========== 收藏 ==========

func TestAPI_GetFavorites(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/get_favorites", map[string]interface{}{"conn_id": "c1", "db": 0})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestAPI_GetFavorites_MissingConnID(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/get_favorites", nil)
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_ToggleFavorite_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/toggle_favorite", map[string]string{"conn_id": "no-conn", "key": "k"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

// ========== 解码器 ==========

func TestAPI_GetDecoders(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/get_decoders", nil)
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestAPI_SaveAndDeleteDecoder(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/save_decoder", map[string]interface{}{
		"id": "d1", "name": "test-decoder", "type": "command", "command": "echo",
	})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("save_decoder failed: %+v", resp)
	}
	w = testsetup.PostJSON(router, "/api/delete_decoder", map[string]string{"id": "d1"})
	resp = testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("delete_decoder failed: %+v", resp)
	}
}

func TestAPI_DecodeValue_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/decode_value", map[string]string{
		"conn_id": "no-conn", "key": "k", "decoder_id": "d1",
	})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 && resp.Code != 400 {
		t.Fatalf("unexpected code %d", resp.Code)
	}
}

// ========== 操作日志 ==========

func TestAPI_GetOpLog(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/get_op_log", nil)
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestAPI_ClearOpLog(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/clear_op_log", nil)
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

// TestAPI_OpLog_AddAndGet 写入操作日志后能正确查询
func TestAPI_OpLog_AddAndGet(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	// 先清空
	testsetup.PostJSON(router, "/api/clear_op_log", nil)
	// 手动写入 3 条日志（通过 services 包的导出函数）
	services.AddOpLog("conn1", "SET", "mykey", "set value")
	services.AddOpLog("conn2", "DELETE", "otherkey", "deleted")
	services.AddOpLog("conn1", "RENAME", "mykey", "mykey → newkey")

	w := testsetup.PostJSON(router, "/api/get_op_log", nil)
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	entries, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("expected array, got %T", resp.Data)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

// TestAPI_OpLog_AfterFilter 使用 after 参数过滤
func TestAPI_OpLog_AfterFilter(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	testsetup.PostJSON(router, "/api/clear_op_log", nil)
	// 插入第 1 条，获取其 seq 作为分界线
	services.AddOpLog("c1", "SET", "k1", "d1")
	w := testsetup.PostJSON(router, "/api/get_op_log", nil)
	resp := testsetup.ParseResponse(t, w)
	firstEntries, _ := resp.Data.([]interface{})
	if len(firstEntries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(firstEntries))
	}
	firstEntry := firstEntries[0].(map[string]interface{})
	afterSeq := firstEntry["seq"].(float64)

	// 再插入 2 条
	services.AddOpLog("c1", "SET", "k2", "d2")
	services.AddOpLog("c1", "SET", "k3", "d3")

	// after=firstSeq 应只返回后 2 条
	w = testsetup.PostJSON(router, "/api/get_op_log", map[string]interface{}{"after": afterSeq})
	resp = testsetup.ParseResponse(t, w)
	entries, _ := resp.Data.([]interface{})
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries after seq=%.0f, got %d", afterSeq, len(entries))
	}
}

// TestAPI_OpLog_ConnIDFilter 按连接 ID 过滤
func TestAPI_OpLog_ConnIDFilter(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	testsetup.PostJSON(router, "/api/clear_op_log", nil)
	services.AddOpLog("conn-a", "SET", "k1", "d1")
	services.AddOpLog("conn-b", "DELETE", "k2", "d2")
	services.AddOpLog("conn-a", "GET", "k3", "d3")

	w := testsetup.PostJSON(router, "/api/get_op_log", map[string]string{"conn_id": "conn-a"})
	resp := testsetup.ParseResponse(t, w)
	entries, _ := resp.Data.([]interface{})
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries for conn-a, got %d", len(entries))
	}
}

// TestAPI_OpLog_LimitParam limit 参数限制返回数量
func TestAPI_OpLog_LimitParam(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	testsetup.PostJSON(router, "/api/clear_op_log", nil)
	for i := 0; i < 10; i++ {
		services.AddOpLog("c1", "SET", "k", "d")
	}

	// limit=3 应只返回最新 3 条
	w := testsetup.PostJSON(router, "/api/get_op_log", map[string]interface{}{"limit": 3})
	resp := testsetup.ParseResponse(t, w)
	entries, _ := resp.Data.([]interface{})
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries with limit=3, got %d", len(entries))
	}
}

// TestAPI_OpLog_ClearReturnsCount clear_op_log 返回已清除的条目数
func TestAPI_OpLog_ClearReturnsCount(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	testsetup.PostJSON(router, "/api/clear_op_log", nil)
	services.AddOpLog("c1", "SET", "k1", "d1")
	services.AddOpLog("c1", "SET", "k2", "d2")

	w := testsetup.PostJSON(router, "/api/clear_op_log", nil)
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	count, ok := resp.Data.(float64)
	if !ok {
		t.Fatalf("expected number, got %T: %v", resp.Data, resp.Data)
	}
	if int(count) != 2 {
		t.Fatalf("expected cleared 2, got %d", int(count))
	}
}
