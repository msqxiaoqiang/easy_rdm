package tests

import (
	"strings"
	"testing"

	"easy_rdm/testsetup"
)

// ========== 健康检查 ==========

func TestAPI_Ping_GET(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.GetJSON(router, "/api/ping")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 || resp.Data != "pong" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// ========== 连接配置 CRUD ==========

func TestAPI_SaveAndGetConnections(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	conn := map[string]interface{}{"id": "test-conn-1", "name": "Test Redis", "host": "127.0.0.1", "port": 6379}
	w := testsetup.PostJSON(router, "/api/save_connection", conn)
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("save_connection failed: %+v", resp)
	}
	w = testsetup.PostJSON(router, "/api/get_connections", nil)
	resp = testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("get_connections failed: %+v", resp)
	}
	conns, ok := resp.Data.([]interface{})
	if !ok || len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %v", resp.Data)
	}
}

func TestAPI_SaveConnection_PasswordEncryption(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	conn := map[string]interface{}{"id": "enc-test", "name": "Encrypted", "host": "127.0.0.1", "port": 6379, "password": "my-secret-password"}
	testsetup.PostJSON(router, "/api/save_connection", conn)
	w := testsetup.PostJSON(router, "/api/get_connections", nil)
	resp := testsetup.ParseResponse(t, w)
	conns := resp.Data.([]interface{})
	saved := conns[0].(map[string]interface{})
	if saved["password"] == "my-secret-password" {
		t.Fatal("password should be encrypted")
	}
	if saved["password_encrypted"] != true {
		t.Fatal("password_encrypted should be true")
	}
}

func TestAPI_SaveConnection_Update(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	conn := map[string]interface{}{"id": "upd-test", "name": "Original", "host": "127.0.0.1", "port": 6379}
	testsetup.PostJSON(router, "/api/save_connection", conn)
	conn["name"] = "Updated"
	testsetup.PostJSON(router, "/api/save_connection", conn)
	w := testsetup.PostJSON(router, "/api/get_connections", nil)
	resp := testsetup.ParseResponse(t, w)
	conns := resp.Data.([]interface{})
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection after update, got %d", len(conns))
	}
	if conns[0].(map[string]interface{})["name"] != "Updated" {
		t.Fatal("name should be updated")
	}
}

func TestAPI_DeleteConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	conn := map[string]interface{}{"id": "del-test", "name": "ToDelete", "host": "127.0.0.1", "port": 6379}
	testsetup.PostJSON(router, "/api/save_connection", conn)
	w := testsetup.PostJSON(router, "/api/delete_connection", map[string]string{"id": "del-test"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("delete failed: %+v", resp)
	}
	w = testsetup.PostJSON(router, "/api/get_connections", nil)
	resp = testsetup.ParseResponse(t, w)
	conns := resp.Data.([]interface{})
	if len(conns) != 0 {
		t.Fatalf("expected 0 connections after delete, got %d", len(conns))
	}
}

// ========== 设置 ==========

func TestAPI_SaveAndGetSettings(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	settings := map[string]interface{}{"theme": "dark", "language": "zh-CN", "font_size": 14}
	w := testsetup.PostJSON(router, "/api/save_settings", settings)
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("save_settings failed: %+v", resp)
	}
	w = testsetup.PostJSON(router, "/api/get_settings", nil)
	resp = testsetup.ParseResponse(t, w)
	data := resp.Data.(map[string]interface{})
	if data["theme"] != "dark" {
		t.Fatalf("expected theme 'dark', got %v", data["theme"])
	}
}

func TestAPI_GetSettings_Empty(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/get_settings", nil)
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

// ========== 会话 ==========

func TestAPI_SaveAndGetSession(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	session := map[string]interface{}{
		"tabs":        []map[string]string{{"id": "t1", "name": "Tab1"}},
		"activeTabId": "t1",
	}
	testsetup.PostJSON(router, "/api/save_session", session)
	w := testsetup.PostJSON(router, "/api/get_session", nil)
	resp := testsetup.ParseResponse(t, w)
	data := resp.Data.(map[string]interface{})
	if data["activeTabId"] != "t1" {
		t.Fatalf("expected activeTabId 't1', got %v", data["activeTabId"])
	}
}

// ========== 导入/导出连接 ==========

func TestAPI_ImportExportConnections(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	conn := map[string]interface{}{"id": "ie-test", "name": "ImportExport", "host": "127.0.0.1", "port": 6379}
	testsetup.PostJSON(router, "/api/save_connection", conn)

	w := testsetup.PostJSON(router, "/api/export_connections", nil)
	resp := testsetup.ParseResponse(t, w)
	exported := resp.Data.([]interface{})
	if len(exported) != 1 {
		t.Fatalf("expected 1 exported connection, got %d", len(exported))
	}
	ec := exported[0].(map[string]interface{})
	if _, ok := ec["password"]; ok {
		t.Fatal("exported connection should not contain password")
	}

	importData := map[string]interface{}{
		"connections": []map[string]interface{}{
			{"id": "imported-1", "name": "Imported", "host": "10.0.0.1", "port": 6379},
		},
	}
	w = testsetup.PostJSON(router, "/api/import_connections", importData)
	resp = testsetup.ParseResponse(t, w)
	data := resp.Data.(map[string]interface{})
	if data["imported"] != float64(1) {
		t.Fatalf("expected imported=1, got %v", data["imported"])
	}

	w = testsetup.PostJSON(router, "/api/import_connections", importData)
	resp = testsetup.ParseResponse(t, w)
	data = resp.Data.(map[string]interface{})
	if data["imported"] != float64(0) {
		t.Fatalf("expected imported=0 for duplicate, got %v", data["imported"])
	}
}

// ========== 参数校验 ==========

func TestAPI_Connect_MissingID(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/connect", map[string]string{"id": ""})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_Connect_NotFound(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/connect", map[string]string{"id": "nonexistent"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_ScanKeys_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/scan_keys", map[string]string{"conn_id": "no-such-conn"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_GetKeyValue_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/get_key_value", map[string]string{"conn_id": "no-conn", "key": "test"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_ExecuteCommand_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/execute_command", map[string]string{"conn_id": "no-conn", "command": "PING"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_ExecuteCommand_EmptyParams(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/execute_command", map[string]string{})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400 or 404 for empty params, got %d", resp.Code)
	}
}

func TestAPI_SaveSettings_BadJSON(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostRaw(router, "/api/save_settings", "not json")
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400 for bad JSON, got %d", resp.Code)
	}
}

// ========== create_key 参数校验 ==========

func TestAPI_CreateKey_EmptyKey(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/create_key", map[string]interface{}{
		"conn_id": "test", "key": "", "type": "string",
	})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400 for empty key, got %d: %s", resp.Code, resp.Msg)
	}
}

func TestAPI_CreateKey_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/create_key", map[string]interface{}{
		"conn_id": "no-conn", "key": "test", "type": "string",
	})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Msg)
	}
}

func TestAPI_CreateKey_UnsupportedType(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/create_key", map[string]interface{}{
		"conn_id": "no-conn", "key": "test", "type": "unknown_type",
	})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Msg)
	}
}

func TestAPI_CreateKey_AllTypesRecognized(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	types := []struct {
		name   string
		params map[string]interface{}
	}{
		{"string", map[string]interface{}{"conn_id": "c1", "key": "k", "type": "string", "value": "v"}},
		{"hash", map[string]interface{}{"conn_id": "c1", "key": "k", "type": "hash"}},
		{"list", map[string]interface{}{"conn_id": "c1", "key": "k", "type": "list"}},
		{"set", map[string]interface{}{"conn_id": "c1", "key": "k", "type": "set"}},
		{"zset", map[string]interface{}{"conn_id": "c1", "key": "k", "type": "zset"}},
		{"stream", map[string]interface{}{"conn_id": "c1", "key": "k", "type": "stream"}},
		{"bitmap", map[string]interface{}{"conn_id": "c1", "key": "k", "type": "bitmap", "bitmap_offset": 0}},
		{"hll", map[string]interface{}{"conn_id": "c1", "key": "k", "type": "hll", "hll_elements": []string{"a"}}},
		{"geo", map[string]interface{}{"conn_id": "c1", "key": "k", "type": "geo", "geo_members": []map[string]interface{}{{"longitude": 116.4, "latitude": 39.9, "member": "beijing"}}}},
	}
	for _, tc := range types {
		t.Run(tc.name, func(t *testing.T) {
			w := testsetup.PostJSON(router, "/api/create_key", tc.params)
			resp := testsetup.ParseResponse(t, w)
			// 合法类型应报"连接不存在"，不应报"不支持的类型"
			if resp.Code != 400 {
				t.Fatalf("type %q: expected 400, got %d: %s", tc.name, resp.Code, resp.Msg)
			}
			if strings.Contains(resp.Msg, "不支持的类型") {
				t.Fatalf("type %q should be supported, but got: %s", tc.name, resp.Msg)
			}
		})
	}
}
