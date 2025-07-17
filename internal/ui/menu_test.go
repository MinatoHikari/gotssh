package ui

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gotssh/internal/config"
	"gotssh/internal/forward"
)

// 测试辅助函数
func createTestUIManager(t *testing.T) (*Menu, *config.Manager, *forward.Manager) {
	tempDir := t.TempDir()
	configPath := tempDir + "/test-config.yaml"

	configManager, err := config.NewManager(configPath)
	require.NoError(t, err)

	forwardManager := forward.NewManager(configManager)
	menu := NewMenu(configManager, forwardManager)

	return menu, configManager, forwardManager
}

func createTestServer(t *testing.T, configManager *config.Manager) *config.ServerConfig {
	server := config.NewServerConfig("test.example.com")
	server.Alias = "test-server"
	server.User = "testuser"
	server.Port = 22
	server.AuthType = config.AuthTypePassword
	server.Password = "testpass"
	server.Description = "Test server"

	err := configManager.AddServer(server)
	require.NoError(t, err)

	return server
}

func createTestCredential(t *testing.T, configManager *config.Manager) *config.CredentialConfig {
	cred := config.NewCredentialConfig()
	cred.Alias = "test-cred"
	cred.Username = "testuser"
	cred.Type = config.CredentialTypePassword
	cred.Password = "testpass"
	cred.Description = "Test credential"

	err := configManager.AddCredential(cred)
	require.NoError(t, err)

	return cred
}

func createTestPortForward(t *testing.T, configManager *config.Manager, serverID string) *config.PortForwardConfig {
	pf := config.NewPortForwardConfig(serverID)
	pf.Alias = "test-forward"
	pf.LocalHost = "127.0.0.1"
	pf.LocalPort = 8080
	pf.RemoteHost = "127.0.0.1"
	pf.RemotePort = 80
	pf.Type = config.ForwardTypeLocal
	pf.Description = "Test port forward"

	err := configManager.AddPortForward(pf)
	require.NoError(t, err)

	return pf
}

// TestNewMenu 测试菜单创建
func TestNewMenu(t *testing.T) {
	t.Run("创建菜单实例", func(t *testing.T) {
		menu, configManager, forwardManager := createTestUIManager(t)

		assert.NotNil(t, menu)
		assert.Equal(t, configManager, menu.configManager)
		assert.Equal(t, forwardManager, menu.forwardManager)
	})
}

// TestMenuStructure 测试菜单结构
func TestMenuStructure(t *testing.T) {
	t.Run("菜单字段验证", func(t *testing.T) {
		menu, configManager, forwardManager := createTestUIManager(t)

		// 验证菜单字段
		assert.NotNil(t, menu.configManager)
		assert.NotNil(t, menu.forwardManager)

		// 验证字段类型
		assert.IsType(t, &config.Manager{}, menu.configManager)
		assert.IsType(t, &forward.Manager{}, menu.forwardManager)

		// 验证实例是否正确
		assert.Same(t, configManager, menu.configManager)
		assert.Same(t, forwardManager, menu.forwardManager)
	})
}

// TestShowServerList 测试显示服务器列表功能
func TestShowServerList(t *testing.T) {
	t.Run("空服务器列表", func(t *testing.T) {
		menu, _, _ := createTestUIManager(t)

		// 测试显示空列表（这个方法主要是显示逻辑，不返回错误）
		assert.NotPanics(t, func() {
			menu.ShowServerList()
		})
	})

	t.Run("有服务器的列表", func(t *testing.T) {
		menu, configManager, _ := createTestUIManager(t)

		// 添加测试服务器
		createTestServer(t, configManager)

		// 测试显示有内容的列表
		assert.NotPanics(t, func() {
			menu.ShowServerList()
		})
	})
}

// TestShowCredentialList 测试显示凭证列表功能
func TestShowCredentialList(t *testing.T) {
	t.Run("空凭证列表", func(t *testing.T) {
		_, configManager, _ := createTestUIManager(t)
		credentialMenu := NewCredentialMenu(configManager)

		// 测试显示空列表
		assert.NotPanics(t, func() {
			credentialMenu.ShowCredentialList()
		})
	})

	t.Run("有凭证的列表", func(t *testing.T) {
		_, configManager, _ := createTestUIManager(t)
		credentialMenu := NewCredentialMenu(configManager)

		// 添加测试凭证
		createTestCredential(t, configManager)

		// 测试显示有内容的列表
		assert.NotPanics(t, func() {
			credentialMenu.ShowCredentialList()
		})
	})
}

// TestShowPortForwardList 测试显示端口转发列表功能
func TestShowPortForwardList(t *testing.T) {
	t.Run("空端口转发列表", func(t *testing.T) {
		menu, _, _ := createTestUIManager(t)

		// 测试显示空列表
		assert.NotPanics(t, func() {
			menu.ShowPortForwardList()
		})
	})

	t.Run("有端口转发的列表", func(t *testing.T) {
		menu, configManager, _ := createTestUIManager(t)

		// 添加测试服务器和端口转发
		server := createTestServer(t, configManager)
		createTestPortForward(t, configManager, server.ID)

		// 测试显示有内容的列表
		assert.NotPanics(t, func() {
			menu.ShowPortForwardList()
		})
	})
}

// TestDataValidation 测试数据验证
func TestDataValidation(t *testing.T) {
	t.Run("验证服务器配置", func(t *testing.T) {
		menu, configManager, _ := createTestUIManager(t)

		// 创建测试服务器
		server := createTestServer(t, configManager)

		// 验证服务器配置
		assert.Equal(t, "test.example.com", server.Host)
		assert.Equal(t, 22, server.Port)
		assert.Equal(t, "testuser", server.User)
		assert.Equal(t, config.AuthTypePassword, server.AuthType)
		assert.Equal(t, "test-server", server.Alias)

		// 验证服务器存在于配置中
		servers := menu.configManager.ListServers()
		assert.Len(t, servers, 1)
		assert.Equal(t, server.ID, servers[0].ID)
	})

	t.Run("验证凭证配置", func(t *testing.T) {
		menu, configManager, _ := createTestUIManager(t)

		// 创建测试凭证
		cred := createTestCredential(t, configManager)

		// 验证凭证配置
		assert.Equal(t, "test-cred", cred.Alias)
		assert.Equal(t, "testuser", cred.Username)
		assert.Equal(t, config.CredentialTypePassword, cred.Type)
		assert.Equal(t, "testpass", cred.Password)

		// 验证凭证存在于配置中
		credentials := menu.configManager.ListCredentials()
		assert.Len(t, credentials, 1)
		assert.Equal(t, cred.ID, credentials[0].ID)
	})

	t.Run("验证端口转发配置", func(t *testing.T) {
		menu, configManager, _ := createTestUIManager(t)

		// 创建测试服务器和端口转发
		server := createTestServer(t, configManager)
		pf := createTestPortForward(t, configManager, server.ID)

		// 验证端口转发配置
		assert.Equal(t, "test-forward", pf.Alias)
		assert.Equal(t, server.ID, pf.ServerID)
		assert.Equal(t, "127.0.0.1", pf.LocalHost)
		assert.Equal(t, 8080, pf.LocalPort)
		assert.Equal(t, "127.0.0.1", pf.RemoteHost)
		assert.Equal(t, 80, pf.RemotePort)
		assert.Equal(t, config.ForwardTypeLocal, pf.Type)

		// 验证端口转发存在于配置中
		portForwards := menu.configManager.ListPortForwards()
		assert.Len(t, portForwards, 1)
		assert.Equal(t, pf.ID, portForwards[0].ID)
	})
}

// TestManagerIntegration 测试管理器集成
func TestManagerIntegration(t *testing.T) {
	t.Run("配置管理器集成", func(t *testing.T) {
		menu, configManager, _ := createTestUIManager(t)

		// 验证配置管理器功能
		assert.NotNil(t, menu.configManager)
		assert.Same(t, configManager, menu.configManager)

		// 测试配置管理器操作
		servers := menu.configManager.ListServers()
		assert.NotNil(t, servers)
		assert.Empty(t, servers)

		credentials := menu.configManager.ListCredentials()
		assert.NotNil(t, credentials)
		assert.Empty(t, credentials)

		portForwards := menu.configManager.ListPortForwards()
		assert.NotNil(t, portForwards)
		assert.Empty(t, portForwards)
	})

	t.Run("端口转发管理器集成", func(t *testing.T) {
		menu, _, forwardManager := createTestUIManager(t)

		// 验证端口转发管理器功能
		assert.NotNil(t, menu.forwardManager)
		assert.Same(t, forwardManager, menu.forwardManager)

		// 测试端口转发管理器操作
		activeForwards := menu.forwardManager.ListActiveForwards()
		assert.NotNil(t, activeForwards)
		assert.Empty(t, activeForwards)

		// 测试检查转发状态
		active := menu.forwardManager.IsForwardActive("non-existent")
		assert.False(t, active)
	})
}

// TestMenuStateManagement 测试菜单状态管理
func TestMenuStateManagement(t *testing.T) {
	t.Run("菜单状态一致性", func(t *testing.T) {
		menu, configManager, forwardManager := createTestUIManager(t)

		// 验证初始状态
		assert.Empty(t, menu.configManager.ListServers())
		assert.Empty(t, menu.configManager.ListCredentials())
		assert.Empty(t, menu.configManager.ListPortForwards())
		assert.Empty(t, menu.forwardManager.ListActiveForwards())

		// 添加数据
		server := createTestServer(t, configManager)
		cred := createTestCredential(t, configManager)
		pf := createTestPortForward(t, configManager, server.ID)

		// 使用forwardManager避免未使用变量的警告
		_ = forwardManager

		// 验证数据存在
		assert.Len(t, menu.configManager.ListServers(), 1)
		assert.Len(t, menu.configManager.ListCredentials(), 1)
		assert.Len(t, menu.configManager.ListPortForwards(), 1)

		// 验证数据内容
		servers := menu.configManager.ListServers()
		assert.Equal(t, server.ID, servers[0].ID)

		credentials := menu.configManager.ListCredentials()
		assert.Equal(t, cred.ID, credentials[0].ID)

		portForwards := menu.configManager.ListPortForwards()
		assert.Equal(t, pf.ID, portForwards[0].ID)
	})
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	t.Run("空配置管理器", func(t *testing.T) {
		// 测试使用nil配置管理器创建菜单
		menu := NewMenu(nil, nil)

		assert.NotNil(t, menu)
		assert.Nil(t, menu.configManager)
		assert.Nil(t, menu.forwardManager)

		// 尝试显示列表应该不会panic（但可能会出错）
		assert.NotPanics(t, func() {
			menu.ShowServerList()
		})
	})
}

// TestConfigurationPersistence 测试配置持久化
func TestConfigurationPersistence(t *testing.T) {
	t.Run("配置保存和加载", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := tempDir + "/persistence-test.yaml"

		// 创建第一个管理器实例
		configManager1, err := config.NewManager(configPath)
		require.NoError(t, err)

		forwardManager1 := forward.NewManager(configManager1)
		menu1 := NewMenu(configManager1, forwardManager1)

		// 添加测试数据
		server := createTestServer(t, configManager1)
		cred := createTestCredential(t, configManager1)
		pf := createTestPortForward(t, configManager1, server.ID)

		// 验证数据存在
		assert.Len(t, menu1.configManager.ListServers(), 1)
		assert.Len(t, menu1.configManager.ListCredentials(), 1)
		assert.Len(t, menu1.configManager.ListPortForwards(), 1)

		// 创建第二个管理器实例（加载相同配置）
		configManager2, err := config.NewManager(configPath)
		require.NoError(t, err)

		forwardManager2 := forward.NewManager(configManager2)
		menu2 := NewMenu(configManager2, forwardManager2)

		// 验证数据被正确加载
		servers := menu2.configManager.ListServers()
		assert.Len(t, servers, 1)
		assert.Equal(t, server.Alias, servers[0].Alias)

		credentials := menu2.configManager.ListCredentials()
		assert.Len(t, credentials, 1)
		assert.Equal(t, cred.Alias, credentials[0].Alias)

		portForwards := menu2.configManager.ListPortForwards()
		assert.Len(t, portForwards, 1)
		assert.Equal(t, pf.Alias, portForwards[0].Alias)
	})
}

// BenchmarkMenuCreation 性能测试
func BenchmarkMenuCreation(b *testing.B) {
	tempDir := b.TempDir()
	configPath := tempDir + "/benchmark-config.yaml"

	configManager, err := config.NewManager(configPath)
	if err != nil {
		b.Fatal(err)
	}

	forwardManager := forward.NewManager(configManager)

	for i := 0; i < b.N; i++ {
		NewMenu(configManager, forwardManager)
	}
}

// BenchmarkShowServerList 性能测试
func BenchmarkShowServerList(b *testing.B) {
	tempDir := b.TempDir()
	configPath := tempDir + "/benchmark-config.yaml"

	configManager, err := config.NewManager(configPath)
	if err != nil {
		b.Fatal(err)
	}

	// 添加一些测试服务器
	for i := 0; i < 10; i++ {
		server := config.NewServerConfig(fmt.Sprintf("test%d.example.com", i))
		server.Alias = fmt.Sprintf("test-server-%d", i)
		configManager.AddServer(server)
	}

	forwardManager := forward.NewManager(configManager)
	menu := NewMenu(configManager, forwardManager)

	for i := 0; i < b.N; i++ {
		menu.ShowServerList()
	}
}
