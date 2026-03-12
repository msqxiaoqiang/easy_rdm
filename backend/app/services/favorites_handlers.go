package services

import (
	"encoding/json"
	"fmt"

)

// RegisterFavoritesHandlers 注册收藏夹相关 RPC 方法
func RegisterFavoritesHandlers(register func(string, RPCHandlerFunc)) {
	register("get_favorites", handleGetFavorites)
	register("toggle_favorite", handleToggleFavorite)
}

// favorites.json 结构: { "connId:db": ["key1", "key2", ...] }
type FavoritesData map[string][]string

func buildFavKey(connID string, db int) string {
	b, _ := json.Marshal(db)
	return connID + ":" + string(b)
}

func loadFavorites() FavoritesData {
	var data FavoritesData
	if err := ReadJSON("favorites.json", &data); err != nil || data == nil {
		return FavoritesData{}
	}
	return data
}

func saveFavorites(data FavoritesData) error {
	return WriteJSON("favorites.json", data)
}

func handleGetFavorites(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		DB     int    `json:"db"`
	}
	if err := json.Unmarshal(params, &req); err != nil || req.ConnID == "" {
		return nil, fmt.Errorf("参数错误")
	}

	data := loadFavorites()
	key := buildFavKey(req.ConnID, req.DB)
	favs := data[key]
	if favs == nil {
		favs = []string{}
	}
	return favs, nil
}

func handleToggleFavorite(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		DB     int    `json:"db"`
		Key    string `json:"key"`
	}
	if err := json.Unmarshal(params, &req); err != nil || req.ConnID == "" || req.Key == "" {
		return nil, fmt.Errorf("参数错误")
	}

	data := loadFavorites()
	fk := buildFavKey(req.ConnID, req.DB)
	favs := data[fk]

	// 查找是否已收藏
	idx := -1
	for i, k := range favs {
		if k == req.Key {
			idx = i
			break
		}
	}

	added := false
	if idx >= 0 {
		// 取消收藏
		favs = append(favs[:idx], favs[idx+1:]...)
	} else {
		// 添加收藏
		favs = append(favs, req.Key)
		added = true
	}
	data[fk] = favs

	if err := saveFavorites(data); err != nil {
		return nil, fmt.Errorf("保存失败")
	}

	return map[string]interface{}{
		"added":     added,
		"favorites": favs,
	}, nil
}
