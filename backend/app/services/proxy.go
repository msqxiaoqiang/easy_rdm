package services

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

// ProxyConfig 代理配置
type ProxyConfig struct {
	Type     string `json:"proxy_type"`     // http | https | socks5 | socks5h
	Host     string `json:"proxy_host"`
	Port     int    `json:"proxy_port"`
	Username string `json:"proxy_username"`
	Password string `json:"proxy_password"`
}

// ProxyDialer 根据代理配置创建拨号函数（签名匹配 redis.Options.Dialer）
func ProxyDialer(cfg *ProxyConfig, timeout time.Duration) (func(ctx context.Context, network, addr string) (net.Conn, error), error) {
	proxyAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	switch cfg.Type {
	case "socks5", "socks5h":
		var auth *proxy.Auth
		if cfg.Username != "" {
			auth = &proxy.Auth{User: cfg.Username, Password: cfg.Password}
		}
		dialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, &net.Dialer{Timeout: timeout})
		if err != nil {
			return nil, fmt.Errorf("创建 SOCKS5 代理失败: %w", err)
		}
		return func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}, nil

	case "http", "https":
		return func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := net.DialTimeout("tcp", proxyAddr, timeout)
			if err != nil {
				return nil, fmt.Errorf("连接 HTTP 代理失败: %w", err)
			}
			// HTTP CONNECT 隧道
			connectReq := &http.Request{
				Method: "CONNECT",
				URL:    &url.URL{Opaque: addr},
				Host:   addr,
				Header: make(http.Header),
			}
			if cfg.Username != "" {
				cred := base64.StdEncoding.EncodeToString([]byte(cfg.Username + ":" + cfg.Password))
				connectReq.Header.Set("Proxy-Authorization", "Basic "+cred)
			}
			if err := connectReq.Write(conn); err != nil {
				conn.Close()
				return nil, fmt.Errorf("发送 CONNECT 请求失败: %w", err)
			}
			resp, err := http.ReadResponse(bufio.NewReader(conn), connectReq)
			if err != nil {
				conn.Close()
				return nil, fmt.Errorf("读取代理响应失败: %w", err)
			}
			if resp.StatusCode != 200 {
				conn.Close()
				return nil, fmt.Errorf("代理 CONNECT 失败: %s", resp.Status)
			}
			return conn, nil
		}, nil

	default:
		return nil, fmt.Errorf("不支持的代理类型: %s", cfg.Type)
	}
}
