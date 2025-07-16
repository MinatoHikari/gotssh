package ssh

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/net/proxy"
	"golang.org/x/term"

	"gotssh/internal/config"
)

// Client SSH客户端
type Client struct {
	config        *config.ServerConfig
	credential    *config.CredentialConfig
	configManager *config.Manager
	conn          *ssh.Client
}

// NewClient 创建新的SSH客户端
func NewClient(cfg *config.ServerConfig, configManager *config.Manager) *Client {
	client := &Client{
		config:        cfg,
		configManager: configManager,
	}
	
	// 如果服务器配置引用了凭证，尝试加载凭证
	if cfg.CredentialID != "" && configManager != nil {
		if cred, err := configManager.GetCredential(cfg.CredentialID); err == nil {
			client.credential = cred
		}
	}
	
	return client
}

// Connect 连接到SSH服务器
func (c *Client) Connect() error {
	sshConfig, err := c.buildSSHConfig()
	if err != nil {
		return fmt.Errorf("构建SSH配置失败: %w", err)
	}

	var conn net.Conn
	address := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	// 如果配置了代理，则使用代理连接
	if c.config.Proxy != nil {
		conn, err = c.connectViaProxy(address)
		if err != nil {
			return fmt.Errorf("代理连接失败: %w", err)
		}
	} else {
		conn, err = net.DialTimeout("tcp", address, time.Duration(30)*time.Second)
		if err != nil {
			return fmt.Errorf("连接失败: %w", err)
		}
	}

	// 创建SSH连接
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, address, sshConfig)
	if err != nil {
		conn.Close()
		return fmt.Errorf("SSH握手失败: %w", err)
	}

	c.conn = ssh.NewClient(sshConn, chans, reqs)
	return nil
}

// connectViaProxy 通过代理连接
func (c *Client) connectViaProxy(address string) (net.Conn, error) {
	proxyURL := &url.URL{
		Scheme: c.config.Proxy.Type,
		Host:   fmt.Sprintf("%s:%d", c.config.Proxy.Host, c.config.Proxy.Port),
	}

	if c.config.Proxy.Username != "" {
		proxyURL.User = url.UserPassword(c.config.Proxy.Username, c.config.Proxy.Password)
	}

	dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("创建代理拨号器失败: %w", err)
	}

	conn, err := dialer.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("代理拨号失败: %w", err)
	}

	return conn, nil
}

// buildSSHConfig 构建SSH配置
func (c *Client) buildSSHConfig() (*ssh.ClientConfig, error) {
	var authMethods []ssh.AuthMethod
	var err error
	var username string

	// 确定用户名：优先使用凭证中的用户名，其次是服务器配置中的用户名
	if c.credential != nil && c.credential.Username != "" {
		username = c.credential.Username
	} else {
		username = c.config.User
	}

	switch c.config.AuthType {
	case config.AuthTypePassword:
		password := c.config.Password
		// 如果有凭证且为密码类型，使用凭证中的密码
		if c.credential != nil && c.credential.Type == config.CredentialTypePassword && c.credential.Password != "" {
			password = c.credential.Password
		}
		if password == "" {
			return nil, fmt.Errorf("密码认证需要提供密码")
		}
		authMethods = append(authMethods, ssh.Password(password))

	case config.AuthTypeKey:
		keyAuth, err := c.getKeyAuth()
		if err != nil {
			return nil, fmt.Errorf("获取密钥认证失败: %w", err)
		}
		authMethods = append(authMethods, keyAuth)

	case config.AuthTypeCredential:
		// 使用引用的凭证
		if c.credential != nil {
			if c.credential.Type == config.CredentialTypePassword {
				if c.credential.Password == "" {
					return nil, fmt.Errorf("凭证中的密码为空")
				}
				authMethods = append(authMethods, ssh.Password(c.credential.Password))
			} else if c.credential.Type == config.CredentialTypeKey {
				keyAuth, err := c.getCredentialKeyAuth()
				if err != nil {
					return nil, fmt.Errorf("获取凭证密钥认证失败: %w", err)
				}
				authMethods = append(authMethods, keyAuth)
			}
		} else {
			// 后备方案：尝试使用SSH代理和默认密钥
			if agentAuth, err := c.getAgentAuth(); err == nil {
				authMethods = append(authMethods, agentAuth)
			}
			
			if keyAuth, err := c.getDefaultKeyAuth(); err == nil {
				authMethods = append(authMethods, keyAuth)
			}
		}

	case config.AuthTypeAsk:
		// 交互式询问认证方式
		authMethods, err = c.getInteractiveAuth()
		if err != nil {
			return nil, fmt.Errorf("交互式认证失败: %w", err)
		}
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("未找到有效的认证方式")
	}

	return &ssh.ClientConfig{
		User:            username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 在生产环境中应该使用适当的主机密钥验证
		Timeout:         30 * time.Second,
	}, nil
}

// getKeyAuth 获取密钥认证
func (c *Client) getKeyAuth() (ssh.AuthMethod, error) {
	keyPath := c.config.KeyPath
	keyPassphrase := c.config.KeyPassphrase
	
	// 如果有凭证且为密钥类型，优先使用凭证中的密钥
	if c.credential != nil && c.credential.Type == config.CredentialTypeKey {
		if c.credential.KeyPath != "" {
			keyPath = c.credential.KeyPath
		}
		if c.credential.KeyPassphrase != "" {
			keyPassphrase = c.credential.KeyPassphrase
		}
	}
	
	if keyPath == "" {
		return nil, fmt.Errorf("密钥认证需要提供密钥文件路径")
	}
	
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("读取密钥文件失败: %w", err)
	}

	var signer ssh.Signer
	if keyPassphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(keyPassphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(keyData)
	}

	if err != nil {
		return nil, fmt.Errorf("解析私钥失败: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}

// getCredentialKeyAuth 获取凭证密钥认证
func (c *Client) getCredentialKeyAuth() (ssh.AuthMethod, error) {
	if c.credential == nil || c.credential.Type != config.CredentialTypeKey {
		return nil, fmt.Errorf("无效的凭证类型")
	}

	var keyData []byte
	var err error

	// 优先使用密钥内容，其次使用密钥文件路径
	if c.credential.KeyContent != "" {
		keyData = []byte(c.credential.KeyContent)
	} else if c.credential.KeyPath != "" {
		keyData, err = os.ReadFile(c.credential.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("读取密钥文件失败: %w", err)
		}
	} else {
		return nil, fmt.Errorf("凭证中既没有密钥内容也没有密钥文件路径")
	}

	var signer ssh.Signer
	if c.credential.KeyPassphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(c.credential.KeyPassphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(keyData)
	}

	if err != nil {
		return nil, fmt.Errorf("解析私钥失败: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}

// getAgentAuth 获取SSH代理认证
func (c *Client) getAgentAuth() (ssh.AuthMethod, error) {
	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, fmt.Errorf("连接SSH代理失败: %w", err)
	}

	return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers), nil
}

// getDefaultKeyAuth 获取默认密钥认证
func (c *Client) getDefaultKeyAuth() (ssh.AuthMethod, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("获取用户主目录失败: %w", err)
	}

	keyPaths := []string{
		filepath.Join(homeDir, ".ssh", "id_rsa"),
		filepath.Join(homeDir, ".ssh", "id_ecdsa"),
		filepath.Join(homeDir, ".ssh", "id_ed25519"),
	}

	for _, keyPath := range keyPaths {
		if _, err := os.Stat(keyPath); err == nil {
			keyData, err := os.ReadFile(keyPath)
			if err != nil {
				continue
			}

			signer, err := ssh.ParsePrivateKey(keyData)
			if err != nil {
				continue
			}

			return ssh.PublicKeys(signer), nil
		}
	}

	return nil, fmt.Errorf("未找到默认密钥")
}

// getInteractiveAuth 获取交互式认证
func (c *Client) getInteractiveAuth() ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod

	fmt.Printf("连接到 %s@%s:%d\n", c.config.User, c.config.Host, c.config.Port)
	fmt.Println("请选择认证方式:")
	fmt.Println("1. 密码认证")
	fmt.Println("2. 密钥认证")
	fmt.Println("3. SSH代理认证")

	var choice int
	fmt.Print("请输入选择 (1-3): ")
	if _, err := fmt.Scanf("%d", &choice); err != nil {
		return nil, fmt.Errorf("读取选择失败: %w", err)
	}

	switch choice {
	case 1:
		fmt.Print("请输入密码: ")
		password, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return nil, fmt.Errorf("读取密码失败: %w", err)
		}
		fmt.Println()
		authMethods = append(authMethods, ssh.Password(string(password)))

	case 2:
		fmt.Print("请输入密钥文件路径: ")
		var keyPath string
		if _, err := fmt.Scanf("%s", &keyPath); err != nil {
			return nil, fmt.Errorf("读取密钥路径失败: %w", err)
		}

		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("读取密钥文件失败: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			// 尝试使用密码短语
			fmt.Print("请输入密钥密码短语: ")
			passphrase, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return nil, fmt.Errorf("读取密码短语失败: %w", err)
			}
			fmt.Println()

			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, passphrase)
			if err != nil {
				return nil, fmt.Errorf("解析私钥失败: %w", err)
			}
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))

	case 3:
		agentAuth, err := c.getAgentAuth()
		if err != nil {
			return nil, fmt.Errorf("SSH代理认证失败: %w", err)
		}
		authMethods = append(authMethods, agentAuth)

	default:
		return nil, fmt.Errorf("无效的选择")
	}

	return authMethods, nil
}

// NewSession 创建新的SSH会话
func (c *Client) NewSession() (*ssh.Session, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("SSH连接未建立")
	}

	session, err := c.conn.NewSession()
	if err != nil {
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}

	return session, nil
}

// ExecuteCommand 执行命令
func (c *Client) ExecuteCommand(command string) (string, error) {
	session, err := c.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", fmt.Errorf("执行命令失败: %w", err)
	}

	return string(output), nil
}

// Shell 创建交互式Shell
func (c *Client) Shell() error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// 执行启动脚本
	if c.config.StartupScript != "" {
		if _, err := c.ExecuteCommand(c.config.StartupScript); err != nil {
			fmt.Printf("警告: 启动脚本执行失败: %v\n", err)
		}
	}

	// 设置终端模式
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		state, err := term.MakeRaw(fd)
		if err != nil {
			return fmt.Errorf("设置终端模式失败: %w", err)
		}
		defer term.Restore(fd, state)

		// 获取终端大小
		width, height, err := term.GetSize(fd)
		if err != nil {
			width, height = 80, 24
		}

		// 请求伪终端
		if err := session.RequestPty("xterm-256color", height, width, ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}); err != nil {
			return fmt.Errorf("请求伪终端失败: %w", err)
		}
	}

	// 设置IO
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	// 启动Shell
	if err := session.Shell(); err != nil {
		return fmt.Errorf("启动Shell失败: %w", err)
	}

	// 等待会话结束
	if err := session.Wait(); err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			return fmt.Errorf("Shell退出，退出码: %d", exitErr.ExitStatus())
		}
		return fmt.Errorf("Shell会话错误: %w", err)
	}

	return nil
}

// Close 关闭连接
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// LocalPortForward 本地端口转发
func (c *Client) LocalPortForward(localAddr, remoteAddr string) error {
	if c.conn == nil {
		return fmt.Errorf("SSH连接未建立")
	}

	// 监听本地端口
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return fmt.Errorf("监听本地端口失败: %w", err)
	}
	defer listener.Close()

	fmt.Printf("本地端口转发已启动: %s -> %s\n", localAddr, remoteAddr)
	fmt.Println("按 Ctrl+C 停止转发")

	for {
		localConn, err := listener.Accept()
		if err != nil {
			// 检查是否是因为listener关闭导致的错误
			if strings.Contains(err.Error(), "use of closed network connection") {
				fmt.Println("端口转发监听器已关闭")
				return nil
			}
			return fmt.Errorf("接受连接失败: %w", err)
		}

		go func() {
			defer localConn.Close()

			// 连接到远程地址
			remoteConn, err := c.conn.Dial("tcp", remoteAddr)
			if err != nil {
				fmt.Printf("连接远程地址失败: %v\n", err)
				return
			}
			defer remoteConn.Close()

			// 双向数据转发
			go io.Copy(localConn, remoteConn)
			io.Copy(remoteConn, localConn)
		}()
	}
}

// RemotePortForward 远程端口转发
func (c *Client) RemotePortForward(remoteAddr, localAddr string) error {
	if c.conn == nil {
		return fmt.Errorf("SSH连接未建立")
	}

	// 监听远程端口
	listener, err := c.conn.Listen("tcp", remoteAddr)
	if err != nil {
		return fmt.Errorf("监听远程端口失败: %w", err)
	}
	defer listener.Close()

	fmt.Printf("远程端口转发已启动: %s -> %s\n", remoteAddr, localAddr)
	fmt.Println("按 Ctrl+C 停止转发")

	for {
		remoteConn, err := listener.Accept()
		if err != nil {
			// 检查是否是因为listener关闭导致的错误
			if strings.Contains(err.Error(), "use of closed network connection") {
				fmt.Println("端口转发监听器已关闭")
				return nil
			}
			return fmt.Errorf("接受远程连接失败: %w", err)
		}

		go func() {
			defer remoteConn.Close()

			// 连接到本地地址
			localConn, err := net.Dial("tcp", localAddr)
			if err != nil {
				fmt.Printf("连接本地地址失败: %v\n", err)
				return
			}
			defer localConn.Close()

			// 双向数据转发
			go io.Copy(remoteConn, localConn)
			io.Copy(localConn, remoteConn)
		}()
	}
}

// IsConnected 检查连接状态
func (c *Client) IsConnected() bool {
	if c.conn == nil {
		return false
	}

	// 尝试创建一个会话来测试连接
	session, err := c.conn.NewSession()
	if err != nil {
		return false
	}
	session.Close()

	return true
}

// SendKeepAlive 发送Keep-alive消息
func (c *Client) SendKeepAlive() error {
	if c.conn == nil {
		return fmt.Errorf("SSH连接未建立")
	}

	// 发送全局请求作为Keep-alive
	_, _, err := c.conn.SendRequest("keepalive@openssh.com", true, nil)
	if err != nil {
		// 如果不支持这个请求，尝试发送标准的Keep-alive
		_, _, err = c.conn.SendRequest("keepalive", false, nil)
	}
	
	return err
}

// StartKeepAlive 开始Keep-alive心跳
func (c *Client) StartKeepAlive(interval time.Duration) {
	if c.conn == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := c.SendKeepAlive(); err != nil {
				// Keep-alive失败，可能连接已断开
				fmt.Printf("Keep-alive失败: %v\n", err)
				return
			}
		}
	}()
}

// LocalPortForwardWithTimeout 本地端口转发（带超时）
func (c *Client) LocalPortForwardWithTimeout(localAddr, remoteAddr string, timeout time.Duration) error {
	if c.conn == nil {
		return fmt.Errorf("SSH连接未建立")
	}

	// 启动Keep-alive
	c.StartKeepAlive(30 * time.Second)

	// 监听本地端口
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return fmt.Errorf("监听本地端口失败: %w", err)
	}
	defer listener.Close()

	fmt.Printf("本地端口转发已启动: %s -> %s\n", localAddr, remoteAddr)
	fmt.Println("按 Ctrl+C 停止转发")

	for {
		// 设置Accept超时
		if tcpListener, ok := listener.(*net.TCPListener); ok {
			tcpListener.SetDeadline(time.Now().Add(timeout))
		}

		localConn, err := listener.Accept()
		if err != nil {
			// 检查是否是超时错误
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 超时，检查连接状态
				if !c.IsConnected() {
					return fmt.Errorf("SSH连接已断开")
				}
				continue
			}

			// 检查是否是因为listener关闭导致的错误
			if strings.Contains(err.Error(), "use of closed network connection") {
				fmt.Println("端口转发监听器已关闭")
				return nil
			}
			return fmt.Errorf("接受连接失败: %w", err)
		}

		go func() {
			defer localConn.Close()

			// 连接到远程地址
			remoteConn, err := c.conn.Dial("tcp", remoteAddr)
			if err != nil {
				fmt.Printf("连接远程地址失败: %v\n", err)
				return
			}
			defer remoteConn.Close()

			// 双向数据转发
			go io.Copy(localConn, remoteConn)
			io.Copy(remoteConn, localConn)
		}()
	}
}

// RemotePortForwardWithTimeout 远程端口转发（带超时）
func (c *Client) RemotePortForwardWithTimeout(remoteAddr, localAddr string, timeout time.Duration) error {
	if c.conn == nil {
		return fmt.Errorf("SSH连接未建立")
	}

	// 启动Keep-alive
	c.StartKeepAlive(30 * time.Second)

	// 监听远程端口
	listener, err := c.conn.Listen("tcp", remoteAddr)
	if err != nil {
		return fmt.Errorf("监听远程端口失败: %w", err)
	}
	defer listener.Close()

	fmt.Printf("远程端口转发已启动: %s -> %s\n", remoteAddr, localAddr)
	fmt.Println("按 Ctrl+C 停止转发")

	for {
		remoteConn, err := listener.Accept()
		if err != nil {
			// 检查是否是因为listener关闭导致的错误
			if strings.Contains(err.Error(), "use of closed network connection") {
				fmt.Println("端口转发监听器已关闭")
				return nil
			}
			return fmt.Errorf("接受远程连接失败: %w", err)
		}

		go func() {
			defer remoteConn.Close()

			// 连接到本地地址
			localConn, err := net.DialTimeout("tcp", localAddr, timeout)
			if err != nil {
				fmt.Printf("连接本地地址失败: %v\n", err)
				return
			}
			defer localConn.Close()

			// 双向数据转发
			go io.Copy(remoteConn, localConn)
			io.Copy(localConn, remoteConn)
		}()
	}
} 