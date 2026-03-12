package tests

import (
	"testing"

	"easy_rdm/testsetup"
)

// ========== Lua 脚本参数校验 ==========

func TestAPI_LuaEval_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/lua_eval", map[string]interface{}{"conn_id": "no-conn", "script": "return 1"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_LuaScriptsList_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/lua_scripts_list", nil)
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestAPI_LuaScriptSave(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/lua_script_save", map[string]interface{}{"id": "s1", "name": "test", "script": "return 1"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestAPI_LuaScriptDelete(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/lua_script_delete", map[string]string{"id": "s1"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}
