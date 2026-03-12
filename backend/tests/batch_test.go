package tests

import (
	"testing"

	"easy_rdm/testsetup"
)

// ========== 批量操作参数校验 ==========

func TestAPI_BatchSetTTL_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/batch_set_ttl", map[string]interface{}{"conn_id": "no-conn", "keys": []string{"k1"}, "ttl": 60})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_BatchMoveDB_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/batch_move_db", map[string]interface{}{"conn_id": "no-conn", "keys": []string{"k1"}, "target_db": 1})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_MigrateKeys_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/migrate_keys", map[string]interface{}{"conn_id": "no-conn", "keys": []string{"k1"}, "target_conn_id": "no-conn2"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}
