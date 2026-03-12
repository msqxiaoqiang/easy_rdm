package services

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHTunnel SSH 隧道
type SSHTunnel struct {
	listener net.Listener
	client   *ssh.Client
	localAddr string
	done     chan struct{}
}

// SSHConfig SSH 连接配置
type SSHConfig struct {
	Host       string `json:"ssh_host"`
	Port       int    `json:"ssh_port"`
	Username   string `json:"ssh_username"`
	Password   string `json:"ssh_password"`
	PrivateKey string `json:"ssh_private_key"` // 私钥文件路径
	Passphrase string `json:"ssh_passphrase"`  // 私钥密码
}

var (
	tunnels   = make(map[string]*SSHTunnel) // connID -> tunnel
	tunnelMu  sync.Mutex
)

// CreateTunnel 创建 SSH 隧道，返回本地监听地址
func CreateTunnel(connID string, sshCfg *SSHConfig, remoteAddr string) (string, error) {
	tunnelMu.Lock()
	defer tunnelMu.Unlock()

	// 关闭已有隧道
	if t, ok := tunnels[connID]; ok {
		t.Close()
		delete(tunnels, connID)
	}

	// 构建 SSH 认证方式
	var authMethods []ssh.AuthMethod
	if sshCfg.PrivateKey != "" {
		keyData, err := os.ReadFile(sshCfg.PrivateKey)
		if err != nil {
			return "", fmt.Errorf("读取 SSH 私钥失败: %w", err)
		}
		var signer ssh.Signer
		if sshCfg.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(sshCfg.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(keyData)
		}
		if err != nil {
			return "", fmt.Errorf("解析 SSH 私钥失败: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}
	if sshCfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(sshCfg.Password))
	}
	if len(authMethods) == 0 {
		return "", fmt.Errorf("SSH 认证方式未配置（需要密码或私钥）")
	}

	sshAddr := fmt.Sprintf("%s:%d", sshCfg.Host, sshCfg.Port)
	config := &ssh.ClientConfig{
		User:            sshCfg.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", sshAddr, config)
	if err != nil {
		return "", fmt.Errorf("SSH 连接失败: %w", err)
	}

	// 本地随机端口监听
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		client.Close()
		return "", fmt.Errorf("创建本地监听失败: %w", err)
	}

	tunnel := &SSHTunnel{
		listener:  listener,
		client:    client,
		localAddr: listener.Addr().String(),
		done:      make(chan struct{}),
	}
	tunnels[connID] = tunnel

	// 后台转发
	go tunnel.forward(remoteAddr)

	return tunnel.localAddr, nil
}

// forward 接受本地连接并转发到远程
func (t *SSHTunnel) forward(remoteAddr string) {
	for {
		local, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.done:
				return
			default:
				continue
			}
		}
		go func() {
			remote, err := t.client.Dial("tcp", remoteAddr)
			if err != nil {
				local.Close()
				return
			}
			go copyAndClose(local, remote)
			go copyAndClose(remote, local)
		}()
	}
}

func copyAndClose(dst, src net.Conn) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, wErr := dst.Write(buf[:n]); wErr != nil {
				break
			}
		}
		if err != nil {
			break
		}
	}
	dst.Close()
}

// Close 关闭隧道
func (t *SSHTunnel) Close() {
	close(t.done)
	t.listener.Close()
	t.client.Close()
}

// CloseTunnel 关闭指定连接的 SSH 隧道
func CloseTunnel(connID string) {
	tunnelMu.Lock()
	defer tunnelMu.Unlock()
	if t, ok := tunnels[connID]; ok {
		t.Close()
		delete(tunnels, connID)
	}
}
