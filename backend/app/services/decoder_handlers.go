package services

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

)

// RegisterDecoderHandlers 注册解码器相关的 RPC 方法
func RegisterDecoderHandlers(register func(string, RPCHandlerFunc)) {
	register("get_decoders", handleGetDecoders)
	register("save_decoder", handleSaveDecoder)
	register("delete_decoder", handleDeleteDecoder)
	register("decode_value", handleDecodeValue)
}

type DecoderConfig struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`    // "builtin" | "command"
	Command string `json:"command"` // for command type: shell command, value passed via stdin
}

// 内置解码器列表
var builtinDecoders = []DecoderConfig{
	{ID: "__base64", Name: "Base64", Type: "builtin"},
	{ID: "__gzip", Name: "Gzip", Type: "builtin"},
	{ID: "__hex", Name: "Hex Dump", Type: "builtin"},
}

func handleGetDecoders(_ json.RawMessage) (any, error) {
	var custom []DecoderConfig
	ReadJSON("decoders.json", &custom)

	// Combine builtin + custom
	all := make([]DecoderConfig, 0, len(builtinDecoders)+len(custom))
	all = append(all, builtinDecoders...)
	all = append(all, custom...)

	return all, nil
}

func handleSaveDecoder(params json.RawMessage) (any, error) {
	var dec DecoderConfig
	if err := json.Unmarshal(params, &dec); err != nil || dec.Name == "" {
		return nil, fmt.Errorf("参数错误: name 必填")
	}
	if dec.Type == "" {
		dec.Type = "command"
	}
	if dec.Type == "command" && dec.Command == "" {
		return nil, fmt.Errorf("参数错误: command 必填")
	}

	var decoders []DecoderConfig
	ReadJSON("decoders.json", &decoders)

	// Generate ID if new
	if dec.ID == "" {
		dec.ID = fmt.Sprintf("custom_%d", time.Now().UnixMilli())
	}

	// Update or append
	found := false
	for i, d := range decoders {
		if d.ID == dec.ID {
			decoders[i] = dec
			found = true
			break
		}
	}
	if !found {
		decoders = append(decoders, dec)
	}

	if err := WriteJSON("decoders.json", decoders); err != nil {
		return nil, err
	}
	return dec, nil
}

func handleDeleteDecoder(params json.RawMessage) (any, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(params, &req); err != nil || req.ID == "" {
		return nil, fmt.Errorf("参数错误: id 必填")
	}
	// Prevent deleting builtins
	if strings.HasPrefix(req.ID, "__") {
		return nil, fmt.Errorf("不能删除内置解码器")
	}

	var decoders []DecoderConfig
	ReadJSON("decoders.json", &decoders)

	filtered := make([]DecoderConfig, 0, len(decoders))
	for _, d := range decoders {
		if d.ID != req.ID {
			filtered = append(filtered, d)
		}
	}

	if err := WriteJSON("decoders.json", filtered); err != nil {
		return nil, err
	}
	return nil, nil
}

func handleDecodeValue(params json.RawMessage) (any, error) {
	var req struct {
		DecoderID string `json:"decoder_id"`
		Value     string `json:"value"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	// Find decoder
	var decoder *DecoderConfig

	// Check builtins
	for _, d := range builtinDecoders {
		if d.ID == req.DecoderID {
			dc := d
			decoder = &dc
			break
		}
	}

	// Check custom
	if decoder == nil {
		var custom []DecoderConfig
		ReadJSON("decoders.json", &custom)
		for _, d := range custom {
			if d.ID == req.DecoderID {
				dc := d
				decoder = &dc
				break
			}
		}
	}

	if decoder == nil {
		return nil, fmt.Errorf("解码器不存在")
	}

	result, err := applyDecoder(decoder, req.Value)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func applyDecoder(dec *DecoderConfig, value string) (string, error) {
	if dec.Type == "builtin" {
		return applyBuiltinDecoder(dec.ID, value)
	}
	return applyCommandDecoder(dec.Command, value)
}

func applyBuiltinDecoder(id string, value string) (string, error) {
	switch id {
	case "__base64":
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
		if err != nil {
			// Try URL-safe base64
			decoded, err = base64.URLEncoding.DecodeString(strings.TrimSpace(value))
			if err != nil {
				// Try raw (no padding)
				decoded, err = base64.RawStdEncoding.DecodeString(strings.TrimSpace(value))
				if err != nil {
					return "", fmt.Errorf("Base64 解码失败: %w", err)
				}
			}
		}
		return string(decoded), nil

	case "__gzip":
		// Value might be base64-encoded gzip data
		data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
		if err != nil {
			// Try raw bytes
			data = []byte(value)
		}
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return "", fmt.Errorf("Gzip 解压失败: %w", err)
		}
		defer reader.Close()
		result, err := io.ReadAll(reader)
		if err != nil {
			return "", fmt.Errorf("Gzip 读取失败: %w", err)
		}
		return string(result), nil

	case "__hex":
		// Convert string to hex dump format
		data := []byte(value)
		var buf strings.Builder
		for i := 0; i < len(data); i += 16 {
			// Offset
			fmt.Fprintf(&buf, "%08x  ", i)
			// Hex bytes
			end := i + 16
			if end > len(data) {
				end = len(data)
			}
			for j := i; j < end; j++ {
				fmt.Fprintf(&buf, "%02x ", data[j])
				if j == i+7 {
					buf.WriteByte(' ')
				}
			}
			// Padding
			for j := end; j < i+16; j++ {
				buf.WriteString("   ")
				if j == i+7 {
					buf.WriteByte(' ')
				}
			}
			// ASCII
			buf.WriteString(" |")
			for j := i; j < end; j++ {
				if data[j] >= 32 && data[j] < 127 {
					buf.WriteByte(data[j])
				} else {
					buf.WriteByte('.')
				}
			}
			buf.WriteString("|\n")
		}
		return buf.String(), nil

	default:
		return "", fmt.Errorf("未知的内置解码器: %s", id)
	}
}

func applyCommandDecoder(command string, value string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Stdin = strings.NewReader(value)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("命令执行超时（30秒）")
		}
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("命令执行失败: %s", errMsg)
	}

	return stdout.String(), nil
}
