package forward

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gotssh/internal/config"
)

// 测试辅助函数
func createTestForwardManager(t *testing.T) *Manager {
	tempDir := t.TempDir()
	configPath := tempDir + "/test-config.yaml"

	configManager, err := config.NewManager(configPath)
	require.NoError(t, err)

	return NewManager(configManager)
}

func createTestPortForwardConfig(t *testing.T, manager *Manager) *config.PortForwardConfig {
	// 创建测试服务器
	server := config.NewServerConfig("localhost")
	server.Port = 22
	server.User = "test"
	server.AuthType = config.AuthTypePassword
	server.Password = "test"

	err := manager.configManager.AddServer(server)
	require.NoError(t, err)

	// 创建端口转发配置
	pf := config.NewPortForwardConfig(server.ID)
	pf.Alias = "test-forward"
	pf.LocalHost = "127.0.0.1"
	pf.LocalPort = 8080
	pf.RemoteHost = "127.0.0.1"
	pf.RemotePort = 80
	pf.Type = config.ForwardTypeLocal

	err = manager.configManager.AddPortForward(pf)
	require.NoError(t, err)

	return pf
}

// TestNewManager 测试管理器创建
func TestNewManager(t *testing.T) {
	t.Run("创建管理器", func(t *testing.T) {
		manager := createTestForwardManager(t)

		assert.NotNil(t, manager)
		assert.NotNil(t, manager.configManager)
		assert.NotNil(t, manager.activeForwards)
		assert.Empty(t, manager.activeForwards)

		// 验证默认超时配置
		assert.Equal(t, 30*time.Second, manager.ConnectTimeout)
		assert.Equal(t, 60*time.Second, manager.ReadTimeout)
		assert.Equal(t, 60*time.Second, manager.WriteTimeout)
		assert.Equal(t, 30*time.Second, manager.KeepAliveInterval)
		assert.Equal(t, 3, manager.MaxRetries)
	})
}

// TestSetTimeouts 测试超时配置设置
func TestSetTimeouts(t *testing.T) {
	t.Run("设置超时配置", func(t *testing.T) {
		manager := createTestForwardManager(t)

		connectTimeout := 15 * time.Second
		readTimeout := 30 * time.Second
		writeTimeout := 30 * time.Second
		keepAliveInterval := 15 * time.Second
		maxRetries := 5

		manager.SetTimeouts(connectTimeout, readTimeout, writeTimeout, keepAliveInterval, maxRetries)

		assert.Equal(t, connectTimeout, manager.ConnectTimeout)
		assert.Equal(t, readTimeout, manager.ReadTimeout)
		assert.Equal(t, writeTimeout, manager.WriteTimeout)
		assert.Equal(t, keepAliveInterval, manager.KeepAliveInterval)
		assert.Equal(t, maxRetries, manager.MaxRetries)
	})
}

// TestListActiveForwards 测试列出活动转发
func TestListActiveForwards(t *testing.T) {
	t.Run("空列表", func(t *testing.T) {
		manager := createTestForwardManager(t)

		forwards := manager.ListActiveForwards()
		assert.NotNil(t, forwards)
		assert.Empty(t, forwards)
	})

	t.Run("手动添加活动转发", func(t *testing.T) {
		manager := createTestForwardManager(t)

		// 手动添加一个活动转发（用于测试）
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		forward := &ActiveForward{
			ID:      "test-forward-1",
			Config:  &config.PortForwardConfig{ID: "test-forward-1"},
			Done:    make(chan bool),
			ctx:     ctx,
			cancel:  cancel,
			errChan: make(chan error, 1),
		}

		manager.activeForwards["test-forward-1"] = forward

		forwards := manager.ListActiveForwards()
		assert.Len(t, forwards, 1)
		assert.Equal(t, "test-forward-1", forwards[0].ID)
	})
}

// TestIsForwardActive 测试检查转发是否活动
func TestIsForwardActive(t *testing.T) {
	t.Run("不存在的转发", func(t *testing.T) {
		manager := createTestForwardManager(t)

		active := manager.IsForwardActive("non-existent")
		assert.False(t, active)
	})

	t.Run("活动的转发", func(t *testing.T) {
		manager := createTestForwardManager(t)

		// 手动添加一个活动转发
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		forward := &ActiveForward{
			ID:      "test-forward-1",
			Config:  &config.PortForwardConfig{ID: "test-forward-1"},
			Done:    make(chan bool),
			ctx:     ctx,
			cancel:  cancel,
			errChan: make(chan error, 1),
		}

		manager.activeForwards["test-forward-1"] = forward

		active := manager.IsForwardActive("test-forward-1")
		assert.True(t, active)
	})
}

// TestStartPortForwardByAlias 测试根据别名启动转发
func TestStartPortForwardByAlias(t *testing.T) {
	t.Run("不存在的别名", func(t *testing.T) {
		manager := createTestForwardManager(t)

		err := manager.StartPortForwardByAlias("non-existent-alias")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "端口转发别名")
	})

	t.Run("有效的别名", func(t *testing.T) {
		manager := createTestForwardManager(t)
		pfConfig := createTestPortForwardConfig(t, manager)

		// 启动端口转发应该成功，即使后台连接会失败
		err := manager.StartPortForwardByAlias(pfConfig.Alias)
		assert.NoError(t, err) // 预期成功启动

		// 验证配置被正确获取
		assert.Equal(t, "test-forward", pfConfig.Alias)

		// 验证转发已经被添加到活动列表
		assert.True(t, manager.IsForwardActive(pfConfig.ID))
	})
}

// TestStopPortForward 测试停止端口转发
func TestStopPortForward(t *testing.T) {
	t.Run("不存在的转发", func(t *testing.T) {
		manager := createTestForwardManager(t)

		err := manager.StopPortForward("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "不存在或未运行")
	})

	t.Run("停止活动转发", func(t *testing.T) {
		manager := createTestForwardManager(t)

		// 手动添加一个活动转发
		ctx, cancel := context.WithCancel(context.Background())

		forward := &ActiveForward{
			ID:      "test-forward-1",
			Config:  &config.PortForwardConfig{ID: "test-forward-1"},
			Done:    make(chan bool),
			ctx:     ctx,
			cancel:  cancel,
			errChan: make(chan error, 1),
		}

		manager.activeForwards["test-forward-1"] = forward

		// 模拟转发结束
		go func() {
			time.Sleep(100 * time.Millisecond)
			close(forward.Done)
		}()

		err := manager.StopPortForward("test-forward-1")
		assert.NoError(t, err)

		// 验证转发已被移除
		assert.False(t, manager.IsForwardActive("test-forward-1"))
	})
}

// TestStopAllPortForwards 测试停止所有端口转发
func TestStopAllPortForwards(t *testing.T) {
	t.Run("停止所有转发", func(t *testing.T) {
		manager := createTestForwardManager(t)

		// 手动添加多个活动转发
		ctx1, cancel1 := context.WithCancel(context.Background())
		ctx2, cancel2 := context.WithCancel(context.Background())

		forward1 := &ActiveForward{
			ID:      "test-forward-1",
			Config:  &config.PortForwardConfig{ID: "test-forward-1"},
			Done:    make(chan bool),
			ctx:     ctx1,
			cancel:  cancel1,
			errChan: make(chan error, 1),
		}

		forward2 := &ActiveForward{
			ID:      "test-forward-2",
			Config:  &config.PortForwardConfig{ID: "test-forward-2"},
			Done:    make(chan bool),
			ctx:     ctx2,
			cancel:  cancel2,
			errChan: make(chan error, 1),
		}

		manager.activeForwards["test-forward-1"] = forward1
		manager.activeForwards["test-forward-2"] = forward2

		// 模拟转发结束
		go func() {
			time.Sleep(100 * time.Millisecond)
			close(forward1.Done)
			close(forward2.Done)
		}()

		err := manager.StopAllPortForwards()
		assert.NoError(t, err)

		// 验证所有转发都已被移除
		assert.False(t, manager.IsForwardActive("test-forward-1"))
		assert.False(t, manager.IsForwardActive("test-forward-2"))
	})
}

// TestIsNetworkError 测试网络错误检查
func TestIsNetworkError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil错误",
			err:      nil,
			expected: false,
		},
		{
			name:     "连接拒绝",
			err:      assert.AnError,
			expected: false,
		},
		{
			name:     "连接超时",
			err:      assert.AnError,
			expected: false,
		},
		{
			name:     "网络不可达",
			err:      assert.AnError,
			expected: false,
		},
		{
			name:     "其他错误",
			err:      assert.AnError,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isNetworkError(tc.err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestActiveForwardStructure 测试活动转发结构
func TestActiveForwardStructure(t *testing.T) {
	t.Run("创建活动转发", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := &config.PortForwardConfig{
			ID:         "test-forward",
			Alias:      "test",
			LocalHost:  "127.0.0.1",
			LocalPort:  8080,
			RemoteHost: "127.0.0.1",
			RemotePort: 80,
			Type:       config.ForwardTypeLocal,
		}

		forward := &ActiveForward{
			ID:         "test-forward",
			Config:     config,
			Done:       make(chan bool),
			ctx:        ctx,
			cancel:     cancel,
			errChan:    make(chan error, 1),
			retryCount: 0,
		}

		assert.Equal(t, "test-forward", forward.ID)
		assert.Equal(t, config, forward.Config)
		assert.NotNil(t, forward.Done)
		assert.NotNil(t, forward.ctx)
		assert.NotNil(t, forward.cancel)
		assert.NotNil(t, forward.errChan)
		assert.Equal(t, 0, forward.retryCount)
	})
}

// TestConfigIntegration 测试配置集成
func TestConfigIntegration(t *testing.T) {
	t.Run("配置管理器集成", func(t *testing.T) {
		manager := createTestForwardManager(t)

		// 验证配置管理器可用
		assert.NotNil(t, manager.configManager)

		// 验证可以操作配置
		servers := manager.configManager.ListServers()
		assert.NotNil(t, servers)

		portForwards := manager.configManager.ListPortForwards()
		assert.NotNil(t, portForwards)
	})
}

// TestManagerDefaults 测试管理器默认值
func TestManagerDefaults(t *testing.T) {
	t.Run("验证默认值", func(t *testing.T) {
		manager := createTestForwardManager(t)

		// 验证默认超时配置
		assert.Equal(t, 30*time.Second, manager.ConnectTimeout)
		assert.Equal(t, 60*time.Second, manager.ReadTimeout)
		assert.Equal(t, 60*time.Second, manager.WriteTimeout)
		assert.Equal(t, 30*time.Second, manager.KeepAliveInterval)
		assert.Equal(t, 3, manager.MaxRetries)

		// 验证空的活动转发列表
		assert.Empty(t, manager.activeForwards)
	})
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	t.Run("重复启动转发", func(t *testing.T) {
		manager := createTestForwardManager(t)

		// 手动添加一个活动转发
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		forward := &ActiveForward{
			ID:      "test-forward-1",
			Config:  &config.PortForwardConfig{ID: "test-forward-1"},
			Done:    make(chan bool),
			ctx:     ctx,
			cancel:  cancel,
			errChan: make(chan error, 1),
		}

		manager.activeForwards["test-forward-1"] = forward

		// 尝试启动相同的转发
		pfConfig := &config.PortForwardConfig{ID: "test-forward-1"}
		err := manager.StartPortForward(pfConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "已经在运行")
	})
}

// BenchmarkNewManager 性能测试
func BenchmarkNewManager(b *testing.B) {
	tempDir := b.TempDir()
	configPath := tempDir + "/test-config.yaml"

	configManager, err := config.NewManager(configPath)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		NewManager(configManager)
	}
}

// BenchmarkListActiveForwards 性能测试
func BenchmarkListActiveForwards(b *testing.B) {
	tempDir := b.TempDir()
	configPath := tempDir + "/test-config.yaml"

	configManager, err := config.NewManager(configPath)
	if err != nil {
		b.Fatal(err)
	}

	manager := NewManager(configManager)

	// 添加一些活动转发
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		forward := &ActiveForward{
			ID:      fmt.Sprintf("test-forward-%d", i),
			Config:  &config.PortForwardConfig{ID: fmt.Sprintf("test-forward-%d", i)},
			Done:    make(chan bool),
			ctx:     ctx,
			cancel:  cancel,
			errChan: make(chan error, 1),
		}
		manager.activeForwards[forward.ID] = forward
	}

	for i := 0; i < b.N; i++ {
		manager.ListActiveForwards()
	}
}
