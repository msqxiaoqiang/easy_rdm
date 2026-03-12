package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"


	"github.com/go-redis/redis/v8"
)

// RegisterMemoryHandlers 注册内存分析相关的 RPC 方法
func RegisterMemoryHandlers(register func(string, RPCHandlerFunc)) {
	register("memory_scan", handleMemoryScan)
	register("memory_distribution", handleMemoryDistribution)
	register("get_keys_memory", handleGetKeysMemory)
}

// MemoryKeyInfo 大 Key 扫描结果
type MemoryKeyInfo struct {
	Key    string `json:"key"`
	Type   string `json:"type"`
	Memory int64  `json:"memory"` // bytes
	TTL    int64  `json:"ttl"`    // seconds, -1 = no expiry
}

func handleMemoryScan(params json.RawMessage) (any, error) {
	var req struct {
		ConnID  string `json:"conn_id"`
		Pattern string `json:"pattern"`
		Limit   int    `json:"limit"`    // 扫描 key 上限
		TopN    int    `json:"top_n"`    // 返回前 N 个大 key
		MinSize int64  `json:"min_size"` // 最小字节数过滤
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
	if req.Limit <= 0 || req.Limit > 100000 {
		req.Limit = 10000
	}
	if req.TopN <= 0 || req.TopN > 500 {
		req.TopN = 100
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 120*time.Second)
	defer cancel()

	// SCAN 所有 key 并获取 MEMORY USAGE
	var results []MemoryKeyInfo
	var cursor uint64
	scanned := 0

	for {
		batch, next, err := conn.Cmd().Scan(ctx, cursor, req.Pattern, 200).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range batch {
			scanned++
			if scanned > req.Limit {
				break
			}

			// MEMORY USAGE
			memResult, err := conn.Cmd().MemoryUsage(ctx, key).Result()
			if err != nil {
				continue
			}

			if req.MinSize > 0 && memResult < req.MinSize {
				continue
			}

			// TYPE + TTL
			pipe := conn.Cmd().Pipeline()
			typeCmd := pipe.Type(ctx, key)
			ttlCmd := pipe.TTL(ctx, key)
			pipe.Exec(ctx)

			keyType := typeCmd.Val()
			ttlVal := int64(-1)
			if d := ttlCmd.Val(); d > 0 {
				ttlVal = int64(d.Seconds())
			}

			results = append(results, MemoryKeyInfo{
				Key:    key,
				Type:   keyType,
				Memory: memResult,
				TTL:    ttlVal,
			})
		}

		if scanned >= req.Limit {
			break
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}

	// 按内存降序排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Memory > results[j].Memory
	})

	// 截取 TopN
	if len(results) > req.TopN {
		results = results[:req.TopN]
	}

	// 总内存
	var totalMemory int64
	for _, r := range results {
		totalMemory += r.Memory
	}

	AddOpLog(req.ConnID, "MEMORY_SCAN", "", fmt.Sprintf("scanned=%d found=%d", scanned, len(results)))
	return map[string]interface{}{
		"keys":         results,
		"scanned":      scanned,
		"total_memory": totalMemory,
	}, nil
}

func handleMemoryDistribution(params json.RawMessage) (any, error) {
	var req struct {
		ConnID    string `json:"conn_id"`
		ScanLimit int    `json:"scan_limit"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	if req.ScanLimit <= 0 || req.ScanLimit > 50000 {
		req.ScanLimit = 10000
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 120*time.Second)
	defer cancel()

	// 按类型统计
	typeStats := make(map[string]struct {
		Count  int   `json:"count"`
		Memory int64 `json:"memory"`
	})

	// 按前缀统计（取第一个分隔符前的部分）
	prefixStats := make(map[string]struct {
		Count  int   `json:"count"`
		Memory int64 `json:"memory"`
	})

	var cursor uint64
	scanned := 0

	for {
		batch, next, err := conn.Cmd().Scan(ctx, cursor, "*", 200).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range batch {
			scanned++
			if scanned > req.ScanLimit {
				break
			}

			memResult, err := conn.Cmd().MemoryUsage(ctx, key).Result()
			if err != nil {
				continue
			}

			keyType, _ := conn.Cmd().Type(ctx, key).Result()

			// 类型统计
			ts := typeStats[keyType]
			ts.Count++
			ts.Memory += memResult
			typeStats[keyType] = ts

			// 前缀统计
			prefix := extractPrefix(key)
			ps := prefixStats[prefix]
			ps.Count++
			ps.Memory += memResult
			prefixStats[prefix] = ps
		}

		if scanned >= req.ScanLimit {
			break
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}

	// 转换为数组格式
	typeArr := make([]map[string]interface{}, 0, len(typeStats))
	for t, s := range typeStats {
		typeArr = append(typeArr, map[string]interface{}{
			"type":   t,
			"count":  s.Count,
			"memory": s.Memory,
		})
	}
	sort.Slice(typeArr, func(i, j int) bool {
		return typeArr[i]["memory"].(int64) > typeArr[j]["memory"].(int64)
	})

	prefixArr := make([]map[string]interface{}, 0, len(prefixStats))
	for p, s := range prefixStats {
		prefixArr = append(prefixArr, map[string]interface{}{
			"prefix": p,
			"count":  s.Count,
			"memory": s.Memory,
		})
	}
	sort.Slice(prefixArr, func(i, j int) bool {
		return prefixArr[i]["memory"].(int64) > prefixArr[j]["memory"].(int64)
	})
	// 只返回前 30 个前缀
	if len(prefixArr) > 30 {
		prefixArr = prefixArr[:30]
	}

	return map[string]interface{}{
		"scanned":  scanned,
		"by_type":  typeArr,
		"by_prefix": prefixArr,
	}, nil
}

func extractPrefix(key string) string {
	for _, sep := range []string{":", "/", ".", "-", "_"} {
		if idx := strings.Index(key, sep); idx > 0 {
			return key[:idx]
		}
	}
	return "(no prefix)"
}

// handleGetKeysMemory 批量获取指定 key 的内存占用
func handleGetKeysMemory(params json.RawMessage) (any, error) {
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

	if len(req.Keys) == 0 || len(req.Keys) > 5000 {
		return nil, fmt.Errorf("keys 数量需在 1-5000 之间")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 30*time.Second)
	defer cancel()

	result := make(map[string]int64, len(req.Keys))

	// 分批 Pipeline 执行 MEMORY USAGE，每批 500 个
	const batchSize = 500
	for i := 0; i < len(req.Keys); i += batchSize {
		end := i + batchSize
		if end > len(req.Keys) {
			end = len(req.Keys)
		}
		batch := req.Keys[i:end]

		pipe := conn.Cmd().Pipeline()
		cmds := make([]*redis.Cmd, len(batch))
		for j, key := range batch {
			cmds[j] = pipe.Do(ctx, "MEMORY", "USAGE", key)
		}
		pipe.Exec(ctx)

		for j, key := range batch {
			val, err := cmds[j].Int64()
			if err != nil {
				result[key] = -1
			} else {
				result[key] = val
			}
		}
	}

	return result, nil
}
