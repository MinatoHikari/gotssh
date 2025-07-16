package forward

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gotssh/internal/config"
	"gotssh/internal/ssh"
)

// Manager 端口转发管理器
type Manager struct {
	configManager *config.Manager
	activeForwards map[string]*ActiveForward
	// 新增超时配置
	ConnectTimeout    time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	KeepAliveInterval time.Duration
	MaxRetries        int
}

// ActiveForward 活动的端口转发
type ActiveForward struct {
	ID        string
	Config    *config.PortForwardConfig
	SSHClient *ssh.Client
	Done      chan bool
	// 新增控制通道
	ctx       context.Context
	cancel    context.CancelFunc
	errChan   chan error
	retryCount int
}

// NewManager 创建新的端口转发管理器
func NewManager(configManager *config.Manager) *Manager {
	return &Manager{
		configManager:     configManager,
		activeForwards:    make(map[string]*ActiveForward),
		// 默认超时配置
		ConnectTimeout:    30 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		KeepAliveInterval: 30 * time.Second,
		MaxRetries:        3,
	}
}

// SetTimeouts 设置超时配置
func (m *Manager) SetTimeouts(connectTimeout, readTimeout, writeTimeout, keepAliveInterval time.Duration, maxRetries int) {
	m.ConnectTimeout = connectTimeout
	m.ReadTimeout = readTimeout
	m.WriteTimeout = writeTimeout
	m.KeepAliveInterval = keepAliveInterval
	m.MaxRetries = maxRetries
}

// StartPortForward 启动端口转发
func (m *Manager) StartPortForward(pfConfig *config.PortForwardConfig) error {
	// 检查是否已经存在同样的端口转发
	if _, exists := m.activeForwards[pfConfig.ID]; exists {
		return fmt.Errorf("端口转发 %s 已经在运行", pfConfig.ID)
	}

	// 创建上下文和取消函数
	ctx, cancel := context.WithCancel(context.Background())
	
	// 创建活动转发记录
	forward := &ActiveForward{
		ID:        pfConfig.ID,
		Config:    pfConfig,
		Done:      make(chan bool),
		ctx:       ctx,
		cancel:    cancel,
		errChan:   make(chan error, 1),
		retryCount: 0,
	}

	// 启动端口转发协程
	go m.runPortForwardWithRetry(forward)
	
	// 启动连接监控协程
	go m.monitorConnection(forward)

	// 存储活动转发
	m.activeForwards[pfConfig.ID] = forward

	return nil
}

// runPortForwardWithRetry 运行端口转发，带重试机制
func (m *Manager) runPortForwardWithRetry(forward *ActiveForward) {
	defer func() {
		if forward.SSHClient != nil {
			forward.SSHClient.Close()
		}
		delete(m.activeForwards, forward.ID)
		close(forward.Done)
	}()

	for forward.retryCount <= m.MaxRetries {
		select {
		case <-forward.ctx.Done():
			fmt.Printf("端口转发 %s 已取消\n", forward.ID)
			return
		default:
		}

		// 获取服务器配置
		serverConfig, err := m.configManager.GetServer(forward.Config.ServerID)
		if err != nil {
			forward.errChan <- fmt.Errorf("获取服务器配置失败: %w", err)
			return
		}

		// 创建SSH客户端
		client := ssh.NewClient(serverConfig, m.configManager)
		
		// 设置连接超时
		if err := m.connectWithTimeout(client, forward.ctx); err != nil {
			forward.retryCount++
			fmt.Printf("端口转发 %s 连接失败 (重试 %d/%d): %v\n", 
				forward.ID, forward.retryCount, m.MaxRetries, err)
			
			if forward.retryCount <= m.MaxRetries {
				// 等待一段时间后重试
				select {
				case <-forward.ctx.Done():
					return
				case <-time.After(time.Duration(forward.retryCount*2) * time.Second):
					continue
				}
			}
			
			forward.errChan <- fmt.Errorf("连接失败，已达到最大重试次数: %w", err)
			return
		}

		forward.SSHClient = client
		forward.retryCount = 0 // 重置重试计数

		// 启动端口转发
		localAddr := fmt.Sprintf("%s:%d", forward.Config.LocalHost, forward.Config.LocalPort)
		remoteAddr := fmt.Sprintf("%s:%d", forward.Config.RemoteHost, forward.Config.RemotePort)

		fmt.Printf("端口转发 %s 已启动: %s -> %s\n", forward.ID, localAddr, remoteAddr)

		var forwardErr error
		if forward.Config.Type == config.ForwardTypeLocal {
			forwardErr = m.localPortForwardWithContext(client, forward.ctx, localAddr, remoteAddr)
		} else {
			forwardErr = m.remotePortForwardWithContext(client, forward.ctx, remoteAddr, localAddr)
		}

		if forwardErr != nil {
			fmt.Printf("端口转发 %s 错误: %v\n", forward.ID, forwardErr)
			
			// 检查是否是网络错误，如果是则重试
			if isNetworkError(forwardErr) && forward.retryCount < m.MaxRetries {
				forward.retryCount++
				fmt.Printf("网络错误，准备重试 %d/%d\n", forward.retryCount, m.MaxRetries)
				
				// 关闭当前连接
				if forward.SSHClient != nil {
					forward.SSHClient.Close()
					forward.SSHClient = nil
				}
				
				// 等待后重试
				select {
				case <-forward.ctx.Done():
					return
				case <-time.After(time.Duration(forward.retryCount*2) * time.Second):
					continue
				}
			} else {
				forward.errChan <- forwardErr
				return
			}
		}

		// 如果到达这里，说明转发正常结束
		break
	}
}

// connectWithTimeout 带超时的连接
func (m *Manager) connectWithTimeout(client *ssh.Client, ctx context.Context) error {
	connChan := make(chan error, 1)
	
	go func() {
		connChan <- client.Connect()
	}()

	select {
	case err := <-connChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(m.ConnectTimeout):
		return fmt.Errorf("连接超时")
	}
}

// monitorConnection 监控连接状态
func (m *Manager) monitorConnection(forward *ActiveForward) {
	ticker := time.NewTicker(m.KeepAliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-forward.ctx.Done():
			return
		case <-ticker.C:
			if forward.SSHClient == nil {
				continue
			}
			
			// 检查连接状态
			if !forward.SSHClient.IsConnected() {
				fmt.Printf("端口转发 %s 连接已断开，触发重连\n", forward.ID)
				
				// 发送错误信号触发重连
				select {
				case forward.errChan <- fmt.Errorf("连接断开"):
				default:
				}
			}
		case err := <-forward.errChan:
			fmt.Printf("端口转发 %s 监控到错误: %v\n", forward.ID, err)
		}
	}
}

// isNetworkError 检查是否是网络错误
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	networkErrors := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"network is unreachable",
		"no route to host",
		"broken pipe",
		"connection lost",
		"EOF",
	}
	
	for _, netErr := range networkErrors {
		if strings.Contains(errStr, netErr) {
			return true
		}
	}
	
	return false
}

// localPortForwardWithContext 本地端口转发（带Context）
func (m *Manager) localPortForwardWithContext(client *ssh.Client, ctx context.Context, localAddr, remoteAddr string) error {
	errChan := make(chan error, 1)
	
	go func() {
		errChan <- client.LocalPortForwardWithTimeout(localAddr, remoteAddr, m.ReadTimeout)
	}()
	
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// remotePortForwardWithContext 远程端口转发（带Context）
func (m *Manager) remotePortForwardWithContext(client *ssh.Client, ctx context.Context, remoteAddr, localAddr string) error {
	errChan := make(chan error, 1)
	
	go func() {
		errChan <- client.RemotePortForwardWithTimeout(remoteAddr, localAddr, m.ReadTimeout)
	}()
	
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// StopPortForward 停止端口转发
func (m *Manager) StopPortForward(pfID string) error {
	forward, exists := m.activeForwards[pfID]
	if !exists {
		return fmt.Errorf("端口转发 %s 不存在或未运行", pfID)
	}

	// 取消上下文，这会停止所有相关的goroutine
	forward.cancel()

	// 关闭SSH连接
	if forward.SSHClient != nil {
		if err := forward.SSHClient.Close(); err != nil {
			fmt.Printf("关闭SSH连接时出现错误: %v\n", err)
		}
	}

	// 等待转发停止（带超时）
	select {
	case <-forward.Done:
		// 转发已停止
		fmt.Printf("端口转发 %s 已停止\n", pfID)
	case <-time.After(5 * time.Second):
		// 超时，强制清理
		fmt.Printf("等待端口转发 %s 停止超时，强制清理\n", pfID)
		delete(m.activeForwards, pfID)
	}

	return nil
}

// StopAllPortForwards 停止所有端口转发
func (m *Manager) StopAllPortForwards() error {
	for pfID := range m.activeForwards {
		if err := m.StopPortForward(pfID); err != nil {
			fmt.Printf("停止端口转发 %s 失败: %v\n", pfID, err)
		}
	}
	return nil
}

// ListActiveForwards 列出所有活动的端口转发
func (m *Manager) ListActiveForwards() []*ActiveForward {
	var forwards []*ActiveForward
	for _, forward := range m.activeForwards {
		forwards = append(forwards, forward)
	}
	return forwards
}

// IsForwardActive 检查端口转发是否活动
func (m *Manager) IsForwardActive(pfID string) bool {
	_, exists := m.activeForwards[pfID]
	return exists
}

// StartPortForwardByAlias 根据别名启动端口转发
func (m *Manager) StartPortForwardByAlias(alias string) error {
	pfConfig, err := m.configManager.GetPortForwardByAlias(alias)
	if err != nil {
		return fmt.Errorf("获取端口转发配置失败: %w", err)
	}

	return m.StartPortForward(pfConfig)
}

// StartPortForwardWithSignalHandler 启动端口转发并处理信号
func (m *Manager) StartPortForwardWithSignalHandler(pfConfig *config.PortForwardConfig) error {
	// 启动端口转发
	if err := m.StartPortForward(pfConfig); err != nil {
		return err
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号或转发结束
	forward := m.activeForwards[pfConfig.ID]
	select {
	case sig := <-sigChan:
		fmt.Printf("\n接收到停止信号 (%v)，正在关闭端口转发...\n", sig)
		if err := m.StopPortForward(pfConfig.ID); err != nil {
			fmt.Printf("停止端口转发失败: %v\n", err)
		}
		fmt.Println("端口转发已停止")
		// 程序正常退出
		os.Exit(0)
	case <-forward.Done:
		fmt.Println("端口转发已结束")
	}

	return nil
}

// StartPortForwardByAliasWithSignalHandler 根据别名启动端口转发并处理信号
func (m *Manager) StartPortForwardByAliasWithSignalHandler(alias string) error {
	pfConfig, err := m.configManager.GetPortForwardByAlias(alias)
	if err != nil {
		return fmt.Errorf("获取端口转发配置失败: %w", err)
	}

	return m.StartPortForwardWithSignalHandler(pfConfig)
}

// GetActiveForward 获取活动的端口转发
func (m *Manager) GetActiveForward(pfID string) (*ActiveForward, error) {
	forward, exists := m.activeForwards[pfID]
	if !exists {
		return nil, fmt.Errorf("端口转发 %s 不存在或未运行", pfID)
	}
	return forward, nil
}

// GetForwardStatus 获取端口转发状态
func (m *Manager) GetForwardStatus(pfID string) string {
	if m.IsForwardActive(pfID) {
		return "运行中"
	}
	return "已停止"
}

// TestPortForward 测试端口转发配置
func (m *Manager) TestPortForward(pfConfig *config.PortForwardConfig) error {
	// 获取服务器配置
	serverConfig, err := m.configManager.GetServer(pfConfig.ServerID)
	if err != nil {
		return fmt.Errorf("获取服务器配置失败: %w", err)
	}

	// 创建SSH客户端进行测试
	client := ssh.NewClient(serverConfig, m.configManager)
	if err := client.Connect(); err != nil {
		return fmt.Errorf("连接SSH服务器失败: %w", err)
	}
	defer client.Close()

	// 测试连接是否正常
	if !client.IsConnected() {
		return fmt.Errorf("SSH连接不稳定")
	}

	fmt.Printf("端口转发配置测试成功: %s -> %s\n", 
		fmt.Sprintf("%s:%d", pfConfig.LocalHost, pfConfig.LocalPort),
		fmt.Sprintf("%s:%d", pfConfig.RemoteHost, pfConfig.RemotePort))

	return nil
}

// CreateAndStartPortForward 创建并启动端口转发
func (m *Manager) CreateAndStartPortForward(pfConfig *config.PortForwardConfig) error {
	// 保存配置
	if err := m.configManager.AddPortForward(pfConfig); err != nil {
		return fmt.Errorf("保存端口转发配置失败: %w", err)
	}

	// 启动端口转发
	if err := m.StartPortForward(pfConfig); err != nil {
		return fmt.Errorf("启动端口转发失败: %w", err)
	}

	return nil
} 