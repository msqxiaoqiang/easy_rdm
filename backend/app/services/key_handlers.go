package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"easy_rdm/app/consts"
	"easy_rdm/app/utils"

	"github.com/go-redis/redis/v8"
	"gopkg.in/yaml.v3"
)

// RegisterKeyHandlers 注册 Key 操作相关的 RPC 方法
func RegisterKeyHandlers(register func(string, RPCHandlerFunc)) {
	register("scan_keys", handleScanKeys)
	register("scan_tree_level", handleScanTreeLevel)
	register("scan_pattern_keys", handleScanPatternKeys)
	register("get_key_info", handleGetKeyInfo)
	register("get_key_value", handleGetKeyValue)
	register("set_key_value", handleSetKeyValue)
	register("delete_keys", handleDeleteKeys)
	register("rename_key", handleRenameKey)
	register("set_ttl", handleSetTTL)
	register("create_key", handleCreateKey)
	register("check_key_exists", handleCheckKeyExists)
	register("select_db", handleSelectDB)
	register("get_db_list", handleGetDBList)
	register("cross_db_search", handleCrossDbSearch)
	register("get_server_status", handleGetServerStatus)
	register("execute_command", handleExecuteCommand)
	register("import_connections", handleImportConnections)
	register("export_connections", handleExportConnections)
	register("export_connections_zip", handleExportConnectionsZip)
	register("import_connections_zip", handleImportConnectionsZip)
	register("flush_db", handleFlushDB)
	register("copy_as_command", handleCopyAsCommand)
}

// ========== Key 扫描与信息 ==========

func handleScanKeys(params json.RawMessage) (any, error) {
	var req struct {
		ConnID  string `json:"conn_id"`
		Pattern string `json:"pattern"`
		Count   int64  `json:"count"`
		Cursor  uint64 `json:"cursor"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	if req.Pattern == "" {
		req.Pattern = "*"
	}
	if req.Count <= 0 {
		req.Count = 200
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	keys, nextCursor, err := conn.Cmd().Scan(ctx, req.Cursor, req.Pattern, req.Count).Result()
	if err != nil {
		return nil, err
	}

	// 批量获取 type 和 TTL
	type KeyItem struct {
		Key  string `json:"key"`
		Type string `json:"type"`
		TTL  int64  `json:"ttl"` // -1=永久, -2=不存在, 其他=秒数
	}

	items := make([]KeyItem, 0, len(keys))
	if len(keys) > 0 {
		pipe := conn.Cmd().Pipeline()
		typeCmds := make([]*redis.StatusCmd, len(keys))
		ttlCmds := make([]*redis.DurationCmd, len(keys))
		for i, key := range keys {
			typeCmds[i] = pipe.Type(ctx, key)
			ttlCmds[i] = pipe.TTL(ctx, key)
		}
		pipe.Exec(ctx)

		for i, key := range keys {
			ttlVal := int64(-1)
			if d := ttlCmds[i].Val(); d == -1 {
				ttlVal = -1 // 永久
			} else if d == -2 {
				ttlVal = -2 // 不存在
			} else {
				ttlVal = int64(d.Seconds())
			}
			keyType := typeCmds[i].Val()
			items = append(items, KeyItem{
				Key:  key,
				Type: keyType,
				TTL:  ttlVal,
			})
		}
	}

	AddOpLog(req.ConnID, "SCAN", "", fmt.Sprintf("pattern=%s count=%d", req.Pattern, len(items)))
	return map[string]interface{}{
		"keys":   items,
		"cursor": nextCursor,
	}, nil
}

// handleScanTreeLevel 按前缀聚合扫描，返回当前层的分组（含 count）和叶子 key
// 用于树形视图懒加载：每次只加载一层，展开分组时再请求下一层
func handleScanTreeLevel(params json.RawMessage) (any, error) {
	var req struct {
		ConnID    string `json:"conn_id"`
		Prefix    string `json:"prefix"`    // 当前层前缀，根层为 ""
		Separator string `json:"separator"` // 分隔符，默认 ":"
		MaxScan   int    `json:"max_scan"`  // 最大扫描 key 数，默认 10000
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	if req.Separator == "" {
		req.Separator = ":"
	}
	if req.MaxScan <= 0 || req.MaxScan > 50000 {
		req.MaxScan = 10000
	}

	pattern := req.Prefix + "*"
	prefixLen := len(req.Prefix)

	ctx, cancel := context.WithTimeout(conn.Ctx, 30*time.Second)
	defer cancel()

	// 聚合：groupCounts 记录子分组的 key 计数，leafKeys 记录当前层叶子 key 名
	groupCounts := make(map[string]int)
	var leafKeys []string
	scanned := 0
	complete := false

	var cursor uint64
	for {
		keys, next, err := conn.Cmd().Scan(ctx, cursor, pattern, 500).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			scanned++
			rest := key[prefixLen:] // prefix 之后的部分
			idx := strings.Index(rest, req.Separator)
			if idx >= 0 {
				// 有分隔符 → 属于子分组
				groupLabel := rest[:idx]
				groupPrefix := req.Prefix + groupLabel + req.Separator
				groupCounts[groupPrefix]++
			} else {
				// 无分隔符 → 叶子 key
				leafKeys = append(leafKeys, key)
			}
		}

		cursor = next
		if cursor == 0 {
			complete = true
			break
		}
		if scanned >= req.MaxScan {
			break
		}
	}

	// 构建分组列表
	type GroupItem struct {
		Prefix string `json:"prefix"`
		Label  string `json:"label"`
		Count  int    `json:"count"`
	}
	groups := make([]GroupItem, 0, len(groupCounts))
	for gp, cnt := range groupCounts {
		label := gp[prefixLen : len(gp)-len(req.Separator)]
		groups = append(groups, GroupItem{Prefix: gp, Label: label, Count: cnt})
	}

	// 批量获取叶子 key 的 type 和 TTL
	type KeyItem struct {
		Key  string `json:"key"`
		Type string `json:"type"`
		TTL  int64  `json:"ttl"`
	}
	items := make([]KeyItem, 0, len(leafKeys))
	if len(leafKeys) > 0 {
		pipe := conn.Cmd().Pipeline()
		typeCmds := make([]*redis.StatusCmd, len(leafKeys))
		ttlCmds := make([]*redis.DurationCmd, len(leafKeys))
		for i, key := range leafKeys {
			typeCmds[i] = pipe.Type(ctx, key)
			ttlCmds[i] = pipe.TTL(ctx, key)
		}
		pipe.Exec(ctx)

		for i, key := range leafKeys {
			ttlVal := int64(-1)
			if d := ttlCmds[i].Val(); d == -1 {
				ttlVal = -1
			} else if d == -2 {
				ttlVal = -2
			} else {
				ttlVal = int64(d.Seconds())
			}
			items = append(items, KeyItem{Key: key, Type: typeCmds[i].Val(), TTL: ttlVal})
		}
	}

	AddOpLog(req.ConnID, "SCAN_TREE", "", fmt.Sprintf("prefix=%q groups=%d keys=%d scanned=%d", req.Prefix, len(groups), len(items), scanned))
	return map[string]interface{}{
		"groups":   groups,
		"keys":     items,
		"scanned":  scanned,
		"complete": complete,
	}, nil
}

// handleScanPatternKeys 扫描所有匹配 pattern 的 key（用于分组删除预览）
func handleScanPatternKeys(params json.RawMessage) (any, error) {
	var req struct {
		ConnID  string `json:"conn_id"`
		Pattern string `json:"pattern"`
		Limit   int    `json:"limit"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	if req.Pattern == "" {
		return nil, fmt.Errorf("pattern 不能为空")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	if req.Limit <= 0 || req.Limit > 10000 {
		req.Limit = 10000
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 30*time.Second)
	defer cancel()

	var allKeys []string
	var cursor uint64
	for {
		keys, next, err := conn.Cmd().Scan(ctx, cursor, req.Pattern, 500).Result()
		if err != nil {
			return nil, err
		}
		allKeys = append(allKeys, keys...)
		if len(allKeys) >= req.Limit {
			allKeys = allKeys[:req.Limit]
			break
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}

	AddOpLog(req.ConnID, "SCAN_PATTERN", "", fmt.Sprintf("pattern=%s found=%d", req.Pattern, len(allKeys)))
	return map[string]interface{}{
		"keys":  allKeys,
		"total": len(allKeys),
	}, nil
}

func handleGetKeyInfo(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	pipe := conn.Cmd().Pipeline()
	typeCmd := pipe.Type(ctx, req.Key)
	ttlCmd := pipe.TTL(ctx, req.Key)
	encodingCmd := pipe.ObjectEncoding(ctx, req.Key)
	pipe.Exec(ctx)

	ttlVal := int64(-1)
	if d := ttlCmd.Val(); d == -1 {
		ttlVal = -1
	} else if d == -2 {
		ttlVal = -2
	} else {
		ttlVal = int64(d.Seconds())
	}

	// 获取长度（根据类型）
	keyType := typeCmd.Val()
	var length int64
	switch keyType {
	case "string":
		length, _ = conn.Cmd().StrLen(ctx, req.Key).Result()
	case "list":
		length, _ = conn.Cmd().LLen(ctx, req.Key).Result()
	case "set":
		length, _ = conn.Cmd().SCard(ctx, req.Key).Result()
	case "zset":
		length, _ = conn.Cmd().ZCard(ctx, req.Key).Result()
	case "hash":
		length, _ = conn.Cmd().HLen(ctx, req.Key).Result()
	case "stream":
		length, _ = conn.Cmd().XLen(ctx, req.Key).Result()
	}

	// 获取内存占用（MEMORY USAGE）
	var memoryUsage int64
	memResult, memErr := conn.Cmd().MemoryUsage(ctx, req.Key).Result()
	if memErr == nil {
		memoryUsage = memResult
	}

	AddOpLog(req.ConnID, "KEY_INFO", req.Key, "type="+keyType)
	return map[string]interface{}{
		"key":          req.Key,
		"type":         keyType,
		"ttl":          ttlVal,
		"encoding":     encodingCmd.Val(),
		"length":       length,
		"memory_usage": memoryUsage,
	}, nil
}

// ========== Key 值操作 ==========

func handleGetKeyValue(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	keyType, err := conn.Cmd().Type(ctx, req.Key).Result()
	if err != nil {
		return nil, err
	}

	var value interface{}
	switch keyType {
	case "string":
		value, err = conn.Cmd().Get(ctx, req.Key).Result()
	case "list":
		value, err = conn.Cmd().LRange(ctx, req.Key, 0, -1).Result()
	case "set":
		value, err = conn.Cmd().SMembers(ctx, req.Key).Result()
	case "zset":
		value, err = conn.Cmd().ZRangeWithScores(ctx, req.Key, 0, -1).Result()
	case "hash":
		value, err = conn.Cmd().HGetAll(ctx, req.Key).Result()
	case "stream":
		value, err = conn.Cmd().XRange(ctx, req.Key, "-", "+").Result()
	case consts.TypeJSON:
		// RedisJSON module: use JSON.GET to retrieve the full document
		var jsonStr string
		jsonStr, err = conn.Do(ctx, "JSON.GET", req.Key, ".").Text()
		if err == nil {
			value = jsonStr
		}
		keyType = "ReJSON-RL" // preserve original type for frontend
	case "none":
		return nil, fmt.Errorf("Key 不存在")
	default:
		return nil, fmt.Errorf("不支持的类型: %s", keyType)
	}

	if err != nil {
		return nil, err
	}

	// ZSet 转换为更友好的格式
	if keyType == "zset" {
		if zMembers, ok := value.([]redis.Z); ok {
			formatted := make([]map[string]interface{}, len(zMembers))
			for i, z := range zMembers {
				formatted[i] = map[string]interface{}{
					"member": z.Member,
					"score":  z.Score,
				}
			}
			value = formatted
		}
	}

	AddOpLog(req.ConnID, "GET", req.Key, "type="+keyType)

	// 大 String 值截断预览（超过 10MB 只返回前 10MB）
	truncated := false
	const maxPreviewSize = 10 * 1024 * 1024 // 10MB
	if keyType == "string" {
		if s, ok := value.(string); ok && len(s) > maxPreviewSize {
			value = s[:maxPreviewSize]
			truncated = true
		}
	}

	return map[string]interface{}{
		"type":      keyType,
		"value":     value,
		"truncated": truncated,
	}, nil
}

func handleSetKeyValue(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Value  string `json:"value"`
		TTL    int64  `json:"ttl"` // -1=不修改TTL, 0=移除TTL, >0=设置秒数
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	AddOpLog(req.ConnID, "SET", req.Key, "set value")

	// 使用 WATCH/MULTI/EXEC 乐观锁保存 String 值
	const maxRetries = 3
	for i := 0; i < maxRetries; i++ {
		err := conn.Watch(ctx, func(tx *redis.Tx) error {
			// 在 WATCH 保护下检查 Key 类型
			keyType, err := tx.Type(ctx, req.Key).Result()
			if err != nil {
				return err
			}
			// Key 已被删除
			if keyType == "none" {
				return fmt.Errorf("KEY_DELETED")
			}
			// Key 类型已变更
			if keyType != "string" {
				return fmt.Errorf("TYPE_CHANGED:%s", keyType)
			}

			// 读取当前 TTL（在 WATCH 保护下）
			var setTTL time.Duration
			switch {
			case req.TTL > 0:
				setTTL = time.Duration(req.TTL) * time.Second
			case req.TTL == -1:
				oldTTL, _ := tx.TTL(ctx, req.Key).Result()
				if oldTTL > 0 {
					setTTL = oldTTL
				}
			default:
				setTTL = 0 // 移除 TTL
			}

			// 在事务中执行 SET
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, req.Key, req.Value, setTTL)
				return nil
			})
			return err
		}, req.Key)

		if err == nil {
			return nil, nil
		}
		// 事务冲突（Key 在 WATCH 和 EXEC 之间被修改），重试
		if err == redis.TxFailedErr {
			continue
		}
		// Key 已被删除
		if err.Error() == "KEY_DELETED" {
			return nil, fmt.Errorf("key_deleted")
		}
		// Key 类型已变更
		if strings.HasPrefix(err.Error(), "TYPE_CHANGED:") {
			newType := strings.TrimPrefix(err.Error(), "TYPE_CHANGED:")
			return nil, fmt.Errorf("type_changed:%s", newType)
		}
		return nil, err
	}

	// 重试耗尽，返回冲突
	return nil, fmt.Errorf("concurrent_conflict")
}

// ========== Key 管理操作 ==========

func handleDeleteKeys(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string   `json:"conn_id"`
		Keys   []string `json:"keys"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	deleted, err := conn.Cmd().Del(ctx, req.Keys...).Result()
	if err != nil {
		return nil, err
	}

	for _, k := range req.Keys {
		AddOpLog(req.ConnID, "DELETE", k, fmt.Sprintf("deleted %d keys", deleted))
	}

	return map[string]interface{}{
		"deleted": deleted,
	}, nil
}

func handleRenameKey(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		NewKey string `json:"new_key"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	if err := conn.Cmd().Rename(ctx, req.Key, req.NewKey).Err(); err != nil {
		return nil, err
	}

	AddOpLog(req.ConnID, "RENAME", req.Key, req.Key+" → "+req.NewKey)

	return nil, nil
}

func handleSetTTL(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		TTL    int64  `json:"ttl"` // -1=移除过期, >0=设置秒数
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	if req.TTL <= 0 {
		if err := conn.Cmd().Persist(ctx, req.Key).Err(); err != nil {
			return nil, err
		}
	} else {
		if err := conn.Cmd().Expire(ctx, req.Key, time.Duration(req.TTL)*time.Second).Err(); err != nil {
			return nil, err
		}
	}

	detail := "persist"
	if req.TTL > 0 {
		detail = fmt.Sprintf("TTL=%ds", req.TTL)
	}
	AddOpLog(req.ConnID, "SET_TTL", req.Key, detail)

	return nil, nil
}

// ========== DB 操作 ==========

func handleSelectDB(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		DB     int    `json:"db"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	if err := conn.SelectDB(req.DB); err != nil {
		return nil, err
	}

	AddOpLog(req.ConnID, "SELECT_DB", "", fmt.Sprintf("db=%d", req.DB))
	return nil, nil
}

func handleGetDBList(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	// 通过 CONFIG GET databases 获取数据库数量
	dbCount := 16 // 默认 16
	cfgResult, err := conn.Cmd().ConfigGet(ctx, "databases").Result()
	if err == nil && len(cfgResult) >= 2 {
		if n, e := strconv.Atoi(cfgResult[1].(string)); e == nil {
			dbCount = n
		}
	}

	// 通过 INFO keyspace 获取各 DB 的 key 数量
	info, _ := conn.Cmd().Info(ctx, "keyspace").Result()
	dbKeys := parseKeyspaceInfo(info)

	type DBInfo struct {
		DB   int   `json:"db"`
		Keys int64 `json:"keys"`
	}

	dbs := make([]DBInfo, dbCount)
	for i := 0; i < dbCount; i++ {
		dbs[i] = DBInfo{DB: i, Keys: dbKeys[i]}
	}

	AddOpLog(req.ConnID, "DB_LIST", "", fmt.Sprintf("%d databases", dbCount))
	return dbs, nil
}

// ========== 跨库搜索 ==========

func handleCrossDbSearch(params json.RawMessage) (any, error) {
	var req struct {
		ConnID    string `json:"conn_id"`
		Pattern   string `json:"pattern"`
		MaxPerDB  int    `json:"max_per_db"`
	}
	if err := json.Unmarshal(params, &req); err != nil || req.ConnID == "" {
		return nil, fmt.Errorf("参数错误")
	}
	if req.Pattern == "" || req.Pattern == "*" {
		return nil, fmt.Errorf("请输入具体的匹配模式，不支持 * 全量搜索")
	}
	if req.MaxPerDB <= 0 {
		req.MaxPerDB = 100
	} else if req.MaxPerDB > 500 {
		req.MaxPerDB = 500
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 30*time.Second)
	defer cancel()

	// 获取 DB 数量
	dbCount := 16
	cfgResult, err := conn.Cmd().ConfigGet(ctx, "databases").Result()
	if err == nil && len(cfgResult) >= 2 {
		if n, e := strconv.Atoi(cfgResult[1].(string)); e == nil {
			dbCount = n
		}
	}

	// 获取各 DB key 数量
	info, _ := conn.Cmd().Info(ctx, "keyspace").Result()
	dbKeys := parseKeyspaceInfo(info)

	// 获取主连接的选项用于创建临时连接
	opts := conn.ClientOptions()

	type SearchKeyItem struct {
		Key  string `json:"key"`
		Type string `json:"type"`
		TTL  int64  `json:"ttl"`
	}
	type DBResult struct {
		DB       int             `json:"db"`
		Keys     []SearchKeyItem `json:"keys"`
		Total    int64           `json:"total"`
		HasMore  bool            `json:"has_more"`
	}

	// 筛选需要扫描的 DB（空库跳过）
	var dbsToScan []int
	for db := 0; db < dbCount; db++ {
		if dbKeys[db] == 0 {
			continue
		}
		dbsToScan = append(dbsToScan, db)
	}

	// 并发扫描各 DB
	type dbScanResult struct {
		dbResult DBResult
		db       int
	}
	ch := make(chan dbScanResult, len(dbsToScan))
	sem := make(chan struct{}, 4) // 最多 4 个并发连接

	for _, db := range dbsToScan {
		sem <- struct{}{}
		go func(db int) {
			defer func() { <-sem }()

			totalKeys := dbKeys[db]
			tmpOpts := *opts
			tmpOpts.DB = db
			tmpClient := redis.NewClient(&tmpOpts)
			defer tmpClient.Close()

			var matched []SearchKeyItem
			var scanCursor uint64
			hasMore := false

			for {
				// 检查超时
				select {
				case <-ctx.Done():
					ch <- dbScanResult{db: db}
					return
				default:
				}

				keys, nextCursor, err := tmpClient.Scan(ctx, scanCursor, req.Pattern, 200).Result()
				if err != nil {
					break
				}

				if len(keys) > 0 {
					pipe := tmpClient.Pipeline()
					typeCmds := make([]*redis.StatusCmd, len(keys))
					ttlCmds := make([]*redis.DurationCmd, len(keys))
					for i, k := range keys {
						typeCmds[i] = pipe.Type(ctx, k)
						ttlCmds[i] = pipe.TTL(ctx, k)
					}
					pipe.Exec(ctx)

					for i, k := range keys {
						if len(matched) >= req.MaxPerDB {
							hasMore = true
							break
						}
						t := typeCmds[i].Val()
						dur := ttlCmds[i].Val()
						var ttl int64
						if dur < 0 {
							ttl = int64(dur) // -1 (永不过期) 或 -2 (key 不存在)
						} else {
							ttl = int64(dur.Seconds())
						}
						matched = append(matched, SearchKeyItem{Key: k, Type: t, TTL: ttl})
					}
				}

				if len(matched) >= req.MaxPerDB || nextCursor == 0 {
					if nextCursor != 0 {
						hasMore = true
					}
					break
				}
				scanCursor = nextCursor
			}

			if len(matched) > 0 {
				ch <- dbScanResult{
					db: db,
					dbResult: DBResult{
						DB:      db,
						Keys:    matched,
						Total:   totalKeys,
						HasMore: hasMore,
					},
				}
			} else {
				ch <- dbScanResult{db: db}
			}
		}(db)
	}

	// 收集结果
	var results []DBResult
	for range dbsToScan {
		r := <-ch
		if len(r.dbResult.Keys) > 0 {
			results = append(results, r.dbResult)
		}
	}

	// 按 DB 号排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].DB < results[j].DB
	})

	if results == nil {
		results = []DBResult{}
	}

	AddOpLog(req.ConnID, "CROSS_DB_SEARCH", "", fmt.Sprintf("pattern=%s dbs=%d", req.Pattern, dbCount))
	return map[string]interface{}{
		"results":     results,
		"scanned_dbs": dbCount,
	}, nil
}

// 格式: db0:keys=123,expires=10,avg_ttl=1000
func parseKeyspaceInfo(info string) map[int]int64 {
	result := make(map[int]int64)
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "db") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		dbNum, err := strconv.Atoi(strings.TrimPrefix(parts[0], "db"))
		if err != nil {
			continue
		}
		for _, kv := range strings.Split(parts[1], ",") {
			if strings.HasPrefix(kv, "keys=") {
				if n, e := strconv.ParseInt(strings.TrimPrefix(kv, "keys="), 10, 64); e == nil {
					result[dbNum] = n
				}
			}
		}
	}
	return result
}

// ========== 服务器状态 ==========

func handleGetServerStatus(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	info, err := conn.Cmd().Info(ctx).Result()
	if err != nil {
		return nil, err
	}

	// 解析 INFO 为结构化数据
	sections := parseInfoSections(info)

	// 提取关键指标
	server := sections["Server"]
	memory := sections["Memory"]
	clients := sections["Clients"]
	stats := sections["Stats"]
	keyspace := sections["Keyspace"]

	summary := map[string]interface{}{
		"redis_version":    server["redis_version"],
		"redis_mode":       server["redis_mode"],
		"role":             sections["Replication"]["role"],
		"uptime_in_seconds": server["uptime_in_seconds"],
		"connected_clients": clients["connected_clients"],
		"used_memory_human": memory["used_memory_human"],
		"used_memory":       memory["used_memory"],
		"maxmemory_human":   memory["maxmemory_human"],
		"total_keys":        countTotalKeys(keyspace),
		"ops_per_sec":       stats["instantaneous_ops_per_sec"],
		"input_kbps":        stats["instantaneous_input_kbps"],
		"output_kbps":       stats["instantaneous_output_kbps"],
	}

	return map[string]interface{}{
		"summary":  summary,
		"sections": sections,
	}, nil
}

// parseInfoSections 将 INFO 输出解析为 section -> key -> value
func parseInfoSections(info string) map[string]map[string]string {
	sections := make(map[string]map[string]string)
	currentSection := ""
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			currentSection = strings.TrimPrefix(line, "# ")
			sections[currentSection] = make(map[string]string)
			continue
		}
		if currentSection != "" && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			sections[currentSection][parts[0]] = parts[1]
		}
	}
	return sections
}

func countTotalKeys(keyspace map[string]string) int64 {
	var total int64
	for _, v := range keyspace {
		for _, kv := range strings.Split(v, ",") {
			if strings.HasPrefix(kv, "keys=") {
				if n, e := strconv.ParseInt(strings.TrimPrefix(kv, "keys="), 10, 64); e == nil {
					total += n
				}
			}
		}
	}
	return total
}

// ========== CLI 命令执行 ==========

func handleExecuteCommand(params json.RawMessage) (any, error) {
	var req struct {
		ConnID  string `json:"conn_id"`
		Command string `json:"command"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 30*time.Second)
	defer cancel()

	// 解析命令字符串为参数列表
	args := parseCommand(req.Command)
	if len(args) == 0 {
		return nil, fmt.Errorf("空命令")
	}

	// 拦截 SELECT 命令：用 SelectDB 正确重建客户端，避免连接池 DB 不一致
	if strings.EqualFold(args[0], "SELECT") {
		if len(args) != 2 {
			return map[string]interface{}{
				"result": "(error) ERR wrong number of arguments for 'select' command",
			}, nil
		}
		db, err := strconv.Atoi(args[1])
		if err != nil {
			return map[string]interface{}{
				"result": "(error) ERR value is not an integer or out of range",
			}, nil
		}
		if err := conn.SelectDB(db); err != nil {
			return map[string]interface{}{
				"result": "(error) " + err.Error(),
			}, nil
		}
		AddOpLog(req.ConnID, "CLI", "", req.Command)
		return map[string]interface{}{
			"result": "OK",
		}, nil
	}

	// 转换为 interface{} 切片
	iArgs := make([]interface{}, len(args))
	for i, a := range args {
		iArgs[i] = a
	}

	result, err := conn.Do(ctx, iArgs...).Result()
	if err != nil {
		if err == redis.Nil {
			return map[string]interface{}{
				"result": "(nil)",
			}, nil
		}
		return map[string]interface{}{
			"result": "(error) " + err.Error(),
		}, nil
	}

	AddOpLog(req.ConnID, "CLI", "", req.Command)

	return map[string]interface{}{
		"result": formatResult(result),
	}, nil
}

// parseCommand 解析 Redis 命令字符串，支持引号
func parseCommand(cmd string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(cmd); i++ {
		ch := cmd[i]
		if inQuote {
			if ch == quoteChar {
				inQuote = false
			} else if ch == '\\' && i+1 < len(cmd) {
				i++
				switch cmd[i] {
				case 'n':
					current.WriteByte('\n')
				case 't':
					current.WriteByte('\t')
				case '\\':
					current.WriteByte('\\')
				case '"':
					current.WriteByte('"')
				case '\'':
					current.WriteByte('\'')
				default:
					current.WriteByte('\\')
					current.WriteByte(cmd[i])
				}
			} else {
				current.WriteByte(ch)
			}
		} else {
			if ch == '"' || ch == '\'' {
				inQuote = true
				quoteChar = ch
			} else if ch == ' ' || ch == '\t' {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
			} else {
				current.WriteByte(ch)
			}
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// formatResult 格式化 Redis 命令结果为可读字符串
func formatResult(result interface{}) string {
	switch v := result.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case int64:
		return fmt.Sprintf("(integer) %d", v)
	case []interface{}:
		if len(v) == 0 {
			return "(empty array)"
		}
		var sb strings.Builder
		for i, item := range v {
			if i > 0 {
				sb.WriteByte('\n')
			}
			sb.WriteString(fmt.Sprintf("%d) %s", i+1, formatResult(item)))
		}
		return sb.String()
	case nil:
		return "(nil)"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ========== 连接导入/导出 ==========

func handleImportConnections(params json.RawMessage) (any, error) {
	var req struct {
		Connections []map[string]interface{} `json:"connections"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	var existing []map[string]interface{}
	ReadJSON("connections.json", &existing)

	// 按 id 去重合并
	idSet := make(map[string]bool)
	for _, c := range existing {
		if id, ok := c["id"].(string); ok {
			idSet[id] = true
		}
	}

	// 需要加密的密码字段 → 对应的 encrypted 标志
	passwordFields := map[string]string{
		"password":          "password_encrypted",
		"ssh_password":      "ssh_password_encrypted",
		"ssh_passphrase":    "ssh_passphrase_encrypted",
		"sentinel_password": "sentinel_password_encrypted",
		"proxy_password":    "proxy_password_encrypted",
	}

	// Base64 data 字段 → 对应的文件路径字段 → 文件名
	fileDataFields := map[string]struct {
		pathField string
		fileName  string
	}{
		"tls_ca_data":           {pathField: "tls_ca_file", fileName: "ca.crt"},
		"tls_cert_data":         {pathField: "tls_cert_file", fileName: "client.crt"},
		"tls_key_data":          {pathField: "tls_key_file", fileName: "client.key"},
		"ssh_private_key_data":  {pathField: "ssh_private_key", fileName: "ssh_key"},
	}

	imported := 0
	for _, c := range req.Connections {
		id, _ := c["id"].(string)
		if id == "" || idSet[id] {
			continue
		}

		// 加密所有密码字段
		for pwdField, encFlag := range passwordFields {
			if pwd, ok := c[pwdField].(string); ok && pwd != "" {
				encrypted, err := utils.Encrypt(pwd)
				if err == nil {
					c[pwdField] = encrypted
					c[encFlag] = true
				}
			}
		}

		// 还原 Base64 嵌入的证书/密钥文件
		for dataField, info := range fileDataFields {
			if b64, ok := c[dataField].(string); ok && b64 != "" {
				content, err := base64.StdEncoding.DecodeString(b64)
				if err == nil {
					certDir := filepath.Join(GetDataDir(), "certs", id)
					os.MkdirAll(certDir, 0755)
					filePath := filepath.Join(certDir, info.fileName)
					if err := os.WriteFile(filePath, content, 0600); err == nil {
						c[info.pathField] = filePath
					}
				}
				delete(c, dataField) // 不存储 Base64 数据到 connections.json
			}
		}

		existing = append(existing, c)
		idSet[id] = true
		imported++
	}

	if err := WriteJSON("connections.json", existing); err != nil {
		return nil, err
	}

	AddOpLog("", "IMPORT_CONNS", "", fmt.Sprintf("%d connections", imported))
	return map[string]interface{}{
		"imported": imported,
	}, nil
}

// ========== Key 创建 ==========

type HashFieldPair struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

type ZSetMemberPair struct {
	Member string  `json:"member"`
	Score  float64 `json:"score"`
}

type StreamFieldPair struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

type GeoMemberPair struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
	Member    string  `json:"member"`
}

func handleCreateKey(params json.RawMessage) (any, error) {
	var req struct {
		ConnID       string            `json:"conn_id"`
		Key          string            `json:"key"`
		Type         string            `json:"type"`
		Value        string            `json:"value"`
		TTL          int64             `json:"ttl"`
		HashFields   []HashFieldPair   `json:"hash_fields"`
		ListValues   []string          `json:"list_values"`
		SetMembers   []string          `json:"set_members"`
		ZSetMembers  []ZSetMemberPair  `json:"zset_members"`
		StreamID     string            `json:"stream_id"`
		StreamFields []StreamFieldPair `json:"stream_fields"`
		BitmapOffset int64             `json:"bitmap_offset"`
		HllElements  []string          `json:"hll_elements"`
		GeoMembers   []GeoMemberPair   `json:"geo_members"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	if req.Key == "" {
		return nil, fmt.Errorf("Key 不能为空")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	// 检查 key 是否已存在
	exists, err := conn.Cmd().Exists(ctx, req.Key).Result()
	if err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, fmt.Errorf("Key 已存在")
	}

	// 根据类型创建
	switch req.Type {
	case "string":
		ttl := time.Duration(0)
		if req.TTL > 0 {
			ttl = time.Duration(req.TTL) * time.Second
		}
		if err := conn.Cmd().Set(ctx, req.Key, req.Value, ttl).Err(); err != nil {
			return nil, err
		}
	case "hash":
		if len(req.HashFields) > 0 {
			args := make([]interface{}, 0, len(req.HashFields)*2)
			for _, f := range req.HashFields {
				if f.Field != "" {
					args = append(args, f.Field, f.Value)
				}
			}
			if len(args) > 0 {
				if err := conn.Cmd().HSet(ctx, req.Key, args...).Err(); err != nil {
					return nil, err
				}
			} else {
				if err := conn.Cmd().HSet(ctx, req.Key, "field1", "").Err(); err != nil {
					return nil, err
				}
			}
		} else {
			if err := conn.Cmd().HSet(ctx, req.Key, "field1", "").Err(); err != nil {
				return nil, err
			}
		}
	case "list":
		if len(req.ListValues) > 0 {
			vals := make([]interface{}, 0, len(req.ListValues))
			for _, v := range req.ListValues {
				vals = append(vals, v)
			}
			if err := conn.Cmd().RPush(ctx, req.Key, vals...).Err(); err != nil {
				return nil, err
			}
		} else {
			if err := conn.Cmd().RPush(ctx, req.Key, "").Err(); err != nil {
				return nil, err
			}
		}
	case "set":
		if len(req.SetMembers) > 0 {
			vals := make([]interface{}, 0, len(req.SetMembers))
			for _, m := range req.SetMembers {
				vals = append(vals, m)
			}
			if err := conn.Cmd().SAdd(ctx, req.Key, vals...).Err(); err != nil {
				return nil, err
			}
		} else {
			if err := conn.Cmd().SAdd(ctx, req.Key, "member1").Err(); err != nil {
				return nil, err
			}
		}
	case "zset":
		if len(req.ZSetMembers) > 0 {
			members := make([]*redis.Z, 0, len(req.ZSetMembers))
			for _, m := range req.ZSetMembers {
				if m.Member != "" {
					members = append(members, &redis.Z{Score: m.Score, Member: m.Member})
				}
			}
			if len(members) > 0 {
				if err := conn.Cmd().ZAdd(ctx, req.Key, members...).Err(); err != nil {
					return nil, err
				}
			} else {
				if err := conn.Cmd().ZAdd(ctx, req.Key, &redis.Z{Score: 0, Member: "member1"}).Err(); err != nil {
					return nil, err
				}
			}
		} else {
			if err := conn.Cmd().ZAdd(ctx, req.Key, &redis.Z{Score: 0, Member: "member1"}).Err(); err != nil {
				return nil, err
			}
		}
	case "stream":
		streamID := "*"
		if req.StreamID != "" {
			streamID = req.StreamID
		}
		values := map[string]interface{}{"field1": "value1"}
		if len(req.StreamFields) > 0 {
			custom := make(map[string]interface{}, len(req.StreamFields))
			for _, f := range req.StreamFields {
				if f.Field != "" {
					custom[f.Field] = f.Value
				}
			}
			if len(custom) > 0 {
				values = custom
			}
		}
		if err := conn.Cmd().XAdd(ctx, &redis.XAddArgs{
			Stream: req.Key,
			ID:     streamID,
			Values: values,
		}).Err(); err != nil {
			return nil, err
		}
	case "bitmap":
		if err := conn.Cmd().SetBit(ctx, req.Key, req.BitmapOffset, 1).Err(); err != nil {
			return nil, err
		}
	case "hll":
		if len(req.HllElements) > 0 {
			vals := make([]interface{}, 0, len(req.HllElements))
			for _, e := range req.HllElements {
				if e != "" {
					vals = append(vals, e)
				}
			}
			if len(vals) > 0 {
				if err := conn.Cmd().PFAdd(ctx, req.Key, vals...).Err(); err != nil {
					return nil, err
				}
			} else {
				if err := conn.Cmd().PFAdd(ctx, req.Key, "element1").Err(); err != nil {
					return nil, err
				}
			}
		} else {
			if err := conn.Cmd().PFAdd(ctx, req.Key, "element1").Err(); err != nil {
				return nil, err
			}
		}
	case "geo":
		if len(req.GeoMembers) > 0 {
			locs := make([]*redis.GeoLocation, 0, len(req.GeoMembers))
			for _, m := range req.GeoMembers {
				if m.Member != "" {
					locs = append(locs, &redis.GeoLocation{
						Name:      m.Member,
						Longitude: m.Longitude,
						Latitude:  m.Latitude,
					})
				}
			}
			if len(locs) > 0 {
				if err := conn.Cmd().GeoAdd(ctx, req.Key, locs...).Err(); err != nil {
					return nil, err
				}
			} else {
				if err := conn.Cmd().GeoAdd(ctx, req.Key, &redis.GeoLocation{Name: "point1", Longitude: 0, Latitude: 0}).Err(); err != nil {
					return nil, err
				}
			}
		} else {
			if err := conn.Cmd().GeoAdd(ctx, req.Key, &redis.GeoLocation{Name: "point1", Longitude: 0, Latitude: 0}).Err(); err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("不支持的类型: %s", req.Type)
	}

	// 非 string 类型设置 TTL
	if req.Type != "string" && req.TTL > 0 {
		conn.Cmd().Expire(ctx, req.Key, time.Duration(req.TTL)*time.Second)
	}

	AddOpLog(req.ConnID, "CREATE", req.Key, "type="+req.Type)

	return nil, nil
}

func handleCheckKeyExists(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	exists, err := conn.Cmd().Exists(ctx, req.Key).Result()
	if err != nil {
		return nil, err
	}

	AddOpLog(req.ConnID, "EXISTS", req.Key, fmt.Sprintf("exists=%v", exists > 0))
	return map[string]interface{}{
		"exists": exists > 0,
	}, nil
}

func handleExportConnections(params json.RawMessage) (any, error) {
	var req struct {
		IncludePasswords bool `json:"include_passwords"`
	}
	// params 可选，默认不含密码
	if params != nil {
		json.Unmarshal(params, &req)
	}

	var conns []map[string]interface{}
	if err := ReadJSON("connections.json", &conns); err != nil {
		conns = []map[string]interface{}{}
	}

	// 需要处理的密码字段
	passwordFields := []string{
		"password", "ssh_password", "ssh_passphrase",
		"sentinel_password", "proxy_password",
	}
	encryptedFlags := []string{
		"password_encrypted", "ssh_password_encrypted",
		"ssh_passphrase_encrypted", "sentinel_password_encrypted",
		"proxy_password_encrypted",
	}

	// 需要嵌入的文件字段 → 对应的 data 字段名
	fileFields := map[string]string{
		"tls_ca_file":      "tls_ca_data",
		"tls_cert_file":    "tls_cert_data",
		"tls_key_file":     "tls_key_data",
		"ssh_private_key":  "ssh_private_key_data",
	}

	exported := make([]map[string]interface{}, len(conns))
	for i, c := range conns {
		cp := make(map[string]interface{})
		for k, v := range c {
			cp[k] = v
		}

		if req.IncludePasswords {
			// 解密密码后导出明文
			for j, field := range passwordFields {
				if pwd, ok := cp[field].(string); ok && pwd != "" {
					flagField := encryptedFlags[j]
					if isEnc, _ := cp[flagField].(bool); isEnc {
						if plain, err := utils.Decrypt(pwd); err == nil {
							cp[field] = plain
						}
					}
					delete(cp, flagField)
				}
			}
		} else {
			// 不含密码：移除所有密码字段
			for _, field := range passwordFields {
				delete(cp, field)
			}
			for _, flag := range encryptedFlags {
				delete(cp, flag)
			}
		}

		// 嵌入证书/密钥文件内容为 Base64
		for pathField, dataField := range fileFields {
			if filePath, ok := cp[pathField].(string); ok && filePath != "" {
				if content, err := os.ReadFile(filePath); err == nil {
					cp[dataField] = base64.StdEncoding.EncodeToString(content)
				}
			}
		}

		exported[i] = cp
	}

	AddOpLog("", "EXPORT_CONNS", "", fmt.Sprintf("%d connections", len(exported)))
	return exported, nil
}

// ========== 连接导出为 YAML+ZIP ==========

func handleExportConnectionsZip(params json.RawMessage) (any, error) {
	var req struct {
		IncludePasswords bool   `json:"include_passwords"`
		ExportPath       string `json:"export_path"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	if req.ExportPath == "" {
		return nil, fmt.Errorf("导出路径不能为空")
	}

	var conns []map[string]interface{}
	if err := ReadJSON("connections.json", &conns); err != nil {
		conns = []map[string]interface{}{}
	}

	// 读取显示顺序和分组元数据
	var displayOrder []string
	if err := ReadJSON("groups.json", &displayOrder); err != nil {
		displayOrder = []string{}
	}
	var groupMeta map[string]string
	if err := ReadJSON("group_meta.json", &groupMeta); err != nil {
		groupMeta = map[string]string{}
	}

	// 按显示顺序排列连接
	connMap := make(map[string]map[string]interface{})
	for _, c := range conns {
		if id, ok := c["id"].(string); ok {
			connMap[id] = c
		}
	}

	ordered := make([]map[string]interface{}, 0, len(conns))
	usedIDs := make(map[string]bool)
	groupPrefix := "__group__"

	for _, key := range displayOrder {
		if strings.HasPrefix(key, groupPrefix) {
			// 分组 key：找出该分组下的所有连接（group 字段存的是分组 ID）
			groupID := key[len(groupPrefix):]
			for _, c := range conns {
				if g, _ := c["group"].(string); g == groupID {
					if id, _ := c["id"].(string); !usedIDs[id] {
						ordered = append(ordered, c)
						usedIDs[id] = true
					}
				}
			}
		} else {
			// 未分组连接 ID
			if c, ok := connMap[key]; ok && !usedIDs[key] {
				ordered = append(ordered, c)
				usedIDs[key] = true
			}
		}
	}
	// 追加不在 displayOrder 中的连接
	for _, c := range conns {
		if id, _ := c["id"].(string); !usedIDs[id] {
			ordered = append(ordered, c)
		}
	}

	passwordFields := []string{
		"password", "ssh_password", "ssh_passphrase",
		"sentinel_password", "proxy_password",
	}
	encryptedFlags := []string{
		"password_encrypted", "ssh_password_encrypted",
		"ssh_passphrase_encrypted", "sentinel_password_encrypted",
		"proxy_password_encrypted",
	}

	// 需要省略的默认值字段（值为零值/默认值时不输出）
	defaultValues := map[string]interface{}{
		"db":                  float64(0),
		"conn_type":           "tcp",
		"conn_timeout":        float64(10),
		"exec_timeout":        float64(10),
		"username":            "",
		"key_filter":          "",
		"key_separator":       ":",
		"default_view":        "tree",
		"scan_count":          float64(200),
		"db_filter_mode":      "all",
		"unix_socket":         "",
		"use_tls":             false,
		"tls_cert_file":       "",
		"tls_key_file":        "",
		"tls_ca_file":         "",
		"tls_skip_verify":     false,
		"use_ssh":             false,
		"ssh_host":            "",
		"ssh_port":            float64(22),
		"ssh_username":        "",
		"ssh_private_key":     "",
		"use_proxy":           false,
		"proxy_type":          "socks5",
		"proxy_host":          "",
		"proxy_port":          float64(1080),
		"proxy_username":      "",
		"use_sentinel":        false,
		"sentinel_addrs":      "",
		"sentinel_master_name": "",
		"use_cluster":         false,
		"cluster_addrs":       "",
	}

	exported := make([]map[string]interface{}, 0, len(ordered))
	for _, c := range ordered {
		cp := make(map[string]interface{})
		for k, v := range c {
			cp[k] = v
		}

		// 处理密码
		if req.IncludePasswords {
			for j, field := range passwordFields {
				if pwd, ok := cp[field].(string); ok && pwd != "" {
					flagField := encryptedFlags[j]
					if isEnc, _ := cp[flagField].(bool); isEnc {
						if plain, err := utils.Decrypt(pwd); err == nil {
							cp[field] = plain
						}
					}
					delete(cp, flagField)
				}
			}
		} else {
			for _, field := range passwordFields {
				delete(cp, field)
			}
			for _, flag := range encryptedFlags {
				delete(cp, flag)
			}
		}

		// 省略默认值字段
		for field, defVal := range defaultValues {
			if v, exists := cp[field]; exists && fmt.Sprintf("%v", v) == fmt.Sprintf("%v", defVal) {
				delete(cp, field)
			}
		}

		// 省略空数组 db_filter_list
		if arr, ok := cp["db_filter_list"].([]interface{}); ok && len(arr) == 0 {
			delete(cp, "db_filter_list")
		}

		// 省略空字符串 group
		if g, _ := cp["group"].(string); g == "" {
			delete(cp, "group")
		}

		exported = append(exported, cp)
	}

	// 过滤 display_order：去掉孤立的无效引用
	validOrder := make([]string, 0, len(displayOrder))
	for _, key := range displayOrder {
		if strings.HasPrefix(key, groupPrefix) {
			validOrder = append(validOrder, key)
		} else if _, ok := connMap[key]; ok {
			validOrder = append(validOrder, key)
		}
	}

	// 导出用的 group_meta：只包含 display_order 中出现的分组
	exportMeta := make(map[string]string)
	for _, key := range validOrder {
		if strings.HasPrefix(key, groupPrefix) {
			groupID := key[len(groupPrefix):]
			if name, ok := groupMeta[groupID]; ok {
				exportMeta[groupID] = name
			}
		}
	}

	// 构建 YAML — 包含 group_meta, display_order, connections
	yamlData := map[string]interface{}{
		"group_meta":    exportMeta,
		"display_order": validOrder,
		"connections":   exported,
	}
	yamlBytes, err := yaml.Marshal(yamlData)
	if err != nil {
		return nil, fmt.Errorf("YAML 序列化失败: %v", err)
	}

	// 构建 ZIP
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	w, err := zipWriter.Create("connections.yaml")
	if err != nil {
		return nil, fmt.Errorf("创建 ZIP 条目失败: %v", err)
	}
	if _, err := w.Write(yamlBytes); err != nil {
		return nil, fmt.Errorf("写入 ZIP 失败: %v", err)
	}
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("关闭 ZIP 失败: %v", err)
	}

	// 写入文件
	zipName := fmt.Sprintf("easy_rdm_connections_%s.zip", time.Now().Format("20060102_150405"))
	zipPath := filepath.Join(req.ExportPath, zipName)
	if err := os.WriteFile(zipPath, buf.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("写入文件失败: %v", err)
	}

	AddOpLog("", "EXPORT_CONNS_ZIP", "", fmt.Sprintf("%d connections → %s", len(exported), zipPath))
	return map[string]interface{}{
		"path":  zipPath,
		"count": len(exported),
	}, nil
}

// ========== 从 ZIP 导入连接 ==========

func handleImportConnectionsZip(params json.RawMessage) (any, error) {
	var req struct {
		FilePath string `json:"file_path"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	if req.FilePath == "" {
		return nil, fmt.Errorf("文件路径不能为空")
	}

	// 检查 ZIP 文件大小
	info, err := os.Stat(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}
	if info.Size() > 50*1024*1024 {
		return nil, fmt.Errorf("ZIP 文件过大（%.1fMB），最大支持 50MB", float64(info.Size())/1024/1024)
	}

	// 读取 ZIP 文件
	zipData, err := os.ReadFile(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("解析 ZIP 失败: %v", err)
	}

	// 查找 connections.yaml
	var yamlContent []byte
	for _, f := range zipReader.File {
		if f.Name == "connections.yaml" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("打开 ZIP 条目失败: %v", err)
			}
			yamlContent, err = io.ReadAll(io.LimitReader(rc, 100*1024*1024))
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("读取 YAML 失败: %v", err)
			}
			break
		}
	}
	if yamlContent == nil {
		return nil, fmt.Errorf("ZIP 中未找到 connections.yaml")
	}

	// 解析 YAML（新格式含 group_meta + display_order）
	var yamlData struct {
		GroupMeta    map[string]string        `yaml:"group_meta"`
		DisplayOrder []string                 `yaml:"display_order"`
		Connections  []map[string]interface{} `yaml:"connections"`
	}
	if err := yaml.Unmarshal(yamlContent, &yamlData); err != nil {
		return nil, fmt.Errorf("YAML 解析失败: %v", err)
	}
	if yamlData.GroupMeta == nil {
		yamlData.GroupMeta = map[string]string{}
	}

	// 连接默认值（导出时省略的字段在导入时补齐）
	connDefaults := map[string]interface{}{
		"db":                  float64(0),
		"conn_type":           "tcp",
		"conn_timeout":        float64(10),
		"exec_timeout":        float64(10),
		"username":            "",
		"key_filter":          "",
		"key_separator":       ":",
		"default_view":        "tree",
		"scan_count":          float64(200),
		"db_filter_mode":      "all",
		"db_filter_list":      []interface{}{},
		"use_tls":             false,
		"tls_cert_file":       "",
		"tls_key_file":        "",
		"tls_ca_file":         "",
		"tls_skip_verify":     false,
		"use_ssh":             false,
		"ssh_host":            "",
		"ssh_port":            float64(22),
		"ssh_username":        "",
		"ssh_private_key":     "",
		"use_proxy":           false,
		"proxy_type":          "socks5",
		"proxy_host":          "",
		"proxy_port":          float64(1080),
		"proxy_username":      "",
		"use_sentinel":        false,
		"sentinel_addrs":      "",
		"sentinel_master_name": "",
		"use_cluster":         false,
		"cluster_addrs":       "",
		"group":               "",
	}

	// 读取现有数据
	var existing []map[string]interface{}
	ReadJSON("connections.json", &existing)

	passwordFields := map[string]string{
		"password":          "password_encrypted",
		"ssh_password":      "ssh_password_encrypted",
		"ssh_passphrase":    "ssh_passphrase_encrypted",
		"sentinel_password": "sentinel_password_encrypted",
		"proxy_password":    "proxy_password_encrypted",
	}

	groupPrefix := "__group__"

	// 为导入的分组生成新 ID（旧分组ID → 新分组ID）
	groupIDMapping := make(map[string]string)
	for oldGroupID := range yamlData.GroupMeta {
		newGroupID := fmt.Sprintf("grp_%d%s", time.Now().UnixNano(), randStr(4))
		groupIDMapping[oldGroupID] = newGroupID
	}

	// 为导入的连接生成新 ID（旧连接ID → 新连接ID）
	connIDMapping := make(map[string]string)

	imported := 0
	for _, c := range yamlData.Connections {
		// 补齐默认值
		for field, defVal := range connDefaults {
			if _, exists := c[field]; !exists {
				c[field] = defVal
			}
		}

		// 生成新连接 ID
		newConnID := fmt.Sprintf("%d%s", time.Now().UnixNano(), randStr(6))
		oldConnID, _ := c["id"].(string)
		c["id"] = newConnID
		if oldConnID != "" {
			connIDMapping[oldConnID] = newConnID
		}

		// 映射 group 字段到新分组 ID
		if g, _ := c["group"].(string); g != "" {
			if newGroupID, ok := groupIDMapping[g]; ok {
				c["group"] = newGroupID
			}
			// 如果映射不到（可能是旧格式导出的），保留原值
		}

		// 加密密码字段
		for pwdField, encFlag := range passwordFields {
			if pwd, ok := c[pwdField].(string); ok && pwd != "" {
				encrypted, err := utils.Encrypt(pwd)
				if err == nil {
					c[pwdField] = encrypted
					c[encFlag] = true
				}
			}
		}

		existing = append(existing, c)
		imported++
	}

	if err := WriteJSON("connections.json", existing); err != nil {
		return nil, err
	}

	// 合并 display_order（导入的条目追加到现有顺序末尾，每个都用新 ID）
	var existingOrder []string
	ReadJSON("groups.json", &existingOrder)

	for _, key := range yamlData.DisplayOrder {
		if strings.HasPrefix(key, groupPrefix) {
			// 分组 key：映射到新分组 ID
			oldGroupID := key[len(groupPrefix):]
			if newGroupID, ok := groupIDMapping[oldGroupID]; ok {
				existingOrder = append(existingOrder, groupPrefix+newGroupID)
			}
		} else {
			// 连接 ID：映射到新连接 ID
			if newConnID, ok := connIDMapping[key]; ok {
				existingOrder = append(existingOrder, newConnID)
			}
		}
	}

	// 兜底：确保所有导入的连接都在 display_order 中
	orderSet := make(map[string]bool)
	for _, k := range existingOrder {
		orderSet[k] = true
	}
	for _, c := range yamlData.Connections {
		newID, _ := c["id"].(string)
		g, _ := c["group"].(string)
		if g == "" && !orderSet[newID] {
			existingOrder = append(existingOrder, newID)
		}
		if g != "" {
			groupKey := groupPrefix + g
			if !orderSet[groupKey] {
				existingOrder = append(existingOrder, groupKey)
				orderSet[groupKey] = true
			}
		}
	}

	WriteJSON("groups.json", existingOrder)

	// 合并 group_meta
	var existingMeta map[string]string
	if err := ReadJSON("group_meta.json", &existingMeta); err != nil {
		existingMeta = map[string]string{}
	}
	for oldGroupID, name := range yamlData.GroupMeta {
		if newGroupID, ok := groupIDMapping[oldGroupID]; ok {
			existingMeta[newGroupID] = name
		}
	}
	WriteJSON("group_meta.json", existingMeta)

	AddOpLog("", "IMPORT_CONNS_ZIP", "", fmt.Sprintf("imported %d connections from %s", imported, req.FilePath))
	return map[string]interface{}{
		"imported": imported,
	}, nil
}

func randStr(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	seed := time.Now().UnixNano()
	for i := range b {
		idx := seed % int64(len(letters))
		if idx < 0 {
			idx = -idx
		}
		b[i] = letters[idx]
		seed = seed*1103515245 + 12345
	}
	return string(b)
}

// ========== FLUSHDB ==========

func handleFlushDB(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 30*time.Second)
	defer cancel()

	if err := conn.Cmd().FlushDB(ctx).Err(); err != nil {
		return nil, err
	}

	AddOpLog(req.ConnID, "FLUSHDB", "", "flush current database")
	return nil, nil
}

// ========== 复制为 Redis 命令 ==========

func handleCopyAsCommand(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	keyType, err := conn.Cmd().Type(ctx, req.Key).Result()
	if err != nil {
		return nil, err
	}

	quoted := quoteRedisArg(req.Key)
	var command string

	switch keyType {
	case "string":
		val, err := conn.Cmd().Get(ctx, req.Key).Result()
		if err != nil {
			return nil, err
		}
		command = fmt.Sprintf("SET %s %s", quoted, quoteRedisArg(val))

	case "hash":
		fields, err := conn.Cmd().HGetAll(ctx, req.Key).Result()
		if err != nil {
			return nil, err
		}
		parts := []string{"HSET", quoted}
		for k, v := range fields {
			parts = append(parts, quoteRedisArg(k), quoteRedisArg(v))
		}
		command = strings.Join(parts, " ")

	case "list":
		vals, err := conn.Cmd().LRange(ctx, req.Key, 0, -1).Result()
		if err != nil {
			return nil, err
		}
		parts := []string{"RPUSH", quoted}
		for _, v := range vals {
			parts = append(parts, quoteRedisArg(v))
		}
		command = strings.Join(parts, " ")

	case "set":
		members, err := conn.Cmd().SMembers(ctx, req.Key).Result()
		if err != nil {
			return nil, err
		}
		parts := []string{"SADD", quoted}
		for _, m := range members {
			parts = append(parts, quoteRedisArg(m))
		}
		command = strings.Join(parts, " ")

	case "zset":
		members, err := conn.Cmd().ZRangeWithScores(ctx, req.Key, 0, -1).Result()
		if err != nil {
			return nil, err
		}
		parts := []string{"ZADD", quoted}
		for _, z := range members {
			parts = append(parts, fmt.Sprintf("%g", z.Score), quoteRedisArg(z.Member.(string)))
		}
		command = strings.Join(parts, " ")

	case "stream":
		// Stream 只生成最近 100 条消息的 XADD
		msgs, err := conn.Cmd().XRevRangeN(ctx, req.Key, "+", "-", 100).Result()
		if err != nil {
			return nil, err
		}
		var lines []string
		// 反转为正序
		for i := len(msgs) - 1; i >= 0; i-- {
			m := msgs[i]
			parts := []string{"XADD", quoted, m.ID}
			for k, v := range m.Values {
				parts = append(parts, quoteRedisArg(k), quoteRedisArg(fmt.Sprintf("%v", v)))
			}
			lines = append(lines, strings.Join(parts, " "))
		}
		command = strings.Join(lines, "\n")

	default:
		command = fmt.Sprintf("# Unsupported type: %s", keyType)
	}

	// 追加 TTL
	ttl, _ := conn.Cmd().TTL(ctx, req.Key).Result()
	if ttl > 0 {
		command += fmt.Sprintf("\nEXPIRE %s %d", quoted, int64(ttl.Seconds()))
	}

	AddOpLog(req.ConnID, "COPY_AS_CMD", req.Key, keyType)
	return command, nil
}

// quoteRedisArg 对 Redis 参数进行引号包裹（含空格或特殊字符时）
func quoteRedisArg(s string) string {
	if s == "" {
		return `""`
	}
	needsQuote := false
	for _, c := range s {
		if c <= ' ' || c == '"' || c == '\'' || c == '\\' {
			needsQuote = true
			break
		}
	}
	if !needsQuote {
		return s
	}
	// 使用双引号包裹，转义内部双引号和反斜杠
	var b strings.Builder
	b.WriteByte('"')
	for _, c := range s {
		switch c {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}
