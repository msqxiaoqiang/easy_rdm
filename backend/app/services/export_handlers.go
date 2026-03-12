package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// RegisterExportHandlers 注册导入/导出相关的 RPC 方法
func RegisterExportHandlers(register func(string, RPCHandlerFunc)) {
	register("export_keys", handleExportKeys)
	register("import_keys", handleImportKeys)
	register("export_keys_file", handleExportKeysFile)
	register("import_keys_file", handleImportKeysFile)
}

// ========== 导出 ==========

// ExportKeyItem 导出的 Key 数据结构
type ExportKeyItem struct {
	Key   string      `json:"key"`
	Type  string      `json:"type"`
	TTL   int64       `json:"ttl"`
	Value interface{} `json:"value"`
}

func handleExportKeys(params json.RawMessage) (any, error) {
	var req struct {
		ConnID  string   `json:"conn_id"`
		Format  string   `json:"format"`  // json / csv / redis_cmd
		Scope   string   `json:"scope"`   // all / selected / pattern
		Keys    []string `json:"keys"`    // scope=selected 时使用
		Pattern string   `json:"pattern"` // scope=pattern 时使用
		Limit   int      `json:"limit"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	if req.Format == "" {
		req.Format = "json"
	}
	if req.Limit <= 0 || req.Limit > 50000 {
		req.Limit = 50000
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 120*time.Second)
	defer cancel()

	// 1. 收集要导出的 key 列表
	var keys []string
	switch req.Scope {
	case "selected":
		keys = req.Keys
	case "pattern":
		if req.Pattern == "" {
			req.Pattern = "*"
		}
		var cursor uint64
		for {
			batch, next, err := conn.Cmd().Scan(ctx, cursor, req.Pattern, 500).Result()
			if err != nil {
				return nil, err
			}
			keys = append(keys, batch...)
			if len(keys) >= req.Limit {
				keys = keys[:req.Limit]
				break
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
	default: // all
		var cursor uint64
		for {
			batch, next, err := conn.Cmd().Scan(ctx, cursor, "*", 500).Result()
			if err != nil {
				return nil, err
			}
			keys = append(keys, batch...)
			if len(keys) >= req.Limit {
				keys = keys[:req.Limit]
				break
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
	}

	if len(keys) == 0 {
		return map[string]interface{}{
			"content": "",
			"count":   0,
		}, nil
	}

	// 2. 读取每个 key 的类型、TTL、值
	items := make([]ExportKeyItem, 0, len(keys))
	for _, key := range keys {
		item, err := readKeyForExport(ctx, conn.Client, key)
		if err != nil {
			continue // 跳过读取失败的 key
		}
		if item != nil {
			items = append(items, *item)
		}
	}

	// 3. 按格式序列化
	var content string
	switch req.Format {
	case "csv":
		content = formatCSV(items)
	case "redis_cmd":
		content = formatRedisCommands(items)
	default: // json
		data, _ := json.MarshalIndent(items, "", "  ")
		content = string(data)
	}

	AddOpLog(req.ConnID, "EXPORT", "", fmt.Sprintf("format=%s count=%d", req.Format, len(items)))
	return map[string]interface{}{
		"content": content,
		"count":   len(items),
	}, nil
}

func readKeyForExport(ctx context.Context, client redis.Cmdable, key string) (*ExportKeyItem, error) {
	pipe := client.Pipeline()
	typeCmd := pipe.Type(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)
	pipe.Exec(ctx)

	keyType := typeCmd.Val()
	if keyType == "none" {
		return nil, nil
	}

	ttlVal := int64(-1)
	if d := ttlCmd.Val(); d > 0 {
		ttlVal = int64(d.Seconds())
	}

	var value interface{}
	var err error

	switch keyType {
	case "string":
		value, err = client.Get(ctx, key).Result()
	case "hash":
		value, err = client.HGetAll(ctx, key).Result()
	case "list":
		value, err = client.LRange(ctx, key, 0, -1).Result()
	case "set":
		value, err = client.SMembers(ctx, key).Result()
	case "zset":
		var members []redis.Z
		members, err = client.ZRangeWithScores(ctx, key, 0, -1).Result()
		if err == nil {
			formatted := make([]map[string]interface{}, len(members))
			for i, z := range members {
				formatted[i] = map[string]interface{}{
					"member": z.Member,
					"score":  z.Score,
				}
			}
			value = formatted
		}
	case "stream":
		var msgs []redis.XMessage
		msgs, err = client.XRange(ctx, key, "-", "+").Result()
		if err == nil {
			formatted := make([]map[string]interface{}, len(msgs))
			for i, m := range msgs {
				formatted[i] = map[string]interface{}{
					"id":     m.ID,
					"values": m.Values,
				}
			}
			value = formatted
		}
	default:
		// 不支持的类型（bitmap/hll/geo 等用 DUMP 导出不实际，跳过）
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &ExportKeyItem{
		Key:   key,
		Type:  keyType,
		TTL:   ttlVal,
		Value: value,
	}, nil
}

func formatCSV(items []ExportKeyItem) string {
	var sb strings.Builder
	w := csv.NewWriter(&sb)
	w.Write([]string{"key", "type", "ttl", "value"})
	for _, item := range items {
		var valStr string
		switch v := item.Value.(type) {
		case string:
			valStr = v
		default:
			data, _ := json.Marshal(v)
			valStr = string(data)
		}
		w.Write([]string{item.Key, item.Type, strconv.FormatInt(item.TTL, 10), valStr})
	}
	w.Flush()
	return sb.String()
}

func formatRedisCommands(items []ExportKeyItem) string {
	var lines []string
	for _, item := range items {
		quoted := quoteRedisArg(item.Key)
		switch item.Type {
		case "string":
			if s, ok := item.Value.(string); ok {
				lines = append(lines, fmt.Sprintf("SET %s %s", quoted, quoteRedisArg(s)))
			}
		case "hash":
			if m, ok := item.Value.(map[string]string); ok {
				parts := []string{"HSET", quoted}
				for k, v := range m {
					parts = append(parts, quoteRedisArg(k), quoteRedisArg(v))
				}
				lines = append(lines, strings.Join(parts, " "))
			}
		case "list":
			if arr, ok := item.Value.([]string); ok && len(arr) > 0 {
				parts := []string{"RPUSH", quoted}
				for _, v := range arr {
					parts = append(parts, quoteRedisArg(v))
				}
				lines = append(lines, strings.Join(parts, " "))
			}
		case "set":
			if arr, ok := item.Value.([]string); ok && len(arr) > 0 {
				parts := []string{"SADD", quoted}
				for _, v := range arr {
					parts = append(parts, quoteRedisArg(v))
				}
				lines = append(lines, strings.Join(parts, " "))
			}
		case "zset":
			if arr, ok := item.Value.([]map[string]interface{}); ok && len(arr) > 0 {
				parts := []string{"ZADD", quoted}
				for _, z := range arr {
					score, _ := z["score"].(float64)
					member, _ := z["member"].(string)
					parts = append(parts, fmt.Sprintf("%g", score), quoteRedisArg(member))
				}
				lines = append(lines, strings.Join(parts, " "))
			}
		case "stream":
			if arr, ok := item.Value.([]map[string]interface{}); ok {
				for _, msg := range arr {
					id, _ := msg["id"].(string)
					values, _ := msg["values"].(map[string]interface{})
					parts := []string{"XADD", quoted, id}
					for k, v := range values {
						parts = append(parts, quoteRedisArg(k), quoteRedisArg(fmt.Sprintf("%v", v)))
					}
					lines = append(lines, strings.Join(parts, " "))
				}
			}
		}
		// TTL
		if item.TTL > 0 {
			lines = append(lines, fmt.Sprintf("EXPIRE %s %d", quoted, item.TTL))
		}
	}
	return strings.Join(lines, "\n")
}

// ========== 导入 ==========

func handleImportKeys(params json.RawMessage) (any, error) {
	var req struct {
		ConnID       string `json:"conn_id"`
		Format       string `json:"format"`        // json / csv / redis_cmd
		Content      string `json:"content"`        // 导入内容
		ConflictMode string `json:"conflict_mode"`  // skip / overwrite
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	if req.Content == "" {
		return nil, fmt.Errorf("导入内容为空")
	}
	if req.ConflictMode == "" {
		req.ConflictMode = "skip"
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 120*time.Second)
	defer cancel()

	var items []ExportKeyItem
	var parseErr error

	switch req.Format {
	case "csv":
		items, parseErr = parseCSV(req.Content)
	case "redis_cmd":
		return importRedisCommands(ctx, conn, req.Content, req.ConflictMode)
	default: // json
		items, parseErr = parseJSON(req.Content)
	}

	if parseErr != nil {
		return nil, fmt.Errorf("解析失败: %v", parseErr)
	}

	imported := 0
	skipped := 0
	failed := 0

	for _, item := range items {
		if item.Key == "" {
			continue
		}

		// 冲突检测
		if req.ConflictMode == "skip" {
			exists, _ := conn.Cmd().Exists(ctx, item.Key).Result()
			if exists > 0 {
				skipped++
				continue
			}
		} else {
			// overwrite: 先删除
			conn.Cmd().Del(ctx, item.Key)
		}

		err := writeKeyFromImport(ctx, conn.Client, item)
		if err != nil {
			failed++
			continue
		}
		imported++
	}

	AddOpLog(req.ConnID, "IMPORT", "", fmt.Sprintf("format=%s imported=%d skipped=%d failed=%d", req.Format, imported, skipped, failed))
	return map[string]interface{}{
		"imported": imported,
		"skipped":  skipped,
		"failed":   failed,
	}, nil
}

func parseJSON(content string) ([]ExportKeyItem, error) {
	var items []ExportKeyItem
	if err := json.Unmarshal([]byte(content), &items); err != nil {
		return nil, err
	}
	return items, nil
}

func parseCSV(content string) ([]ExportKeyItem, error) {
	r := csv.NewReader(strings.NewReader(content))
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	var items []ExportKeyItem
	for i, row := range records {
		if i == 0 {
			continue // 跳过表头
		}
		if len(row) < 4 {
			continue
		}
		ttl, _ := strconv.ParseInt(row[2], 10, 64)
		var value interface{}
		keyType := row[1]
		valStr := row[3]

		switch keyType {
		case "string":
			value = valStr
		case "hash":
			var m map[string]string
			if json.Unmarshal([]byte(valStr), &m) == nil {
				value = m
			}
		case "list", "set":
			var arr []string
			if json.Unmarshal([]byte(valStr), &arr) == nil {
				value = arr
			}
		case "zset":
			var arr []map[string]interface{}
			if json.Unmarshal([]byte(valStr), &arr) == nil {
				value = arr
			}
		case "stream":
			var arr []map[string]interface{}
			if json.Unmarshal([]byte(valStr), &arr) == nil {
				value = arr
			}
		default:
			continue
		}

		items = append(items, ExportKeyItem{
			Key:   row[0],
			Type:  keyType,
			TTL:   ttl,
			Value: value,
		})
	}
	return items, nil
}

func writeKeyFromImport(ctx context.Context, client redis.Cmdable, item ExportKeyItem) error {
	switch item.Type {
	case "string":
		val := ""
		if s, ok := item.Value.(string); ok {
			val = s
		}
		ttl := time.Duration(0)
		if item.TTL > 0 {
			ttl = time.Duration(item.TTL) * time.Second
		}
		return client.Set(ctx, item.Key, val, ttl).Err()

	case "hash":
		switch m := item.Value.(type) {
		case map[string]string:
			args := make([]interface{}, 0, len(m)*2)
			for k, v := range m {
				args = append(args, k, v)
			}
			if len(args) > 0 {
				if err := client.HSet(ctx, item.Key, args...).Err(); err != nil {
					return err
				}
			}
		case map[string]interface{}:
			args := make([]interface{}, 0, len(m)*2)
			for k, v := range m {
				args = append(args, k, fmt.Sprintf("%v", v))
			}
			if len(args) > 0 {
				if err := client.HSet(ctx, item.Key, args...).Err(); err != nil {
					return err
				}
			}
		}

	case "list":
		switch arr := item.Value.(type) {
		case []string:
			if len(arr) > 0 {
				vals := make([]interface{}, len(arr))
				for i, v := range arr {
					vals[i] = v
				}
				if err := client.RPush(ctx, item.Key, vals...).Err(); err != nil {
					return err
				}
			}
		case []interface{}:
			if len(arr) > 0 {
				if err := client.RPush(ctx, item.Key, arr...).Err(); err != nil {
					return err
				}
			}
		}

	case "set":
		switch arr := item.Value.(type) {
		case []string:
			if len(arr) > 0 {
				vals := make([]interface{}, len(arr))
				for i, v := range arr {
					vals[i] = v
				}
				if err := client.SAdd(ctx, item.Key, vals...).Err(); err != nil {
					return err
				}
			}
		case []interface{}:
			if len(arr) > 0 {
				if err := client.SAdd(ctx, item.Key, arr...).Err(); err != nil {
					return err
				}
			}
		}

	case "zset":
		if arr, ok := item.Value.([]interface{}); ok {
			members := make([]*redis.Z, 0, len(arr))
			for _, raw := range arr {
				if m, ok := raw.(map[string]interface{}); ok {
					score, _ := m["score"].(float64)
					member, _ := m["member"].(string)
					if member == "" {
						member = fmt.Sprintf("%v", m["member"])
					}
					members = append(members, &redis.Z{Score: score, Member: member})
				}
			}
			if len(members) > 0 {
				if err := client.ZAdd(ctx, item.Key, members...).Err(); err != nil {
					return err
				}
			}
		}

	case "stream":
		if arr, ok := item.Value.([]interface{}); ok {
			for _, raw := range arr {
				if m, ok := raw.(map[string]interface{}); ok {
					id, _ := m["id"].(string)
					if id == "" {
						id = "*"
					}
					values := map[string]interface{}{}
					if v, ok := m["values"].(map[string]interface{}); ok {
						values = v
					}
					if len(values) > 0 {
						client.XAdd(ctx, &redis.XAddArgs{
							Stream: item.Key,
							ID:     id,
							Values: values,
						})
					}
				}
			}
		}

	default:
		return fmt.Errorf("unsupported type: %s", item.Type)
	}

	// 设置 TTL（非 string 类型，string 已在 Set 时设置）
	if item.Type != "string" && item.TTL > 0 {
		client.Expire(ctx, item.Key, time.Duration(item.TTL)*time.Second)
	}

	return nil
}

// importRedisCommands 逐行执行 Redis 命令导入
func importRedisCommands(ctx context.Context, conn *RedisConn, content string, conflictMode string) (any, error) {
	lines := strings.Split(content, "\n")
	imported := 0
	skipped := 0
	failed := 0
	consecutiveFails := 0
	const maxConsecutiveFails = 50

	for _, line := range lines {
		// 检查 context 超时
		select {
		case <-ctx.Done():
			AddOpLog(conn.Config.ID, "IMPORT", "", fmt.Sprintf("format=redis_cmd imported=%d skipped=%d failed=%d (timeout)", imported, skipped, failed))
			return map[string]interface{}{
				"imported": imported,
				"skipped":  skipped,
				"failed":   failed,
			}, fmt.Errorf("导入超时，已导入 %d 个，失败 %d 个", imported, failed)
		default:
		}

		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		args := parseCommand(line)
		if len(args) == 0 {
			continue
		}

		// 冲突检测：对写入命令检查 key 是否存在
		cmd := strings.ToUpper(args[0])
		if conflictMode == "skip" && len(args) >= 2 {
			if isWriteCommand(cmd) {
				exists, _ := conn.Cmd().Exists(ctx, args[1]).Result()
				if exists > 0 && cmd != "EXPIRE" {
					skipped++
					continue
				}
			}
		}

		iArgs := make([]interface{}, len(args))
		for i, a := range args {
			iArgs[i] = a
		}

		_, err := conn.Do(ctx, iArgs...).Result()
		if err != nil && err != redis.Nil {
			failed++
			consecutiveFails++
			// 连续失败超过阈值，判定文件格式错误，快速终止
			if consecutiveFails >= maxConsecutiveFails {
				AddOpLog(conn.Config.ID, "IMPORT", "", fmt.Sprintf("format=redis_cmd imported=%d skipped=%d failed=%d (aborted: too many consecutive failures)", imported, skipped, failed))
				return map[string]interface{}{
					"imported": imported,
					"skipped":  skipped,
					"failed":   failed,
				}, fmt.Errorf("连续 %d 次命令执行失败，文件可能不是有效的 Redis 命令格式", maxConsecutiveFails)
			}
			continue
		}
		imported++
		consecutiveFails = 0 // 成功则重置连续失败计数
	}

	AddOpLog(conn.Config.ID, "IMPORT", "", fmt.Sprintf("format=redis_cmd imported=%d skipped=%d failed=%d", imported, skipped, failed))
	return map[string]interface{}{
		"imported": imported,
		"skipped":  skipped,
		"failed":   failed,
	}, nil
}

func isWriteCommand(cmd string) bool {
	switch cmd {
	case "SET", "HSET", "RPUSH", "LPUSH", "SADD", "ZADD", "XADD",
		"MSET", "SETNX", "SETEX", "PSETEX", "HMSET":
		return true
	}
	return false
}

// exportFileExt 根据导出格式返回文件扩展名
func exportFileExt(format string) string {
	switch format {
	case "csv":
		return ".csv"
	case "redis_cmd":
		return ".txt"
	default:
		return ".json"
	}
}

// handleExportKeysFile 流式导出键数据到文件（GM 环境使用，低内存占用）
func handleExportKeysFile(params json.RawMessage) (any, error) {
	var req struct {
		ConnID   string   `json:"conn_id"`
		Format   string   `json:"format"`
		Scope    string   `json:"scope"`
		Keys     []string `json:"keys"`
		Pattern  string   `json:"pattern"`
		Limit    int      `json:"limit"`
		FilePath string   `json:"file_path"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	if req.FilePath == "" {
		return nil, fmt.Errorf("导出路径不能为空")
	}

	// 校验目录是否存在
	info, err := os.Stat(req.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("目录不存在: %s", req.FilePath)
		}
		return nil, fmt.Errorf("无法访问目录: %v", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("路径不是目录: %s", req.FilePath)
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	if req.Format == "" {
		req.Format = "json"
	}
	if req.Limit <= 0 || req.Limit > 50000 {
		req.Limit = 50000
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 120*time.Second)
	defer cancel()

	exportKeys, err := scanKeysForExport(ctx, conn, req.Scope, req.Keys, req.Pattern, req.Limit)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	fileName := fmt.Sprintf("easy_rdm_keys_%s%s",
		now.Format("20060102_150405_000"),
		exportFileExt(req.Format),
	)
	fullPath := filepath.Join(req.FilePath, fileName)

	file, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %v", err)
	}
	defer file.Close()

	count, err := streamExportKeys(file, conn, req.Format, exportKeys)
	if err != nil {
		os.Remove(fullPath)
		return nil, fmt.Errorf("写入文件失败: %v", err)
	}

	AddOpLog(req.ConnID, "EXPORT", "", fmt.Sprintf("format=%s count=%d file=%s", req.Format, count, fullPath))
	return map[string]interface{}{
		"path":  fullPath,
		"count": count,
	}, nil
}

// handleImportKeysFile 从文件读取内容并导入（GM 环境使用）
func handleImportKeysFile(params json.RawMessage) (any, error) {
	var req struct {
		ConnID       string `json:"conn_id"`
		Format       string `json:"format"`
		FilePath     string `json:"file_path"`
		ConflictMode string `json:"conflict_mode"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	if req.FilePath == "" {
		return nil, fmt.Errorf("文件路径不能为空")
	}

	// 检查文件大小，限制 50MB
	info, err := os.Stat(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}
	const maxImportSize = 50 * 1024 * 1024
	if info.Size() > maxImportSize {
		return nil, fmt.Errorf("文件过大（%.1fMB），最大支持 50MB", float64(info.Size())/1024/1024)
	}

	data, err := os.ReadFile(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}

	importParams, _ := json.Marshal(map[string]interface{}{
		"conn_id":       req.ConnID,
		"format":        req.Format,
		"content":       string(data),
		"conflict_mode": req.ConflictMode,
	})
	return handleImportKeys(importParams)
}

// ========== 流式导出 ==========

// scanKeysForExport 根据 scope 收集要导出的 key 列表
func scanKeysForExport(ctx context.Context, conn *RedisConn, scope string, keys []string, pattern string, limit int) ([]string, error) {
	switch scope {
	case "selected":
		return keys, nil
	case "pattern":
		if pattern == "" {
			pattern = "*"
		}
		return scanAllKeys(ctx, conn, pattern, limit)
	default:
		return scanAllKeys(ctx, conn, "*", limit)
	}
}

func scanAllKeys(ctx context.Context, conn *RedisConn, pattern string, limit int) ([]string, error) {
	var keys []string
	var cursor uint64
	for {
		batch, next, err := conn.Cmd().Scan(ctx, cursor, pattern, 500).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		if len(keys) >= limit {
			keys = keys[:limit]
			break
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

// streamExportKeys 流式写入导出数据到 io.Writer，每次只在内存中保留单个 key 的数据
func streamExportKeys(w io.Writer, conn *RedisConn, format string, keys []string) (int, error) {
	ctx, cancel := context.WithTimeout(conn.Ctx, 120*time.Second)
	defer cancel()

	count := 0

	switch format {
	case "csv":
		csvW := csv.NewWriter(w)
		csvW.Write([]string{"key", "type", "ttl", "value"})
		for _, key := range keys {
			item, err := readKeyForExport(ctx, conn.Client, key)
			if err != nil || item == nil {
				continue
			}
			var valStr string
			switch v := item.Value.(type) {
			case string:
				valStr = v
			default:
				data, _ := json.Marshal(v)
				valStr = string(data)
			}
			csvW.Write([]string{item.Key, item.Type, strconv.FormatInt(item.TTL, 10), valStr})
			count++
			if count%100 == 0 {
				csvW.Flush()
			}
		}
		csvW.Flush()

	case "redis_cmd":
		for _, key := range keys {
			item, err := readKeyForExport(ctx, conn.Client, key)
			if err != nil || item == nil {
				continue
			}
			cmds := formatRedisCommandsForItem(*item)
			for _, cmd := range cmds {
				fmt.Fprintln(w, cmd)
			}
			count++
		}

	default: // json
		fmt.Fprint(w, "[\n")
		for _, key := range keys {
			item, err := readKeyForExport(ctx, conn.Client, key)
			if err != nil || item == nil {
				continue
			}
			if count > 0 {
				fmt.Fprint(w, ",\n")
			}
			data, _ := json.Marshal(item)
			fmt.Fprintf(w, "  %s", data)
			count++
		}
		fmt.Fprint(w, "\n]")
	}

	return count, nil
}

// formatRedisCommandsForItem 为单个 key 生成 Redis 命令
func formatRedisCommandsForItem(item ExportKeyItem) []string {
	var lines []string
	quoted := quoteRedisArg(item.Key)

	switch item.Type {
	case "string":
		if s, ok := item.Value.(string); ok {
			lines = append(lines, fmt.Sprintf("SET %s %s", quoted, quoteRedisArg(s)))
		}
	case "hash":
		if m, ok := item.Value.(map[string]string); ok {
			parts := []string{"HSET", quoted}
			for k, v := range m {
				parts = append(parts, quoteRedisArg(k), quoteRedisArg(v))
			}
			lines = append(lines, strings.Join(parts, " "))
		}
	case "list":
		if arr, ok := item.Value.([]string); ok && len(arr) > 0 {
			parts := []string{"RPUSH", quoted}
			for _, v := range arr {
				parts = append(parts, quoteRedisArg(v))
			}
			lines = append(lines, strings.Join(parts, " "))
		}
	case "set":
		if arr, ok := item.Value.([]string); ok && len(arr) > 0 {
			parts := []string{"SADD", quoted}
			for _, v := range arr {
				parts = append(parts, quoteRedisArg(v))
			}
			lines = append(lines, strings.Join(parts, " "))
		}
	case "zset":
		if arr, ok := item.Value.([]map[string]interface{}); ok && len(arr) > 0 {
			parts := []string{"ZADD", quoted}
			for _, z := range arr {
				score, _ := z["score"].(float64)
				member, _ := z["member"].(string)
				parts = append(parts, fmt.Sprintf("%g", score), quoteRedisArg(member))
			}
			lines = append(lines, strings.Join(parts, " "))
		}
	case "stream":
		if arr, ok := item.Value.([]map[string]interface{}); ok {
			for _, msg := range arr {
				id, _ := msg["id"].(string)
				values, _ := msg["values"].(map[string]interface{})
				parts := []string{"XADD", quoted, id}
				for k, v := range values {
					parts = append(parts, quoteRedisArg(k), quoteRedisArg(fmt.Sprintf("%v", v)))
				}
				lines = append(lines, strings.Join(parts, " "))
			}
		}
	}

	if item.TTL > 0 {
		lines = append(lines, fmt.Sprintf("EXPIRE %s %d", quoted, item.TTL))
	}

	return lines
}

// StreamExportToWriter 流式导出数据到 io.Writer（供 HTTP 端点使用）
func StreamExportToWriter(w io.Writer, connID, format, scope string, keys []string, pattern string, limit int) (int, error) {
	conn, ok := GetConn(connID)
	if !ok {
		return 0, fmt.Errorf("连接不存在")
	}

	if format == "" {
		format = "json"
	}
	if limit <= 0 || limit > 50000 {
		limit = 50000
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 120*time.Second)
	defer cancel()

	exportKeys, err := scanKeysForExport(ctx, conn, scope, keys, pattern, limit)
	if err != nil {
		return 0, err
	}

	count, err := streamExportKeys(w, conn, format, exportKeys)
	if err != nil {
		return count, err
	}

	AddOpLog(connID, "EXPORT", "", fmt.Sprintf("format=%s count=%d", format, count))
	return count, nil
}

// ExportFileExt 根据格式返回文件扩展名（导出供 main.go 使用）
func ExportFileExt(format string) string {
	return exportFileExt(format)
}
