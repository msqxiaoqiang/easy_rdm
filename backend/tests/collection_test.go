package tests

import (
	"testing"

	"easy_rdm/testsetup"
)

// ========== Hash 参数校验 ==========

func TestAPI_HScanFields_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/hscan_fields", map[string]string{"conn_id": "no-conn", "key": "k"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_HSetField_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/hset_field", map[string]string{"conn_id": "no-conn", "key": "k", "field": "f", "value": "v"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_HDelFields_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/hdel_fields", map[string]interface{}{"conn_id": "no-conn", "key": "k", "fields": []string{"f1"}})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

// ========== List 参数校验 ==========

func TestAPI_LRangeValues_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/lrange_values", map[string]string{"conn_id": "no-conn", "key": "k"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_ListPush_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/list_push", map[string]interface{}{"conn_id": "no-conn", "key": "k", "values": []string{"v1"}, "direction": "left"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_ListRemove_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/list_remove", map[string]interface{}{"conn_id": "no-conn", "key": "k", "value": "v", "count": 1})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

// ========== Set 参数校验 ==========

func TestAPI_SScanMembers_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/sscan_members", map[string]string{"conn_id": "no-conn", "key": "k"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_SAddMembers_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/sadd_members", map[string]interface{}{"conn_id": "no-conn", "key": "k", "members": []string{"m1"}})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_SRemMembers_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/srem_members", map[string]interface{}{"conn_id": "no-conn", "key": "k", "members": []string{"m1"}})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

// ========== ZSet 参数校验 ==========

func TestAPI_ZScanMembers_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/zscan_members", map[string]string{"conn_id": "no-conn", "key": "k"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_ZAddMembers_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/zadd_members", map[string]interface{}{"conn_id": "no-conn", "key": "k", "members": []map[string]interface{}{{"member": "m1", "score": 1.0}}})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_ZRemMembers_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/zrem_members", map[string]interface{}{"conn_id": "no-conn", "key": "k", "members": []string{"m1"}})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

// ========== 参数解析错误 ==========

func TestAPI_HScanFields_BadJSON(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostRaw(router, "/api/hscan_fields", "not json")
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}
