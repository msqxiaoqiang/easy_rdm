//go:build integration

package tests

import (
	"strconv"
	"testing"

	"easy_rdm/app/utils"
	"easy_rdm/testsetup"

	"github.com/gin-gonic/gin"
)

// setupIntegrationRouter 启动 Redis 容器并返回已连接的 router 和 connID
func setupIntegrationRouter(t *testing.T) (*gin.Engine, string) {
	t.Helper()
	rc := StartRedis(t)
	router := testsetup.SetupTestRouter(t)
	port, _ := strconv.Atoi(rc.Port)
	connID := "int-test"

	testsetup.PostJSON(router, "/api/save_connection", map[string]interface{}{
		"id": connID, "name": "Integration", "host": rc.Host, "port": port,
	})
	w := testsetup.PostJSON(router, "/api/connect", map[string]string{"id": connID})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 200 {
		t.Fatalf("connect failed: %+v", resp)
	}
	return router, connID
}

func assertOK(t *testing.T, resp utils.BizResponse, label string) {
	t.Helper()
	if resp.Code != 200 {
		t.Fatalf("%s failed: code=%d msg=%s", label, resp.Code, resp.Msg)
	}
}

func TestIntegration_ConnectAndPing(t *testing.T) {
	router, connID := setupIntegrationRouter(t)
	w := testsetup.PostJSON(router, "/api/execute_command", map[string]string{"conn_id": connID, "command": "PING"})
	resp := testsetup.ParseResponse(t, w)
	assertOK(t, resp, "PING")
}

func TestIntegration_StringCRUD(t *testing.T) {
	router, connID := setupIntegrationRouter(t)

	// create_key string
	w := testsetup.PostJSON(router, "/api/create_key", map[string]interface{}{
		"conn_id": connID, "key": "test:str", "type": "string", "value": "hello",
	})
	assertOK(t, testsetup.ParseResponse(t, w), "create_key")

	// get_key_value
	w = testsetup.PostJSON(router, "/api/get_key_value", map[string]string{"conn_id": connID, "key": "test:str"})
	resp := testsetup.ParseResponse(t, w)
	assertOK(t, resp, "get_key_value")

	// set_key_value
	w = testsetup.PostJSON(router, "/api/set_key_value", map[string]string{
		"conn_id": connID, "key": "test:str", "value": "world",
	})
	assertOK(t, testsetup.ParseResponse(t, w), "set_key_value")

	// rename_key
	w = testsetup.PostJSON(router, "/api/rename_key", map[string]string{
		"conn_id": connID, "key": "test:str", "new_key": "test:str2",
	})
	assertOK(t, testsetup.ParseResponse(t, w), "rename_key")

	// set_ttl
	w = testsetup.PostJSON(router, "/api/set_ttl", map[string]interface{}{
		"conn_id": connID, "key": "test:str2", "ttl": 300,
	})
	assertOK(t, testsetup.ParseResponse(t, w), "set_ttl")

	// delete_keys
	w = testsetup.PostJSON(router, "/api/delete_keys", map[string]interface{}{
		"conn_id": connID, "keys": []string{"test:str2"},
	})
	assertOK(t, testsetup.ParseResponse(t, w), "delete_keys")
}

func TestIntegration_HashCRUD(t *testing.T) {
	router, connID := setupIntegrationRouter(t)

	w := testsetup.PostJSON(router, "/api/create_key", map[string]interface{}{
		"conn_id": connID, "key": "test:hash", "type": "hash",
	})
	assertOK(t, testsetup.ParseResponse(t, w), "create_key hash")

	w = testsetup.PostJSON(router, "/api/hset_field", map[string]string{
		"conn_id": connID, "key": "test:hash", "field": "f1", "value": "v1",
	})
	assertOK(t, testsetup.ParseResponse(t, w), "hset_field")

	w = testsetup.PostJSON(router, "/api/hscan_fields", map[string]string{
		"conn_id": connID, "key": "test:hash",
	})
	assertOK(t, testsetup.ParseResponse(t, w), "hscan_fields")

	w = testsetup.PostJSON(router, "/api/hdel_fields", map[string]interface{}{
		"conn_id": connID, "key": "test:hash", "fields": []string{"f1"},
	})
	assertOK(t, testsetup.ParseResponse(t, w), "hdel_fields")
}

func TestIntegration_ListCRUD(t *testing.T) {
	router, connID := setupIntegrationRouter(t)

	w := testsetup.PostJSON(router, "/api/create_key", map[string]interface{}{
		"conn_id": connID, "key": "test:list", "type": "list",
	})
	assertOK(t, testsetup.ParseResponse(t, w), "create_key list")

	w = testsetup.PostJSON(router, "/api/list_push", map[string]interface{}{
		"conn_id": connID, "key": "test:list", "values": []string{"a", "b"}, "direction": "right",
	})
	assertOK(t, testsetup.ParseResponse(t, w), "list_push")

	w = testsetup.PostJSON(router, "/api/lrange_values", map[string]string{
		"conn_id": connID, "key": "test:list",
	})
	assertOK(t, testsetup.ParseResponse(t, w), "lrange_values")

	w = testsetup.PostJSON(router, "/api/list_remove", map[string]interface{}{
		"conn_id": connID, "key": "test:list", "value": "a", "count": 1,
	})
	assertOK(t, testsetup.ParseResponse(t, w), "list_remove")
}

func TestIntegration_SetCRUD(t *testing.T) {
	router, connID := setupIntegrationRouter(t)

	w := testsetup.PostJSON(router, "/api/create_key", map[string]interface{}{
		"conn_id": connID, "key": "test:set", "type": "set",
	})
	assertOK(t, testsetup.ParseResponse(t, w), "create_key set")

	w = testsetup.PostJSON(router, "/api/sadd_members", map[string]interface{}{
		"conn_id": connID, "key": "test:set", "members": []string{"m1", "m2"},
	})
	assertOK(t, testsetup.ParseResponse(t, w), "sadd_members")

	w = testsetup.PostJSON(router, "/api/sscan_members", map[string]string{
		"conn_id": connID, "key": "test:set",
	})
	assertOK(t, testsetup.ParseResponse(t, w), "sscan_members")

	w = testsetup.PostJSON(router, "/api/srem_members", map[string]interface{}{
		"conn_id": connID, "key": "test:set", "members": []string{"m1"},
	})
	assertOK(t, testsetup.ParseResponse(t, w), "srem_members")
}

func TestIntegration_ZSetCRUD(t *testing.T) {
	router, connID := setupIntegrationRouter(t)

	w := testsetup.PostJSON(router, "/api/create_key", map[string]interface{}{
		"conn_id": connID, "key": "test:zset", "type": "zset",
	})
	assertOK(t, testsetup.ParseResponse(t, w), "create_key zset")

	w = testsetup.PostJSON(router, "/api/zadd_members", map[string]interface{}{
		"conn_id": connID, "key": "test:zset",
		"members": []map[string]interface{}{{"member": "z1", "score": 1.0}},
	})
	assertOK(t, testsetup.ParseResponse(t, w), "zadd_members")

	w = testsetup.PostJSON(router, "/api/zscan_members", map[string]string{
		"conn_id": connID, "key": "test:zset",
	})
	assertOK(t, testsetup.ParseResponse(t, w), "zscan_members")

	w = testsetup.PostJSON(router, "/api/zrem_members", map[string]interface{}{
		"conn_id": connID, "key": "test:zset", "members": []string{"z1"},
	})
	assertOK(t, testsetup.ParseResponse(t, w), "zrem_members")
}
