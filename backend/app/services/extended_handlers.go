package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"


	"github.com/go-redis/redis/v8"
)

// RegisterExtendedHandlers 注册扩展类型操作 API（Stream/Geo/Bitmap/Bitfield/HLL/JSON）
func RegisterExtendedHandlers(register func(string, RPCHandlerFunc)) {
	// RedisJSON
	register("json_get", handleJSONGet)
	register("json_set", handleJSONSet)
	register("json_del", handleJSONDel)
	register("json_type", handleJSONType)
	register("json_arrappend", handleJSONArrAppend)
	// Stream
	register("xrange_messages", handleXRangeMessages)
	register("xadd_message", handleXAddMessage)
	register("xdel_messages", handleXDelMessages)
	register("xtrim_stream", handleXTrimStream)
	register("xinfo_stream", handleXInfoStream)
	register("xinfo_groups", handleXInfoGroups)
	register("xgroup_create", handleXGroupCreate)
	register("xgroup_destroy", handleXGroupDestroy)
	// HyperLogLog
	register("pfcount", handlePFCount)
	register("pfadd", handlePFAdd)
	register("pfmerge", handlePFMerge)
	// Geospatial
	register("geo_members", handleGeoMembers)
	register("geo_add", handleGeoAdd)
	register("geo_dist", handleGeoDist)
	register("geo_search", handleGeoSearch)
	// Bitmap
	register("bitmap_get_range", handleBitmapGetRange)
	register("bitmap_set_bit", handleBitmapSetBit)
	register("bitmap_count", handleBitmapCount)
	// Bitfield
	register("bitfield_get", handleBitfieldGet)
	register("bitfield_set", handleBitfieldSet)
	register("bitfield_incrby", handleBitfieldIncrBy)
}

// ========== Stream ==========

func handleXRangeMessages(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Start  string `json:"start"`
		End    string `json:"end"`
		Count  int64  `json:"count"`
		Rev    bool   `json:"rev"`
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

	if req.Start == "" {
		req.Start = "-"
	}
	if req.End == "" {
		req.End = "+"
	}
	if req.Count <= 0 {
		req.Count = 100
	}

	// 多取一条用于判断是否有下一页
	fetchCount := req.Count + 1

	var vals []redis.XMessage
	if req.Rev {
		vals, err = conn.Cmd().XRevRangeN(ctx, req.Key, req.End, req.Start, fetchCount).Result()
	} else {
		vals, err = conn.Cmd().XRangeN(ctx, req.Key, req.Start, req.End, fetchCount).Result()
	}
	if err != nil {
		return nil, err
	}

	hasMore := int64(len(vals)) > req.Count
	if hasMore {
		vals = vals[:req.Count]
	}

	rawMessages := make([]map[string]interface{}, len(vals))
	for i, m := range vals {
		rawMessages[i] = map[string]interface{}{
			"id":     m.ID,
			"fields": m.Values,
		}
	}

	length, _ := conn.Cmd().XLen(ctx, req.Key).Result()

	AddOpLog(req.ConnID, "XRANGE", req.Key, fmt.Sprintf("start=%s end=%s count=%d", req.Start, req.End, len(rawMessages)))
	return map[string]interface{}{
		"messages": rawMessages,
		"total":    length,
		"has_more": hasMore,
	}, nil
}

func handleXAddMessage(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string            `json:"conn_id"`
		Key    string            `json:"key"`
		ID     string            `json:"id"`
		Fields map[string]string `json:"fields"`
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

	if req.ID == "" {
		req.ID = "*"
	}
	values := make(map[string]interface{}, len(req.Fields))
	for k, v := range req.Fields {
		values[k] = v
	}

	msgID, err := conn.Cmd().XAdd(ctx, &redis.XAddArgs{
		Stream: req.Key,
		ID:     req.ID,
		Values: values,
	}).Result()
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "XADD", req.Key, "id="+msgID)
	return map[string]interface{}{"id": msgID}, nil
}

func handleXDelMessages(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string   `json:"conn_id"`
		Key    string   `json:"key"`
		IDs    []string `json:"ids"`
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

	deleted, err := conn.Cmd().XDel(ctx, req.Key, req.IDs...).Result()
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "XDEL", req.Key, fmt.Sprintf("%d messages", deleted))
	return map[string]interface{}{"deleted": deleted}, nil
}

func handleXTrimStream(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		MaxLen int64  `json:"max_len"`
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

	trimmed, err := conn.Cmd().XTrimMaxLen(ctx, req.Key, req.MaxLen).Result()
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "XTRIM", req.Key, fmt.Sprintf("maxlen=%d trimmed=%d", req.MaxLen, trimmed))
	return map[string]interface{}{"trimmed": trimmed}, nil
}

func handleXInfoStream(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
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

	info, err := conn.Cmd().XInfoStream(ctx, req.Key).Result()
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "XINFO", req.Key, fmt.Sprintf("length=%d groups=%d", info.Length, info.Groups))
	return map[string]interface{}{
		"length":            info.Length,
		"radix_tree_keys":   info.RadixTreeKeys,
		"radix_tree_nodes":  info.RadixTreeNodes,
		"groups":            info.Groups,
		"last_generated_id": info.LastGeneratedID,
		"first_entry_id":    fmt.Sprintf("%v", info.FirstEntry),
		"last_entry_id":     fmt.Sprintf("%v", info.LastEntry),
	}, nil
}

func handleXInfoGroups(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
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

	// Use raw Do() to avoid go-redis v8 parse error with Redis 7.x
	// (Redis 7.x returns 12 elements per group, go-redis v8 expects 8)
	var rawCmd *redis.Cmd
	if conn.IsCluster {
		rawCmd = conn.ClusterClient.Do(ctx, "XINFO", "GROUPS", req.Key)
	} else {
		rawCmd = conn.Client.Do(ctx, "XINFO", "GROUPS", req.Key)
	}
	raw, err := rawCmd.Result()
	if err != nil {
		return nil, err
	}
	groupList, ok := raw.([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}
	result := make([]map[string]interface{}, 0, len(groupList))
	for _, g := range groupList {
		fields, ok := g.([]interface{})
		if !ok {
			continue
		}
		m := make(map[string]interface{})
		for j := 0; j+1 < len(fields); j += 2 {
			key, _ := fields[j].(string)
			m[key] = fields[j+1]
		}
		entry := map[string]interface{}{
			"name":              m["name"],
			"consumers":         m["consumers"],
			"pending":           m["pending"],
			"last_delivered_id": m["last-delivered-id"],
		}
		if v, exists := m["entries-read"]; exists {
			entry["entries_read"] = v
		}
		if v, exists := m["lag"]; exists {
			entry["lag"] = v
		}
		result = append(result, entry)
	}
	AddOpLog(req.ConnID, "XINFO_GROUPS", req.Key, fmt.Sprintf("%d groups", len(result)))
	return result, nil
}

func handleXGroupCreate(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Group  string `json:"group"`
		Start  string `json:"start"`
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

	if req.Start == "" {
		req.Start = "$"
	}
	if err := conn.Cmd().XGroupCreate(ctx, req.Key, req.Group, req.Start).Err(); err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "XGROUP_CREATE", req.Key, "group="+req.Group)
	return nil, nil
}

func handleXGroupDestroy(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Group  string `json:"group"`
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

	if err := conn.Cmd().XGroupDestroy(ctx, req.Key, req.Group).Err(); err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "XGROUP_DESTROY", req.Key, "group="+req.Group)
	return nil, nil
}

// ========== HyperLogLog ==========

func handlePFCount(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
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

	count, err := conn.Cmd().PFCount(ctx, req.Key).Result()
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "PFCOUNT", req.Key, fmt.Sprintf("count=%d", count))
	return map[string]interface{}{"count": count}, nil
}

func handlePFAdd(params json.RawMessage) (any, error) {
	var req struct {
		ConnID   string   `json:"conn_id"`
		Key      string   `json:"key"`
		Elements []string `json:"elements"`
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

	ivals := make([]interface{}, len(req.Elements))
	for i, e := range req.Elements {
		ivals[i] = e
	}
	changed, err := conn.Cmd().PFAdd(ctx, req.Key, ivals...).Result()
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "PFADD", req.Key, fmt.Sprintf("%d elements", len(req.Elements)))
	return map[string]interface{}{"changed": changed}, nil
}

func handlePFMerge(params json.RawMessage) (any, error) {
	var req struct {
		ConnID     string   `json:"conn_id"`
		Key        string   `json:"key"`
		SourceKeys []string `json:"source_keys"`
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

	if len(req.SourceKeys) == 0 {
		return nil, fmt.Errorf("source_keys is empty")
	}
	if err := conn.Cmd().PFMerge(ctx, req.Key, req.SourceKeys...).Err(); err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "PFMERGE", req.Key, fmt.Sprintf("from %d keys", len(req.SourceKeys)))
	return nil, nil
}

// ========== Geospatial ==========

func handleGeoMembers(params json.RawMessage) (any, error) {
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

	if req.Stop == 0 {
		req.Stop = req.Start + 99
	}

	members, err := conn.Cmd().ZRange(ctx, req.Key, req.Start, req.Stop).Result()
	if err != nil {
		return nil, err
	}
	total, _ := conn.Cmd().ZCard(ctx, req.Key).Result()

	if len(members) == 0 {
		return map[string]interface{}{
			"members": []interface{}{},
			"total":   total,
		}, nil
	}

	positions, err := conn.Cmd().GeoPos(ctx, req.Key, members...).Result()
	if err != nil {
		return nil, err
	}

	type GeoItem struct {
		Member    string  `json:"member"`
		Longitude float64 `json:"longitude"`
		Latitude  float64 `json:"latitude"`
	}
	items := make([]GeoItem, 0, len(members))
	for i, m := range members {
		item := GeoItem{Member: m}
		if i < len(positions) && positions[i] != nil {
			item.Longitude = positions[i].Longitude
			item.Latitude = positions[i].Latitude
		}
		items = append(items, item)
	}

	AddOpLog(req.ConnID, "GEO_MEMBERS", req.Key, fmt.Sprintf("start=%d count=%d", req.Start, len(items)))
	return map[string]interface{}{
		"members": items,
		"total":   total,
	}, nil
}

func handleGeoAdd(params json.RawMessage) (any, error) {
	var req struct {
		ConnID  string `json:"conn_id"`
		Key     string `json:"key"`
		Members []struct {
			Name      string  `json:"name"`
			Longitude float64 `json:"longitude"`
			Latitude  float64 `json:"latitude"`
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

	locations := make([]*redis.GeoLocation, len(req.Members))
	for i, m := range req.Members {
		locations[i] = &redis.GeoLocation{
			Name:      m.Name,
			Longitude: m.Longitude,
			Latitude:  m.Latitude,
		}
	}

	added, err := conn.Cmd().GeoAdd(ctx, req.Key, locations...).Result()
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "GEOADD", req.Key, fmt.Sprintf("%d locations", added))
	return map[string]interface{}{"added": added}, nil
}

func handleGeoDist(params json.RawMessage) (any, error) {
	var req struct {
		ConnID  string `json:"conn_id"`
		Key     string `json:"key"`
		Member1 string `json:"member1"`
		Member2 string `json:"member2"`
		Unit    string `json:"unit"`
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

	if req.Unit == "" {
		req.Unit = "m"
	}
	dist, err := conn.Cmd().GeoDist(ctx, req.Key, req.Member1, req.Member2, req.Unit).Result()
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "GEODIST", req.Key, fmt.Sprintf("%s↔%s %.2f%s", req.Member1, req.Member2, dist, req.Unit))
	return map[string]interface{}{"distance": dist, "unit": req.Unit}, nil
}

func handleGeoSearch(params json.RawMessage) (any, error) {
	var req struct {
		ConnID    string  `json:"conn_id"`
		Key       string  `json:"key"`
		Longitude float64 `json:"longitude"`
		Latitude  float64 `json:"latitude"`
		Radius    float64 `json:"radius"`
		Unit      string  `json:"unit"`
		Count     int64   `json:"count"`
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

	if req.Unit == "" {
		req.Unit = "km"
	}
	if req.Count <= 0 {
		req.Count = 100
	}

	results, err := conn.Cmd().GeoRadius(ctx, req.Key, req.Longitude, req.Latitude, &redis.GeoRadiusQuery{
		Radius:    req.Radius,
		Unit:      req.Unit,
		WithCoord: true,
		WithDist:  true,
		Count:     int(req.Count),
		Sort:      "ASC",
	}).Result()
	if err != nil {
		return nil, err
	}

	type GeoResult struct {
		Name      string  `json:"name"`
		Dist      float64 `json:"dist"`
		Longitude float64 `json:"longitude"`
		Latitude  float64 `json:"latitude"`
	}
	items := make([]GeoResult, len(results))
	for i, r := range results {
		items[i] = GeoResult{
			Name: r.Name,
			Dist: r.Dist,
		}
		if r.Longitude != 0 || r.Latitude != 0 {
			items[i].Longitude = r.Longitude
			items[i].Latitude = r.Latitude
		}
	}

	AddOpLog(req.ConnID, "GEOSEARCH", req.Key, fmt.Sprintf("radius=%.1f%s results=%d", req.Radius, req.Unit, len(items)))
	return map[string]interface{}{"results": items}, nil
}

// ========== Bitmap ==========

func handleBitmapGetRange(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Start  int64  `json:"start"`
		Count  int64  `json:"count"`
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

	if req.Count <= 0 || req.Count > 1024 {
		req.Count = 256
	}

	pipe := conn.Cmd().Pipeline()
	cmds := make([]*redis.IntCmd, req.Count)
	for i := int64(0); i < req.Count; i++ {
		cmds[i] = pipe.GetBit(ctx, req.Key, req.Start+i)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}
	bits := make([]int, req.Count)
	for i, cmd := range cmds {
		bits[i] = int(cmd.Val())
	}

	AddOpLog(req.ConnID, "BITMAP_GET", req.Key, fmt.Sprintf("start=%d count=%d", req.Start, req.Count))
	return map[string]interface{}{"bits": bits, "start": req.Start}, nil
}

func handleBitmapSetBit(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Offset int64  `json:"offset"`
		Value  int    `json:"value"`
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

	old, err := conn.Cmd().SetBit(ctx, req.Key, req.Offset, req.Value).Result()
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "SETBIT", req.Key, fmt.Sprintf("offset=%d value=%d", req.Offset, req.Value))
	return map[string]interface{}{"old_value": old}, nil
}

func handleBitmapCount(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
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

	count, err := conn.Cmd().BitCount(ctx, req.Key, nil).Result()
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "BITCOUNT", req.Key, fmt.Sprintf("count=%d", count))
	return map[string]interface{}{"count": count}, nil
}

// ========== Bitfield ==========

func handleBitfieldGet(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Fields []struct {
			Type   string `json:"type"`
			Offset string `json:"offset"`
		} `json:"fields"`
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

	args := []interface{}{"BITFIELD", req.Key}
	for _, f := range req.Fields {
		args = append(args, "GET", f.Type, f.Offset)
	}
	result, err := conn.Do(ctx, args...).Int64Slice()
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "BITFIELD_GET", req.Key, fmt.Sprintf("%d fields", len(req.Fields)))
	return map[string]interface{}{"values": result}, nil
}

func handleBitfieldSet(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Type   string `json:"type"`
		Offset string `json:"offset"`
		Value  int64  `json:"value"`
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

	result, err := conn.Do(ctx, "BITFIELD", req.Key, "SET", req.Type, req.Offset, req.Value).Int64Slice()
	if err != nil {
		return nil, err
	}
	var old int64
	if len(result) > 0 {
		old = result[0]
	}
	AddOpLog(req.ConnID, "BITFIELD_SET", req.Key, fmt.Sprintf("SET %s %s %d", req.Type, req.Offset, req.Value))
	return map[string]interface{}{"old_value": old}, nil
}

func handleBitfieldIncrBy(params json.RawMessage) (any, error) {
	var req struct {
		ConnID    string `json:"conn_id"`
		Key       string `json:"key"`
		Type      string `json:"type"`
		Offset    string `json:"offset"`
		Increment int64  `json:"increment"`
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

	result, err := conn.Do(ctx, "BITFIELD", req.Key, "INCRBY", req.Type, req.Offset, req.Increment).Int64Slice()
	if err != nil {
		return nil, err
	}
	var newVal int64
	if len(result) > 0 {
		newVal = result[0]
	}
	AddOpLog(req.ConnID, "BITFIELD_INCRBY", req.Key, fmt.Sprintf("INCRBY %s %s %d", req.Type, req.Offset, req.Increment))
	return map[string]interface{}{"new_value": newVal}, nil
}

// ========== RedisJSON ==========

func handleJSONGet(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Path   string `json:"path"`
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

	if req.Path == "" {
		req.Path = "."
	}
	result, err := conn.Do(ctx, "JSON.GET", req.Key, req.Path).Text()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("路径不存在")
		}
		return nil, err
	}
	AddOpLog(req.ConnID, "JSON.GET", req.Key, "path="+req.Path)
	return map[string]interface{}{"value": result}, nil
}

func handleJSONSet(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Path   string `json:"path"`
		Value  string `json:"value"` // JSON string
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

	if req.Path == "" {
		req.Path = "."
	}
	if err := conn.Do(ctx, "JSON.SET", req.Key, req.Path, req.Value).Err(); err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "JSON.SET", req.Key, "path="+req.Path)
	return nil, nil
}

func handleJSONDel(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Path   string `json:"path"`
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

	if req.Path == "" || req.Path == "." {
		return nil, fmt.Errorf("不能删除根路径，请使用 DEL 命令")
	}
	deleted, err := conn.Do(ctx, "JSON.DEL", req.Key, req.Path).Int()
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "JSON.DEL", req.Key, "path="+req.Path)
	return map[string]interface{}{"deleted": deleted}, nil
}

func handleJSONType(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Path   string `json:"path"`
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

	if req.Path == "" {
		req.Path = "."
	}
	result, err := conn.Do(ctx, "JSON.TYPE", req.Key, req.Path).Text()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("路径不存在")
		}
		return nil, err
	}
	return map[string]interface{}{"type": result}, nil
}

func handleJSONArrAppend(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string   `json:"conn_id"`
		Key    string   `json:"key"`
		Path   string   `json:"path"`
		Values []string `json:"values"` // JSON strings to append
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

	if req.Path == "" {
		req.Path = "."
	}
	args := []interface{}{"JSON.ARRAPPEND", req.Key, req.Path}
	for _, v := range req.Values {
		args = append(args, v)
	}
	newLen, err := conn.Do(ctx, args...).Int()
	if err != nil {
		return nil, err
	}
	AddOpLog(req.ConnID, "JSON.ARRAPPEND", req.Key, fmt.Sprintf("path=%s count=%d", req.Path, len(req.Values)))
	return map[string]interface{}{"new_length": newLen}, nil
}
