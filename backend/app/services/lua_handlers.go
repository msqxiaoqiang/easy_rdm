package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

)

// RegisterLuaHandlers 注册 Lua 脚本相关的 RPC 方法
func RegisterLuaHandlers(register func(string, RPCHandlerFunc)) {
	register("lua_eval", handleLuaEval)
	register("lua_scripts_list", handleLuaScriptsList)
	register("lua_script_save", handleLuaScriptSave)
	register("lua_script_delete", handleLuaScriptDelete)
}

func handleLuaEval(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string   `json:"conn_id"`
		Script string   `json:"script"`
		Keys   []string `json:"keys"`
		Args   []string `json:"args"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	if req.Script == "" {
		return nil, fmt.Errorf("脚本内容为空")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 30*time.Second)
	defer cancel()

	// 构建 EVAL 参数
	args := make([]interface{}, len(req.Args))
	for i, a := range req.Args {
		args[i] = a
	}

	result, err := conn.Cmd().Eval(ctx, req.Script, req.Keys, args...).Result()
	if err != nil {
		AddOpLog(req.ConnID, "EVAL", "", fmt.Sprintf("error: %s", err.Error()))
		return map[string]interface{}{
			"error":  err.Error(),
			"result": nil,
		}, nil
	}

	AddOpLog(req.ConnID, "EVAL", "", fmt.Sprintf("keys=%d args=%d", len(req.Keys), len(req.Args)))
	return map[string]interface{}{
		"result": result,
		"error":  nil,
	}, nil
}

// Lua 脚本持久化

type LuaScript struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Script  string `json:"script"`
	Keys    string `json:"keys"`
	Args    string `json:"args"`
	Updated int64  `json:"updated"`
}

func handleLuaScriptsList(_ json.RawMessage) (any, error) {
	var scripts []LuaScript
	if err := ReadJSON("lua_scripts.json", &scripts); err != nil {
		scripts = []LuaScript{}
	}
	return scripts, nil
}

func handleLuaScriptSave(params json.RawMessage) (any, error) {
	var script LuaScript
	if err := json.Unmarshal(params, &script); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	if script.ID == "" || script.Name == "" {
		return nil, fmt.Errorf("ID 和名称必填")
	}
	script.Updated = time.Now().UnixMilli()

	var scripts []LuaScript
	ReadJSON("lua_scripts.json", &scripts)

	found := false
	for i, s := range scripts {
		if s.ID == script.ID {
			scripts[i] = script
			found = true
			break
		}
	}
	if !found {
		scripts = append(scripts, script)
	}

	if err := WriteJSON("lua_scripts.json", scripts); err != nil {
		return nil, err
	}
	return nil, nil
}

func handleLuaScriptDelete(params json.RawMessage) (any, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	var scripts []LuaScript
	ReadJSON("lua_scripts.json", &scripts)

	filtered := make([]LuaScript, 0, len(scripts))
	for _, s := range scripts {
		if s.ID != req.ID {
			filtered = append(filtered, s)
		}
	}

	if err := WriteJSON("lua_scripts.json", filtered); err != nil {
		return nil, err
	}
	return nil, nil
}
