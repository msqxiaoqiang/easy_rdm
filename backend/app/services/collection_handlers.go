package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"


	"github.com/go-redis/redis/v8"
)

// RegisterCollectionHandlers 注册集合类型字段级操作 API
func RegisterCollectionHandlers(register func(string, RPCHandlerFunc)) {
	// Hash
	register("hscan_fields", handleHScanFields)
	register("hset_field", handleHSetField)
	register("hdel_fields", handleHDelFields)
	// List
	register("lrange_values", handleLRangeValues)
	register("lset_value", handleLSetValue)
	register("list_push", handleListPush)
	register("list_remove", handleListRemove)
	// Set
	register("sscan_members", handleSScanMembers)
	register("sadd_members", handleSAddMembers)
	register("srem_members", handleSRemMembers)
	// ZSet
	register("zscan_members", handleZScanMembers)
	register("zadd_members", handleZAddMembers)
	register("zrem_members", handleZRemMembers)
	register("zrange_members", handleZRangeMembers)
}

func getConnOrErr(connID string) (*RedisConn, error) {
	conn, ok := GetConn(connID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}
	return conn, nil
}

// ========== Hash ==========

func handleHScanFields(params json.RawMessage) (any, error) {
	var req struct {
		ConnID  string `json:"conn_id"`
		Key     string `json:"key"`
		Pattern string `json:"pattern"`
		Count   int64  `json:"count"`
		Cursor  uint64 `json:"cursor"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	conn, err := getConnOrErr(req.ConnID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	if req.Pattern == "" {
		req.Pattern = "*"
	}
	if req.Count <= 0 {
		req.Count = 100
	}

	fields, nextCursor, err := conn.Cmd().HScan(ctx, req.Key, req.Cursor, req.Pattern, req.Count).Result()
	if err != nil {
		return nil, err
	}

	// HScan 返回 [field1, value1, field2, value2, ...]
	type FieldItem struct {
		Field string `json:"field"`
		Value string `json:"value"`
	}
	items := make([]FieldItem, 0, len(fields)/2)
	for i := 0; i+1 < len(fields); i += 2 {
		items = append(items, FieldItem{Field: fields[i], Value: fields[i+1]})
	}

	AddOpLog(req.ConnID, "HSCAN", req.Key, fmt.Sprintf("pattern=%s count=%d", req.Pattern, len(items)))
	return map[string]interface{}{
		"fields": items,
		"cursor": nextCursor,
	}, nil
}

func handleHSetField(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Field  string `json:"field"`
		Value  string `json:"value"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	conn, err := getConnOrErr(req.ConnID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	if err := conn.Cmd().HSet(ctx, req.Key, req.Field, req.Value).Err(); err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "HSET", req.Key, "field="+req.Field)
	return nil, nil
}

func handleHDelFields(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string   `json:"conn_id"`
		Key    string   `json:"key"`
		Fields []string `json:"fields"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	conn, err := getConnOrErr(req.ConnID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	deleted, err := conn.Cmd().HDel(ctx, req.Key, req.Fields...).Result()
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "HDEL", req.Key, fmt.Sprintf("%d fields", deleted))
	return map[string]interface{}{"deleted": deleted}, nil
}

// ========== List ==========

func handleLRangeValues(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Start  int64  `json:"start"`
		Stop   int64  `json:"stop"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	conn, err := getConnOrErr(req.ConnID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	values, err := conn.Cmd().LRange(ctx, req.Key, req.Start, req.Stop).Result()
	if err != nil {
		return nil, err
	}

	total, _ := conn.Cmd().LLen(ctx, req.Key).Result()

	AddOpLog(req.ConnID, "LRANGE", req.Key, fmt.Sprintf("start=%d stop=%d", req.Start, req.Stop))
	return map[string]interface{}{
		"values": values,
		"total":  total,
	}, nil
}

func handleLSetValue(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Index  int64  `json:"index"`
		Value  string `json:"value"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	conn, err := getConnOrErr(req.ConnID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	if err := conn.Cmd().LSet(ctx, req.Key, req.Index, req.Value).Err(); err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "LSET", req.Key, fmt.Sprintf("index=%d", req.Index))
	return nil, nil
}

func handleListPush(params json.RawMessage) (any, error) {
	var req struct {
		ConnID   string   `json:"conn_id"`
		Key      string   `json:"key"`
		Values   []string `json:"values"`
		Position string   `json:"position"` // "head" or "tail"
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	conn, err := getConnOrErr(req.ConnID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	ivals := make([]interface{}, len(req.Values))
	for i, v := range req.Values {
		ivals[i] = v
	}

	if req.Position == "head" {
		err = conn.Cmd().LPush(ctx, req.Key, ivals...).Err()
	} else {
		err = conn.Cmd().RPush(ctx, req.Key, ivals...).Err()
	}
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "LPUSH", req.Key, fmt.Sprintf("%s ×%d", req.Position, len(req.Values)))
	return nil, nil
}

func handleListRemove(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Index  int64  `json:"index"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	conn, err := getConnOrErr(req.ConnID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	// Redis 没有按索引删除的命令，用占位符+LREM 实现
	placeholder := "__DELETED_" + strconv.FormatInt(time.Now().UnixNano(), 36) + "__"
	if err := conn.Cmd().LSet(ctx, req.Key, req.Index, placeholder).Err(); err != nil {
		return nil, err
	}
	if err := conn.Cmd().LRem(ctx, req.Key, 1, placeholder).Err(); err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "LREM", req.Key, fmt.Sprintf("index=%d", req.Index))
	return nil, nil
}

// ========== Set ==========

func handleSScanMembers(params json.RawMessage) (any, error) {
	var req struct {
		ConnID  string `json:"conn_id"`
		Key     string `json:"key"`
		Pattern string `json:"pattern"`
		Count   int64  `json:"count"`
		Cursor  uint64 `json:"cursor"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	conn, err := getConnOrErr(req.ConnID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	if req.Pattern == "" {
		req.Pattern = "*"
	}
	if req.Count <= 0 {
		req.Count = 100
	}

	members, nextCursor, err := conn.Cmd().SScan(ctx, req.Key, req.Cursor, req.Pattern, req.Count).Result()
	if err != nil {
		return nil, err
	}

	AddOpLog(req.ConnID, "SSCAN", req.Key, fmt.Sprintf("pattern=%s count=%d", req.Pattern, len(members)))
	return map[string]interface{}{
		"members": members,
		"cursor":  nextCursor,
	}, nil
}

func handleSAddMembers(params json.RawMessage) (any, error) {
	var req struct {
		ConnID  string   `json:"conn_id"`
		Key     string   `json:"key"`
		Members []string `json:"members"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	conn, err := getConnOrErr(req.ConnID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	ivals := make([]interface{}, len(req.Members))
	for i, m := range req.Members {
		ivals[i] = m
	}

	added, err := conn.Cmd().SAdd(ctx, req.Key, ivals...).Result()
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "SADD", req.Key, fmt.Sprintf("%d members", added))
	return map[string]interface{}{"added": added}, nil
}

func handleSRemMembers(params json.RawMessage) (any, error) {
	var req struct {
		ConnID  string   `json:"conn_id"`
		Key     string   `json:"key"`
		Members []string `json:"members"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	conn, err := getConnOrErr(req.ConnID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	ivals := make([]interface{}, len(req.Members))
	for i, m := range req.Members {
		ivals[i] = m
	}

	removed, err := conn.Cmd().SRem(ctx, req.Key, ivals...).Result()
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "SREM", req.Key, fmt.Sprintf("%d members", removed))
	return map[string]interface{}{"removed": removed}, nil
}

// ========== ZSet ==========

func handleZScanMembers(params json.RawMessage) (any, error) {
	var req struct {
		ConnID  string `json:"conn_id"`
		Key     string `json:"key"`
		Pattern string `json:"pattern"`
		Count   int64  `json:"count"`
		Cursor  uint64 `json:"cursor"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	conn, err := getConnOrErr(req.ConnID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	if req.Pattern == "" {
		req.Pattern = "*"
	}
	if req.Count <= 0 {
		req.Count = 100
	}

	results, nextCursor, err := conn.Cmd().ZScan(ctx, req.Key, req.Cursor, req.Pattern, req.Count).Result()
	if err != nil {
		return nil, err
	}

	// ZScan 返回 [member1, score1, member2, score2, ...]
	type ZItem struct {
		Member string  `json:"member"`
		Score  float64 `json:"score"`
	}
	items := make([]ZItem, 0, len(results)/2)
	for i := 0; i+1 < len(results); i += 2 {
		score, _ := strconv.ParseFloat(results[i+1], 64)
		items = append(items, ZItem{Member: results[i], Score: score})
	}

	AddOpLog(req.ConnID, "ZSCAN", req.Key, fmt.Sprintf("pattern=%s count=%d", req.Pattern, len(items)))
	return map[string]interface{}{
		"members": items,
		"cursor":  nextCursor,
	}, nil
}

func handleZRangeMembers(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Start  int64  `json:"start"`
		Stop   int64  `json:"stop"`
		Rev    bool   `json:"rev"` // true=降序
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	conn, err := getConnOrErr(req.ConnID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	var zMembers []redis.Z
	if req.Rev {
		zMembers, err = conn.Cmd().ZRevRangeWithScores(ctx, req.Key, req.Start, req.Stop).Result()
	} else {
		zMembers, err = conn.Cmd().ZRangeWithScores(ctx, req.Key, req.Start, req.Stop).Result()
	}
	if err != nil {
		return nil, err
	}

	total, _ := conn.Cmd().ZCard(ctx, req.Key).Result()

	type ZItem struct {
		Member string  `json:"member"`
		Score  float64 `json:"score"`
	}
	items := make([]ZItem, len(zMembers))
	for i, z := range zMembers {
		items[i] = ZItem{Member: z.Member.(string), Score: z.Score}
	}

	AddOpLog(req.ConnID, "ZRANGE", req.Key, fmt.Sprintf("start=%d stop=%d", req.Start, req.Stop))
	return map[string]interface{}{
		"members": items,
		"total":   total,
	}, nil
}

func handleZAddMembers(params json.RawMessage) (any, error) {
	var req struct {
		ConnID  string `json:"conn_id"`
		Key     string `json:"key"`
		Mode    string `json:"mode"` // "" = overwrite (default), "nx" = ignore existing
		Members []struct {
			Member string  `json:"member"`
			Score  float64 `json:"score"`
		} `json:"members"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	conn, err := getConnOrErr(req.ConnID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	zMembers := make([]redis.Z, len(req.Members))
	for i, m := range req.Members {
		zMembers[i] = redis.Z{Score: m.Score, Member: m.Member}
	}

	var added int64
	if req.Mode == "nx" {
		added, err = conn.Cmd().ZAddArgs(ctx, req.Key, redis.ZAddArgs{
			NX:      true,
			Members: zMembers,
		}).Result()
	} else {
		ptrs := make([]*redis.Z, len(zMembers))
		for i := range zMembers {
			ptrs[i] = &zMembers[i]
		}
		added, err = conn.Cmd().ZAdd(ctx, req.Key, ptrs...).Result()
	}
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "ZADD", req.Key, fmt.Sprintf("%d members", len(req.Members)))
	return map[string]interface{}{"added": added}, nil
}

func handleZRemMembers(params json.RawMessage) (any, error) {
	var req struct {
		ConnID  string   `json:"conn_id"`
		Key     string   `json:"key"`
		Members []string `json:"members"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	conn, err := getConnOrErr(req.ConnID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	ivals := make([]interface{}, len(req.Members))
	for i, m := range req.Members {
		ivals[i] = m
	}

	removed, err := conn.Cmd().ZRem(ctx, req.Key, ivals...).Result()
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "ZREM", req.Key, fmt.Sprintf("%d members", removed))
	return map[string]interface{}{"removed": removed}, nil
}
