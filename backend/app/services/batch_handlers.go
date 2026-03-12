package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RegisterBatchHandlers 注册批量操作和数据迁移相关的 RPC 方法
func RegisterBatchHandlers(register func(string, RPCHandlerFunc)) {
	register("batch_set_ttl", handleBatchSetTTL)
	register("batch_move_db", handleBatchMoveDB)
	register("migrate_keys", handleMigrateKeys)
}

// handleBatchSetTTL 批量设置/移除 TTL
func handleBatchSetTTL(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string   `json:"conn_id"`
		Keys   []string `json:"keys"`
		TTL    int64    `json:"ttl"` // -1=移除TTL(persist), >0=设置秒数
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	if len(req.Keys) == 0 {
		return nil, fmt.Errorf("keys 不能为空")
	}
	if len(req.Keys) > 10000 {
		return nil, fmt.Errorf("单次最多 10000 个键")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 60*time.Second)
	defer cancel()

	success := 0
	failed := 0
	pipe := conn.Cmd().Pipeline()

	for _, key := range req.Keys {
		if req.TTL <= 0 {
			pipe.Persist(ctx, key)
		} else {
			pipe.Expire(ctx, key, time.Duration(req.TTL)*time.Second)
		}
	}

	cmds, _ := pipe.Exec(ctx)
	for _, cmd := range cmds {
		if cmd.Err() == nil {
			success++
		} else {
			failed++
		}
	}

	action := "BATCH_PERSIST"
	detail := fmt.Sprintf("persist %d keys", len(req.Keys))
	if req.TTL > 0 {
		action = "BATCH_SET_TTL"
		detail = fmt.Sprintf("TTL=%ds, %d keys", req.TTL, len(req.Keys))
	}
	AddOpLog(req.ConnID, action, "", detail)

	return map[string]interface{}{
		"success": success,
		"failed":  failed,
	}, nil
}

// handleBatchMoveDB 批量移动键到另一个 DB（使用 DUMP/RESTORE）
func handleBatchMoveDB(params json.RawMessage) (any, error) {
	var req struct {
		ConnID   string   `json:"conn_id"`
		Keys     []string `json:"keys"`
		TargetDB int      `json:"target_db"`
		Replace  bool     `json:"replace"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	if len(req.Keys) == 0 {
		return nil, fmt.Errorf("keys 不能为空")
	}
	if len(req.Keys) > 5000 {
		return nil, fmt.Errorf("单次最多 5000 个键")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	// 集群模式不支持多 DB
	if conn.IsCluster {
		return nil, fmt.Errorf("集群模式不支持跨 DB 移动")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 120*time.Second)
	defer cancel()

	// 获取当前 DB
	currentDB := conn.Config.DB

	if req.TargetDB == currentDB {
		return nil, fmt.Errorf("目标 DB 与当前 DB 相同")
	}

	// 创建目标 DB 的独立 Client，不污染连接池
	targetOpts := conn.Client.Options()
	targetOpts.DB = req.TargetDB
	targetClient := redis.NewClient(targetOpts)
	defer targetClient.Close()

	// 验证目标 DB 可连接
	if err := targetClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接目标 DB%d 失败: %v", req.TargetDB, err)
	}

	success := 0
	failed := 0
	var errors []string

	for _, key := range req.Keys {
		// DUMP
		dump, err := conn.Cmd().Dump(ctx, key).Result()
		if err != nil {
			failed++
			if len(errors) < 5 {
				errors = append(errors, fmt.Sprintf("%s: dump failed: %s", key, err.Error()))
			}
			continue
		}

		// 获取 TTL
		ttl, _ := conn.Cmd().PTTL(ctx, key).Result()
		if ttl < 0 {
			ttl = 0 // 永久
		}

		// 在目标 DB 上 RESTORE
		restoreArgs := []interface{}{"RESTORE", key, int64(ttl / time.Millisecond), dump}
		if req.Replace {
			restoreArgs = append(restoreArgs, "REPLACE")
		}
		if err := targetClient.Do(ctx, restoreArgs...).Err(); err != nil {
			failed++
			if len(errors) < 5 {
				errors = append(errors, fmt.Sprintf("%s: restore failed: %s", key, err.Error()))
			}
			continue
		}

		// 删除原键
		conn.Cmd().Del(ctx, key)
		success++
	}

	AddOpLog(req.ConnID, "BATCH_MOVE_DB", "", fmt.Sprintf("db%d→db%d, %d/%d keys", currentDB, req.TargetDB, success, len(req.Keys)))

	result := map[string]interface{}{
		"success": success,
		"failed":  failed,
	}
	if len(errors) > 0 {
		result["errors"] = errors
	}
	return result, nil
}

// handleMigrateKeys 跨连接迁移键（DUMP/RESTORE）
func handleMigrateKeys(params json.RawMessage) (any, error) {
	var req struct {
		SourceConnID string   `json:"source_conn_id"`
		TargetConnID string   `json:"target_conn_id"`
		Keys         []string `json:"keys"`
		Pattern      string   `json:"pattern"`
		Replace      bool     `json:"replace"`
		Limit        int      `json:"limit"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	srcConn, ok := GetConn(req.SourceConnID)
	if !ok {
		return nil, fmt.Errorf("源连接不存在")
	}
	dstConn, ok := GetConn(req.TargetConnID)
	if !ok {
		return nil, fmt.Errorf("目标连接不存在")
	}

	ctx, cancel := context.WithTimeout(srcConn.Ctx, 300*time.Second)
	defer cancel()

	// 如果指定了 pattern 而非 keys，先扫描
	keys := req.Keys
	if len(keys) == 0 && req.Pattern != "" {
		limit := req.Limit
		if limit <= 0 || limit > 50000 {
			limit = 50000
		}
		var cursor uint64
		for {
			batch, next, err := srcConn.Cmd().Scan(ctx, cursor, req.Pattern, 500).Result()
			if err != nil {
				return nil, fmt.Errorf("扫描源键失败: %v", err)
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
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("没有要迁移的键")
	}

	// 检查版本兼容性
	srcInfo, _ := srcConn.Cmd().Info(ctx, "server").Result()
	dstInfo, _ := dstConn.Cmd().Info(ctx, "server").Result()
	srcVer := extractVersion(srcInfo)
	dstVer := extractVersion(dstInfo)

	success := 0
	failed := 0
	skipped := 0
	var errors []string

	for _, key := range keys {
		// DUMP from source
		dump, err := srcConn.Cmd().Dump(ctx, key).Result()
		if err != nil {
			failed++
			if len(errors) < 10 {
				errors = append(errors, fmt.Sprintf("%s: dump failed", key))
			}
			continue
		}

		// 获取 TTL
		ttl, _ := srcConn.Cmd().PTTL(ctx, key).Result()
		if ttl < 0 {
			ttl = 0
		}

		// RESTORE to target
		restoreArgs := []interface{}{"RESTORE", key, int64(ttl / time.Millisecond), dump}
		if req.Replace {
			restoreArgs = append(restoreArgs, "REPLACE")
		}
		if err := dstConn.Do(ctx, restoreArgs...).Err(); err != nil {
			// 如果是 BUSYKEY 且不 replace，算 skipped
			if !req.Replace {
				skipped++
			} else {
				failed++
				if len(errors) < 10 {
					errors = append(errors, fmt.Sprintf("%s: %s", key, err.Error()))
				}
			}
			continue
		}
		success++
	}

	AddOpLog(req.SourceConnID, "MIGRATE_OUT", "", fmt.Sprintf("→%s, %d/%d keys", req.TargetConnID, success, len(keys)))
	AddOpLog(req.TargetConnID, "MIGRATE_IN", "", fmt.Sprintf("←%s, %d/%d keys", req.SourceConnID, success, len(keys)))

	result := map[string]interface{}{
		"success":     success,
		"failed":      failed,
		"skipped":     skipped,
		"total":       len(keys),
		"source_ver":  srcVer,
		"target_ver":  dstVer,
	}
	if len(errors) > 0 {
		result["errors"] = errors
	}
	return result, nil
}

// extractVersion 从 INFO server 输出中提取 redis_version
func extractVersion(info string) string {
	sections := parseInfoSections(info)
	if server, ok := sections["Server"]; ok {
		if v, ok := server["redis_version"]; ok {
			return v
		}
	}
	return "unknown"
}
