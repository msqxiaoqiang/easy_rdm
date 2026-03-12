package tests

import (
	"testing"

	"easy_rdm/testsetup"
)

// ========== Stream 参数校验 ==========

func TestAPI_XRangeMessages_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/xrange_messages", map[string]string{"conn_id": "no-conn", "key": "k"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_XAddMessage_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/xadd_message", map[string]interface{}{"conn_id": "no-conn", "key": "k", "fields": map[string]string{"f": "v"}})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_XDelMessages_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/xdel_messages", map[string]interface{}{"conn_id": "no-conn", "key": "k", "ids": []string{"0-1"}})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_XInfoStream_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/xinfo_stream", map[string]string{"conn_id": "no-conn", "key": "k"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_XGroupCreate_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/xgroup_create", map[string]string{"conn_id": "no-conn", "key": "k", "group": "g1"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

// ========== HyperLogLog 参数校验 ==========

func TestAPI_PFCount_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/pfcount", map[string]string{"conn_id": "no-conn", "key": "k"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_PFAdd_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/pfadd", map[string]interface{}{"conn_id": "no-conn", "key": "k", "elements": []string{"a"}})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_PFMerge_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/pfmerge", map[string]interface{}{"conn_id": "no-conn", "key": "k", "source_keys": []string{"s1"}})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

// ========== Geo 参数校验 ==========

func TestAPI_GeoMembers_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/geo_members", map[string]string{"conn_id": "no-conn", "key": "k"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_GeoAdd_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/geo_add", map[string]interface{}{
		"conn_id": "no-conn", "key": "k",
		"members": []map[string]interface{}{{"longitude": 116.4, "latitude": 39.9, "member": "bj"}},
	})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_GeoDist_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/geo_dist", map[string]string{"conn_id": "no-conn", "key": "k", "member1": "a", "member2": "b"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_GeoSearch_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/geo_search", map[string]interface{}{"conn_id": "no-conn", "key": "k", "longitude": 116.4, "latitude": 39.9, "radius": 100.0, "unit": "km"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

// ========== Bitmap 参数校验 ==========

func TestAPI_BitmapGetRange_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/bitmap_get_range", map[string]interface{}{"conn_id": "no-conn", "key": "k", "start": 0, "end": 7})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_BitmapSetBit_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/bitmap_set_bit", map[string]interface{}{"conn_id": "no-conn", "key": "k", "offset": 0, "value": 1})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_BitmapCount_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/bitmap_count", map[string]string{"conn_id": "no-conn", "key": "k"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

// ========== Bitfield 参数校验 ==========

func TestAPI_BitfieldGet_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/bitfield_get", map[string]interface{}{"conn_id": "no-conn", "key": "k", "type": "u8", "offset": 0})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_BitfieldSet_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/bitfield_set", map[string]interface{}{"conn_id": "no-conn", "key": "k", "type": "u8", "offset": 0, "value": 42})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400 or 404, got %d", resp.Code)
	}
}

func TestAPI_BitfieldIncrBy_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/bitfield_incrby", map[string]interface{}{"conn_id": "no-conn", "key": "k", "type": "u8", "offset": 0, "increment": 1})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400 or 404, got %d", resp.Code)
	}
}

// ========== JSON 参数校验 ==========

func TestAPI_JSONGet_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/json_get", map[string]string{"conn_id": "no-conn", "key": "k"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_JSONSet_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/json_set", map[string]string{"conn_id": "no-conn", "key": "k", "path": "$", "value": "{}"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestAPI_JSONDel_NoConnection(t *testing.T) {
	router := testsetup.SetupTestRouter(t)
	w := testsetup.PostJSON(router, "/api/json_del", map[string]string{"conn_id": "no-conn", "key": "k", "path": "$.foo"})
	resp := testsetup.ParseResponse(t, w)
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}
