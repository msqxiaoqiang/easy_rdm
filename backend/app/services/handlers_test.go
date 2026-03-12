package services

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"easy_rdm/app/utils"

	"gopkg.in/yaml.v3"
)

// ========== handleReorderConnections 测试 ==========

// TestReorderConnections_BasicReorder 测试基本重排：连接按给定顺序排列，group 被更新
func TestReorderConnections_BasicReorder(t *testing.T) {
	setupTestStorage(t)

	// 准备初始连接数据：A, B, C 三个连接
	initial := []map[string]interface{}{
		{"id": "a", "name": "ConnA", "group": ""},
		{"id": "b", "name": "ConnB", "group": ""},
		{"id": "c", "name": "ConnC", "group": ""},
	}
	if err := WriteJSON("connections.json", initial); err != nil {
		t.Fatalf("写入初始数据失败: %v", err)
	}

	// 请求重排为 C, A, B，同时更新 group
	params, _ := json.Marshal(map[string]interface{}{
		"items": []map[string]string{
			{"id": "c", "group": "production"},
			{"id": "a", "group": "staging"},
			{"id": "b", "group": "dev"},
		},
	})

	result, err := handleReorderConnections(json.RawMessage(params))
	if err != nil {
		t.Fatalf("handleReorderConnections 返回错误: %v", err)
	}
	if result != nil {
		t.Fatalf("预期返回 nil，实际返回: %v", result)
	}

	// 验证持久化后的顺序和 group
	var conns []map[string]interface{}
	if err := ReadJSON("connections.json", &conns); err != nil {
		t.Fatalf("读取结果失败: %v", err)
	}

	if len(conns) != 3 {
		t.Fatalf("预期 3 个连接，实际 %d 个", len(conns))
	}

	// 验证顺序：C, A, B
	expectedOrder := []string{"c", "a", "b"}
	for i, expectedID := range expectedOrder {
		actualID, _ := conns[i]["id"].(string)
		if actualID != expectedID {
			t.Errorf("位置 %d: 预期 id=%s，实际 id=%s", i, expectedID, actualID)
		}
	}

	// 验证 group 更新
	expectedGroups := []string{"production", "staging", "dev"}
	for i, expectedGroup := range expectedGroups {
		actualGroup, _ := conns[i]["group"].(string)
		if actualGroup != expectedGroup {
			t.Errorf("位置 %d: 预期 group=%s，实际 group=%s", i, expectedGroup, actualGroup)
		}
	}

	// 验证其他字段保留
	if name, _ := conns[0]["name"].(string); name != "ConnC" {
		t.Errorf("连接 C 的 name 字段丢失，实际: %s", name)
	}
}

// TestReorderConnections_UpdateGroup 测试 group 字段正确更新
func TestReorderConnections_UpdateGroup(t *testing.T) {
	setupTestStorage(t)

	initial := []map[string]interface{}{
		{"id": "x", "name": "ConnX", "group": "old-group", "host": "localhost"},
	}
	if err := WriteJSON("connections.json", initial); err != nil {
		t.Fatalf("写入初始数据失败: %v", err)
	}

	params, _ := json.Marshal(map[string]interface{}{
		"items": []map[string]string{
			{"id": "x", "group": "new-group"},
		},
	})

	_, err := handleReorderConnections(json.RawMessage(params))
	if err != nil {
		t.Fatalf("handleReorderConnections 返回错误: %v", err)
	}

	var conns []map[string]interface{}
	ReadJSON("connections.json", &conns)

	if len(conns) != 1 {
		t.Fatalf("预期 1 个连接，实际 %d 个", len(conns))
	}
	if group, _ := conns[0]["group"].(string); group != "new-group" {
		t.Errorf("预期 group=new-group，实际 group=%s", group)
	}
	// 原有字段应保留
	if host, _ := conns[0]["host"].(string); host != "localhost" {
		t.Errorf("host 字段丢失，实际: %s", host)
	}
}

// TestReorderConnections_MissingIDsAppended 测试缺失的 ID 保底追加到末尾
func TestReorderConnections_MissingIDsAppended(t *testing.T) {
	setupTestStorage(t)

	// 有 A, B, C, D 四个连接
	initial := []map[string]interface{}{
		{"id": "a", "name": "ConnA", "group": ""},
		{"id": "b", "name": "ConnB", "group": ""},
		{"id": "c", "name": "ConnC", "group": ""},
		{"id": "d", "name": "ConnD", "group": ""},
	}
	if err := WriteJSON("connections.json", initial); err != nil {
		t.Fatalf("写入初始数据失败: %v", err)
	}

	// 只指定了 C 和 A，B 和 D 缺失
	params, _ := json.Marshal(map[string]interface{}{
		"items": []map[string]string{
			{"id": "c", "group": "g1"},
			{"id": "a", "group": "g2"},
		},
	})

	_, err := handleReorderConnections(json.RawMessage(params))
	if err != nil {
		t.Fatalf("handleReorderConnections 返回错误: %v", err)
	}

	var conns []map[string]interface{}
	ReadJSON("connections.json", &conns)

	if len(conns) != 4 {
		t.Fatalf("预期 4 个连接，实际 %d 个", len(conns))
	}

	// 前两个按指定顺序
	if id, _ := conns[0]["id"].(string); id != "c" {
		t.Errorf("位置 0: 预期 id=c，实际 id=%s", id)
	}
	if id, _ := conns[1]["id"].(string); id != "a" {
		t.Errorf("位置 1: 预期 id=a，实际 id=%s", id)
	}

	// 后两个是缺失的，按原顺序追加（B 在 D 前面）
	if id, _ := conns[2]["id"].(string); id != "b" {
		t.Errorf("位置 2: 预期 id=b，实际 id=%s", id)
	}
	if id, _ := conns[3]["id"].(string); id != "d" {
		t.Errorf("位置 3: 预期 id=d，实际 id=%s", id)
	}

	// 缺失的连接 group 不变
	if group, _ := conns[2]["group"].(string); group != "" {
		t.Errorf("缺失连接 B 的 group 不应被修改，实际: %s", group)
	}
}

// TestReorderConnections_InvalidParams 测试参数解析错误
func TestReorderConnections_InvalidParams(t *testing.T) {
	setupTestStorage(t)

	_, err := handleReorderConnections(json.RawMessage([]byte(`invalid json`)))
	if err == nil {
		t.Fatal("预期参数错误，实际未返回错误")
	}
}

// TestReorderConnections_EmptyItems 测试空 items：所有连接按原顺序追加
func TestReorderConnections_EmptyItems(t *testing.T) {
	setupTestStorage(t)

	initial := []map[string]interface{}{
		{"id": "a", "name": "ConnA"},
		{"id": "b", "name": "ConnB"},
	}
	WriteJSON("connections.json", initial)

	params, _ := json.Marshal(map[string]interface{}{
		"items": []map[string]string{},
	})

	_, err := handleReorderConnections(json.RawMessage(params))
	if err != nil {
		t.Fatalf("handleReorderConnections 返回错误: %v", err)
	}

	var conns []map[string]interface{}
	ReadJSON("connections.json", &conns)

	if len(conns) != 2 {
		t.Fatalf("预期 2 个连接，实际 %d 个", len(conns))
	}
	// 顺序不变
	if id, _ := conns[0]["id"].(string); id != "a" {
		t.Errorf("位置 0: 预期 id=a，实际 id=%s", id)
	}
	if id, _ := conns[1]["id"].(string); id != "b" {
		t.Errorf("位置 1: 预期 id=b，实际 id=%s", id)
	}
}

// TestReorderConnections_NonexistentIDIgnored 测试 items 中包含不存在的 ID，应被忽略
func TestReorderConnections_NonexistentIDIgnored(t *testing.T) {
	setupTestStorage(t)

	initial := []map[string]interface{}{
		{"id": "a", "name": "ConnA"},
		{"id": "b", "name": "ConnB"},
	}
	WriteJSON("connections.json", initial)

	params, _ := json.Marshal(map[string]interface{}{
		"items": []map[string]string{
			{"id": "b", "group": "g1"},
			{"id": "nonexistent", "group": "g2"},
			{"id": "a", "group": "g3"},
		},
	})

	_, err := handleReorderConnections(json.RawMessage(params))
	if err != nil {
		t.Fatalf("handleReorderConnections 返回错误: %v", err)
	}

	var conns []map[string]interface{}
	ReadJSON("connections.json", &conns)

	// 只有 2 个实际存在的连接
	if len(conns) != 2 {
		t.Fatalf("预期 2 个连接，实际 %d 个", len(conns))
	}
	if id, _ := conns[0]["id"].(string); id != "b" {
		t.Errorf("位置 0: 预期 id=b，实际 id=%s", id)
	}
	if id, _ := conns[1]["id"].(string); id != "a" {
		t.Errorf("位置 1: 预期 id=a，实际 id=%s", id)
	}
}

// ========== handleSaveGroups / handleGetGroups 测试 ==========

// TestSaveGroups_Basic 保存分组列表，验证 groups.json 写入正确
func TestSaveGroups_Basic(t *testing.T) {
	setupTestStorage(t)

	params, _ := json.Marshal(map[string]interface{}{
		"groups": []string{"production", "staging", "dev"},
	})

	result, err := handleSaveGroups(json.RawMessage(params))
	if err != nil {
		t.Fatalf("handleSaveGroups 返回错误: %v", err)
	}
	if result != nil {
		t.Fatalf("预期返回 nil，实际返回: %v", result)
	}

	// 验证持久化到 groups.json
	var groups []string
	if err := ReadJSON("groups.json", &groups); err != nil {
		t.Fatalf("读取 groups.json 失败: %v", err)
	}
	if len(groups) != 3 {
		t.Fatalf("预期 3 个分组，实际 %d 个", len(groups))
	}
	expected := []string{"production", "staging", "dev"}
	for i, exp := range expected {
		if groups[i] != exp {
			t.Errorf("位置 %d: 预期 %s，实际 %s", i, exp, groups[i])
		}
	}
}

// TestSaveGroups_EmptyList 保存空分组列表
func TestSaveGroups_EmptyList(t *testing.T) {
	setupTestStorage(t)

	params, _ := json.Marshal(map[string]interface{}{
		"groups": []string{},
	})

	result, err := handleSaveGroups(json.RawMessage(params))
	if err != nil {
		t.Fatalf("handleSaveGroups 返回错误: %v", err)
	}
	if result != nil {
		t.Fatalf("预期返回 nil，实际返回: %v", result)
	}

	var groups []string
	if err := ReadJSON("groups.json", &groups); err != nil {
		t.Fatalf("读取 groups.json 失败: %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("预期空列表，实际 %d 个分组", len(groups))
	}
}

// TestSaveGroups_InvalidParams 无效参数应返回错误
func TestSaveGroups_InvalidParams(t *testing.T) {
	setupTestStorage(t)

	_, err := handleSaveGroups(json.RawMessage([]byte(`invalid json`)))
	if err == nil {
		t.Fatal("预期参数错误，实际未返回错误")
	}
}

// TestGetGroups_Basic 读取已保存的分组
func TestGetGroups_Basic(t *testing.T) {
	setupTestStorage(t)

	// 先写入分组数据
	groups := []string{"g1", "g2", "g3"}
	if err := WriteJSON("groups.json", groups); err != nil {
		t.Fatalf("写入 groups.json 失败: %v", err)
	}

	result, err := handleGetGroups(nil)
	if err != nil {
		t.Fatalf("handleGetGroups 返回错误: %v", err)
	}

	got, ok := result.([]string)
	if !ok {
		t.Fatalf("预期返回 []string，实际类型: %T", result)
	}
	if len(got) != 3 {
		t.Fatalf("预期 3 个分组，实际 %d 个", len(got))
	}
	expected := []string{"g1", "g2", "g3"}
	for i, exp := range expected {
		if got[i] != exp {
			t.Errorf("位置 %d: 预期 %s，实际 %s", i, exp, got[i])
		}
	}
}

// ========== 导出/导入排序测试 ==========

// TestExportZip_ConnectionsFollowDisplayOrder 导出的连接按 groups.json 显示顺序排列，且保留 id
func TestExportZip_ConnectionsFollowDisplayOrder(t *testing.T) {
	dir := setupTestStorage(t)
	utils.InitCrypto("test-seed")

	// 3 个连接：conn_a（无分组）、conn_b（组 prod）、conn_c（组 prod）
	conns := []map[string]interface{}{
		{"id": "conn_a", "name": "A", "host": "a.com", "port": float64(6379), "group": ""},
		{"id": "conn_b", "name": "B", "host": "b.com", "port": float64(6379), "group": "prod"},
		{"id": "conn_c", "name": "C", "host": "c.com", "port": float64(6379), "group": "prod"},
	}
	WriteJSON("connections.json", conns)

	// 显示顺序：先 conn_a，然后空组 staging，然后 prod 组（含 conn_b, conn_c）
	groups := []string{"conn_a", "__group__staging", "__group__prod"}
	WriteJSON("groups.json", groups)

	// group_meta 映射分组 ID → 显示名称
	WriteJSON("group_meta.json", map[string]string{"prod": "Production", "staging": "Staging"})

	exportDir := filepath.Join(dir, "export_test")
	os.MkdirAll(exportDir, 0755)

	params, _ := json.Marshal(map[string]interface{}{
		"include_passwords": false,
		"export_path":       exportDir,
	})

	result, err := handleExportConnectionsZip(json.RawMessage(params))
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}

	// 从返回值获取实际 ZIP 路径
	resultMap := result.(map[string]interface{})
	zipPath := resultMap["path"].(string)
	yamlData := readYAMLFromZip(t, zipPath)

	// 验证连接顺序：A（未分组）, B（prod）, C（prod）
	if len(yamlData.Connections) != 3 {
		t.Fatalf("预期 3 个连接，实际 %d", len(yamlData.Connections))
	}

	expectedOrder := []string{"A", "B", "C"}
	for i, name := range expectedOrder {
		actual, _ := yamlData.Connections[i]["name"].(string)
		if actual != name {
			t.Errorf("位置 %d: 预期 name=%s，实际 name=%s", i, name, actual)
		}
	}

	// 验证 id 被保留
	for i, c := range yamlData.Connections {
		if _, hasID := c["id"]; !hasID {
			t.Errorf("位置 %d: 连接缺少 id 字段", i)
		}
	}

	// 验证 display_order 正确（过滤了孤立引用）
	if len(yamlData.DisplayOrder) != 3 {
		t.Fatalf("预期 3 个 display_order 条目，实际 %d: %v", len(yamlData.DisplayOrder), yamlData.DisplayOrder)
	}
	expectedDisplayOrder := []string{"conn_a", "__group__staging", "__group__prod"}
	for i, exp := range expectedDisplayOrder {
		if yamlData.DisplayOrder[i] != exp {
			t.Errorf("display_order[%d]: 预期 %s，实际 %s", i, exp, yamlData.DisplayOrder[i])
		}
	}

	// 验证 group_meta 正确
	if yamlData.GroupMeta["prod"] != "Production" {
		t.Errorf("group_meta[prod] should be Production, got %s", yamlData.GroupMeta["prod"])
	}
	if yamlData.GroupMeta["staging"] != "Staging" {
		t.Errorf("group_meta[staging] should be Staging, got %s", yamlData.GroupMeta["staging"])
	}
}

// TestExportZip_OrphanGroupsFiltered 孤立的 ID 引用在导出时被过滤
func TestExportZip_OrphanGroupsFiltered(t *testing.T) {
	dir := setupTestStorage(t)
	utils.InitCrypto("test-seed")

	conns := []map[string]interface{}{
		{"id": "conn_a", "name": "A", "host": "a.com", "port": float64(6379), "group": ""},
	}
	WriteJSON("connections.json", conns)

	// groups 中有一个已删除的连接 ID "deleted_id"
	groups := []string{"conn_a", "deleted_id", "__group__empty"}
	WriteJSON("groups.json", groups)

	// group_meta 映射
	WriteJSON("group_meta.json", map[string]string{"empty": "Empty"})

	exportDir := filepath.Join(dir, "export_test")
	os.MkdirAll(exportDir, 0755)

	params, _ := json.Marshal(map[string]interface{}{
		"include_passwords": false,
		"export_path":       exportDir,
	})

	result, err := handleExportConnectionsZip(json.RawMessage(params))
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}

	resultMap := result.(map[string]interface{})
	zipPath := resultMap["path"].(string)
	yamlData := readYAMLFromZip(t, zipPath)

	// display_order 应只有 conn_a 和 __group__empty，deleted_id 被过滤
	if len(yamlData.DisplayOrder) != 2 {
		t.Fatalf("预期 2 个 display_order 条目，实际 %d: %v", len(yamlData.DisplayOrder), yamlData.DisplayOrder)
	}
	if yamlData.DisplayOrder[0] != "conn_a" || yamlData.DisplayOrder[1] != "__group__empty" {
		t.Errorf("display_order 过滤不正确: %v", yamlData.DisplayOrder)
	}
}

// TestImportZip_AppendToEnd 导入的连接和分组追加到已有数据末尾
func TestImportZip_AppendToEnd(t *testing.T) {
	dir := setupTestStorage(t)
	utils.InitCrypto("test-seed")

	// 已有数据：1 个连接 + 1 个分组
	existingConns := []map[string]interface{}{
		{"id": "existing1", "name": "Existing", "host": "e.com", "port": float64(6379), "group": ""},
	}
	WriteJSON("connections.json", existingConns)
	WriteJSON("groups.json", []string{"existing1", "__group__mygroup"})
	WriteJSON("group_meta.json", map[string]string{"mygroup": "My Group"})

	// 构造导入的 YAML（2 个连接 + 分组信息，新格式）
	importYAML := map[string]interface{}{
		"group_meta":    map[string]string{"newgroup": "New Group"},
		"display_order": []string{"import_a", "__group__newgroup", "import_b"},
		"connections": []map[string]interface{}{
			{"id": "import_a", "name": "ImportA", "host": "ia.com", "port": 6379, "group": ""},
			{"id": "import_b", "name": "ImportB", "host": "ib.com", "port": 6379, "group": "newgroup"},
		},
	}

	zipPath := createTestZip(t, dir, importYAML)

	params, _ := json.Marshal(map[string]interface{}{"file_path": zipPath})
	result, err := handleImportConnectionsZip(json.RawMessage(params))
	if err != nil {
		t.Fatalf("导入失败: %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["imported"] != 2 {
		t.Fatalf("预期导入 2 个，实际 %v", resultMap["imported"])
	}

	// 验证 connections.json：已有的在前，导入的在后
	var conns []map[string]interface{}
	ReadJSON("connections.json", &conns)

	if len(conns) != 3 {
		t.Fatalf("预期 3 个连接，实际 %d", len(conns))
	}
	if name, _ := conns[0]["name"].(string); name != "Existing" {
		t.Errorf("位置 0 应为 Existing，实际 %s", name)
	}
	if name, _ := conns[1]["name"].(string); name != "ImportA" {
		t.Errorf("位置 1 应为 ImportA，实际 %s", name)
	}
	if name, _ := conns[2]["name"].(string); name != "ImportB" {
		t.Errorf("位置 2 应为 ImportB，实际 %s", name)
	}

	// 验证导入的连接有新 ID（不是旧 ID）
	importAID, _ := conns[1]["id"].(string)
	importBID, _ := conns[2]["id"].(string)
	if importAID == "import_a" {
		t.Error("导入连接 A 的 ID 应被替换为新 ID")
	}
	if importBID == "import_b" {
		t.Error("导入连接 B 的 ID 应被替换为新 ID")
	}

	// 验证 groups.json：已有的在前，导入的追加在后，ID 已映射
	var groups []string
	ReadJSON("groups.json", &groups)

	// 已有: existing1, __group__mygroup
	// 导入追加: 新ID_A, __group__新分组ID, 新ID_B
	if len(groups) != 5 {
		t.Fatalf("预期 5 个 groups 条目，实际 %d: %v", len(groups), groups)
	}
	if groups[0] != "existing1" {
		t.Errorf("groups[0] 应为 existing1，实际 %s", groups[0])
	}
	if groups[1] != "__group__mygroup" {
		t.Errorf("groups[1] 应为 __group__mygroup，实际 %s", groups[1])
	}
	// groups[2] 应为 import_a 的新 ID
	if groups[2] != importAID {
		t.Errorf("groups[2] 应为 ImportA 的新 ID %s，实际 %s", importAID, groups[2])
	}
	// groups[3] 应为新分组 key（导入的 newgroup 被映射为新 grp_ ID）
	if !strings.HasPrefix(groups[3], "__group__grp_") {
		t.Errorf("groups[3] should be remapped group ID, got %s", groups[3])
	}
	// groups[4] 应为 import_b 的新 ID
	if groups[4] != importBID {
		t.Errorf("groups[4] 应为 ImportB 的新 ID %s，实际 %s", importBID, groups[4])
	}

	// 验证 group_meta.json 被正确更新
	var meta map[string]string
	ReadJSON("group_meta.json", &meta)

	// 原有的 mygroup 应保留
	if meta["mygroup"] != "My Group" {
		t.Errorf("existing group_meta[mygroup] should be preserved, got %s", meta["mygroup"])
	}
	// 应有一个新的 grp_ 开头的分组映射到 "New Group"
	foundNewGroup := false
	for id, name := range meta {
		if strings.HasPrefix(id, "grp_") && name == "New Group" {
			foundNewGroup = true
			break
		}
	}
	if !foundNewGroup {
		t.Errorf("group_meta should contain new group with name 'New Group', got %v", meta)
	}
}

// TestImportZip_SameNameGroupCreatesNewID 导入的分组即使与已有同名也会创建新 ID
func TestImportZip_SameNameGroupCreatesNewID(t *testing.T) {
	setupTestStorage(t)
	utils.InitCrypto("test-seed")

	existingConns := []map[string]interface{}{
		{"id": "e1", "name": "E1", "host": "e.com", "port": float64(6379), "group": "grp_existing"},
	}
	WriteJSON("connections.json", existingConns)
	WriteJSON("groups.json", []string{"__group__grp_existing", "e1"})
	WriteJSON("group_meta.json", map[string]string{"grp_existing": "Production"})

	// Import data also has a group named "Production"
	importYAML := map[string]interface{}{
		"group_meta":    map[string]string{"prod": "Production"},
		"display_order": []string{"__group__prod", "import1"},
		"connections": []map[string]interface{}{
			{"id": "import1", "name": "Import1", "host": "i.com", "port": 6379, "group": "prod"},
		},
	}

	dir := t.TempDir()
	zipPath := createTestZip(t, dir, importYAML)

	params, _ := json.Marshal(map[string]interface{}{"file_path": zipPath})
	_, err := handleImportConnectionsZip(json.RawMessage(params))
	if err != nil {
		t.Fatalf("导入失败: %v", err)
	}

	// Both groups should exist in group_meta with different IDs
	var meta map[string]string
	ReadJSON("group_meta.json", &meta)

	if meta["grp_existing"] != "Production" {
		t.Errorf("existing group meta should be preserved, got %s", meta["grp_existing"])
	}

	// There should be a new group ID mapping for the imported "prod" group
	foundNewGroup := false
	for id, name := range meta {
		if id != "grp_existing" && name == "Production" {
			foundNewGroup = true
			break
		}
	}
	if !foundNewGroup {
		t.Error("imported group should create new ID with same display name")
	}
}

// --- 测试辅助函数 ---

type exportedYAML struct {
	GroupMeta    map[string]string        `yaml:"group_meta"`
	DisplayOrder []string                 `yaml:"display_order"`
	Connections  []map[string]interface{} `yaml:"connections"`
}

func readYAMLFromZip(t *testing.T, zipPath string) exportedYAML {
	t.Helper()
	data, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatalf("读取 ZIP 失败: %v", err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("解析 ZIP 失败: %v", err)
	}
	for _, f := range reader.File {
		if f.Name == "connections.yaml" {
			rc, _ := f.Open()
			content, _ := io.ReadAll(rc)
			rc.Close()
			var result exportedYAML
			if err := yaml.Unmarshal(content, &result); err != nil {
				t.Fatalf("YAML 解析失败: %v", err)
			}
			return result
		}
	}
	t.Fatal("ZIP 中未找到 connections.yaml")
	return exportedYAML{}
}

func createTestZip(t *testing.T, dir string, yamlData interface{}) string {
	t.Helper()
	yamlBytes, err := yaml.Marshal(yamlData)
	if err != nil {
		t.Fatalf("YAML 序列化失败: %v", err)
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("connections.yaml")
	f.Write(yamlBytes)
	w.Close()

	zipPath := filepath.Join(dir, "test_import.zip")
	os.WriteFile(zipPath, buf.Bytes(), 0644)
	return zipPath
}

// TestGetGroups_FileNotExist groups.json 不存在时返回空数组
func TestGetGroups_FileNotExist(t *testing.T) {
	// 使用临时目录但不创建 groups.json
	dir := t.TempDir()
	InitStorage(dir)
	// 手动删除 groups.json（如果 InitStorage 创建了它）
	os.Remove(filepath.Join(dir, "groups.json"))

	result, err := handleGetGroups(nil)
	if err != nil {
		t.Fatalf("handleGetGroups 应在文件不存在时返回空数组，实际返回错误: %v", err)
	}

	got, ok := result.([]string)
	if !ok {
		t.Fatalf("预期返回 []string，实际类型: %T", result)
	}
	if len(got) != 0 {
		t.Fatalf("预期空数组，实际 %d 个分组", len(got))
	}
}

// TestInitStorage_CreatesGroupsFile 验证 InitStorage 创建 groups.json
func TestInitStorage_CreatesGroupsFile(t *testing.T) {
	dir := t.TempDir()
	InitStorage(dir)

	p := filepath.Join(dir, "groups.json")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Fatalf("groups.json 应在 InitStorage 后存在")
	}

	// 验证默认内容是空数组
	var groups []string
	if err := ReadJSON("groups.json", &groups); err != nil {
		t.Fatalf("读取 groups.json 失败: %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("预期默认为空数组，实际 %d 个分组", len(groups))
	}
}
